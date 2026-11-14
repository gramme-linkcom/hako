import 'dart:async';
import 'dart:convert';

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:http/http.dart' as http;
import 'package:shared_preferences/shared_preferences.dart';

import 'models.dart';

class ApiException implements Exception {
  final int status;
  final String message;
  ApiException(this.status, this.message);
  @override
  String toString() => message;
}

/// Result of a password login: either signed in, or a second factor is needed.
class LoginResult {
  final User? user;
  final String? mfaTicket;
  LoginResult({this.user, this.mfaTicket});
}

/// Talks to the Hako server. Auth is the same session cookie the web app uses,
/// sent explicitly as a Cookie header (also for audio streams and cover images).
class Api {
  Api._();
  static final Api instance = Api._();

  static const _storage = FlutterSecureStorage();
  static const _kToken = 'hako_session';
  static const _kServer = 'server';
  static const cookieName = 'hako_session';

  final _client = http.Client();
  String server = '';
  String? _token;
  User? me;

  /// called when the server says the session is no longer valid
  void Function()? onUnauthorized;

  bool get signedIn => _token != null && server.isNotEmpty;

  Map<String, String> get authHeaders => _token == null ? const {} : {'Cookie': '$cookieName=$_token'};

  Future<void> load() async {
    final prefs = await SharedPreferences.getInstance();
    server = prefs.getString(_kServer) ?? '';
    try {
      _token = await _storage.read(key: _kToken);
    } catch (_) {
      _token = null;
    }
  }

  static String normalizeServer(String s) {
    s = s.trim();
    if (s.isEmpty) return s;
    if (!s.startsWith('http://') && !s.startsWith('https://')) s = 'https://$s';
    while (s.endsWith('/')) {
      s = s.substring(0, s.length - 1);
    }
    return s;
  }

  Uri url(String path, [Map<String, Object?>? query]) {
    final q = <String, String>{};
    query?.forEach((k, v) {
      if (v != null) q[k] = '$v';
    });
    final u = Uri.parse('$server/api$path');
    return q.isEmpty ? u : u.replace(queryParameters: q);
  }

  String coverUrl(int albumId, {int w = 300, int cv = 0}) => '$server/api/albums/$albumId/cover?w=$w&v=$cv';

  Future<dynamic> _send(String method, String path, {Map<String, Object?>? query, Object? body, bool authFail = true}) async {
    final req = http.Request(method, url(path, query));
    req.headers.addAll(authHeaders);
    req.headers['Accept'] = 'application/json';
    if (body != null) {
      req.headers['Content-Type'] = 'application/json';
      req.body = jsonEncode(body);
    }
    final http.Response res;
    try {
      res = await http.Response.fromStream(await _client.send(req).timeout(const Duration(seconds: 30)));
    } on TimeoutException {
      throw ApiException(0, 'サーバーに接続できません（タイムアウト）');
    } catch (e) {
      throw ApiException(0, 'サーバーに接続できません');
    }
    _captureCookie(res);
    dynamic data;
    if (res.bodyBytes.isNotEmpty) {
      try {
        data = jsonDecode(utf8.decode(res.bodyBytes));
      } catch (_) {
        data = null;
      }
    }
    if (res.statusCode >= 400) {
      final msg = data is Map && data['error'] is String ? data['error'] as String : 'エラー (${res.statusCode})';
      if (res.statusCode == 401 && authFail && _token != null) {
        await _clearToken();
        onUnauthorized?.call();
      }
      throw ApiException(res.statusCode, msg);
    }
    return data;
  }

  void _captureCookie(http.Response res) {
    final sc = res.headers['set-cookie'];
    if (sc == null) return;
    final m = RegExp('$cookieName=([^;,\\s]*)').firstMatch(sc);
    if (m == null) return;
    final v = m.group(1)!;
    if (v.isEmpty) {
      _clearToken();
    } else {
      _token = v;
      _storage.write(key: _kToken, value: v);
    }
  }

  Future<void> _clearToken() async {
    _token = null;
    me = null;
    try {
      await _storage.delete(key: _kToken);
    } catch (_) {}
  }

  Future<dynamic> get(String path, [Map<String, Object?>? query]) => _send('GET', path, query: query);
  Future<dynamic> post(String path, [Object? body]) => _send('POST', path, body: body ?? const {});
  Future<dynamic> put(String path, Object body) => _send('PUT', path, body: body);
  Future<dynamic> delete(String path) => _send('DELETE', path);

  // ---- auth ----

  Future<void> setServer(String s) async {
    server = normalizeServer(s);
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_kServer, server);
  }

  /// checks that the address is a Hako server; returns needs_setup
  Future<bool> checkServer() async {
    final d = await _send('GET', '/setup', authFail: false);
    if (d is! Map || !d.containsKey('needs_setup')) throw ApiException(0, 'Hako のサーバーではないようです');
    return d['needs_setup'] == true;
  }

  Future<LoginResult> login(String name, String password) async {
    final d = await _send('POST', '/login', body: {'name': name, 'password': password}, authFail: false);
    if (d is Map && d['mfa_required'] == true) return LoginResult(mfaTicket: d['ticket'] as String);
    me = User.fromJson(d as Map<String, dynamic>);
    return LoginResult(user: me);
  }

  Future<User> loginMfa(String ticket, String code) async {
    final d = await _send('POST', '/login/mfa', body: {'ticket': ticket, 'code': code}, authFail: false);
    me = User.fromJson(d as Map<String, dynamic>);
    return me!;
  }

  Future<User> fetchMe() async {
    final d = await get('/me');
    me = User.fromJson(d as Map<String, dynamic>);
    return me!;
  }

  Future<void> logout() async {
    try {
      await _send('POST', '/logout', authFail: false);
    } catch (_) {}
    await _clearToken();
  }

  // ---- library ----

  Future<HomeData> home() async => HomeData.fromJson(await get('/home') as Map<String, dynamic>);

  Future<SearchResult> search(String q) async => SearchResult.fromJson(await get('/search', {'q': q}) as Map<String, dynamic>);

  Future<List<Album>> albums({String sort = 'added', String? quality, bool fav = false, int? limit, int offset = 0}) async {
    final d = await get('/albums', {'sort': sort, 'quality': quality, 'fav': fav ? 1 : null, 'limit': limit, 'offset': offset});
    return Album.list(d);
  }

  Future<AlbumDetail> album(int id) async => AlbumDetail.fromJson(await get('/albums/$id') as Map<String, dynamic>);

  Future<List<Artist>> artists() async => Artist.list(await get('/artists'));

  Future<ArtistDetail> artist(int id) async => ArtistDetail.fromJson(await get('/artists/$id') as Map<String, dynamic>);

  Future<(List<Track>, int)> tracks({String sort = 'title', String? quality, bool fav = false, int limit = 200, int offset = 0}) async {
    final d = await get('/tracks', {'sort': sort, 'quality': quality, 'fav': fav ? 1 : null, 'limit': limit, 'offset': offset}) as Map<String, dynamic>;
    return (Track.list(d['tracks']), (d['total'] as num?)?.toInt() ?? 0);
  }

  Future<List<Track>> trackBatch(List<int> ids) async {
    final out = <Track>[];
    for (var i = 0; i < ids.length; i += 500) {
      final chunk = ids.sublist(i, i + 500 > ids.length ? ids.length : i + 500);
      out.addAll(Track.list(await get('/tracks/batch', {'ids': chunk.join(',')})));
    }
    return out;
  }

  Future<Lyrics> lyrics(int trackId) async => Lyrics.fromJson(await get('/tracks/$trackId/lyrics') as Map<String, dynamic>);

  Future<void> scrobble(int trackId) async => post('/tracks/$trackId/scrobble');

  Future<void> setFavorite(String kind, int id, bool fav) async => post('/favorites', {'kind': kind, 'id': id, 'fav': fav});

  Future<Favorites> favorites() async => Favorites.fromJson(await get('/favorites') as Map<String, dynamic>);

  Future<List<Playlist>> playlists() async => Playlist.list(await get('/playlists'));

  Future<PlaylistDetail> playlist(int id) async => PlaylistDetail.fromJson(await get('/playlists/$id') as Map<String, dynamic>);

  Future<int> createPlaylist(String name, [List<int> trackIds = const []]) async {
    final d = await post('/playlists', {'name': name, 'track_ids': trackIds}) as Map<String, dynamic>;
    return (d['id'] as num).toInt();
  }

  Future<void> appendToPlaylist(int id, List<int> trackIds) async => post('/playlists/$id/tracks', {'track_ids': trackIds});

  Future<void> updatePlaylist(int id, {String? name, List<int>? trackIds}) async =>
      put('/playlists/$id', {'name': ?name, 'track_ids': ?trackIds});

  Future<void> deletePlaylist(int id) async => delete('/playlists/$id');

  /// URL used by the player. Direct = original file (seekable); otherwise MP3 converted on the fly.
  Uri streamUri(Track t, {required bool direct, int bitrate = 320}) =>
      direct ? url('/tracks/${t.id}/stream', {'mode': 'direct'}) : url('/tracks/${t.id}/stream', {'br': bitrate, 'start': '0.00'});
}

final api = Api.instance;
