int _i(Object? v) => v is num ? v.toInt() : 0;
double _d(Object? v) => v is num ? v.toDouble() : 0;
double? _dn(Object? v) => v is num ? v.toDouble() : null;
String _s(Object? v) => v is String ? v : '';
bool _b(Object? v) => v == true;
List<T> _list<T>(Object? v, T Function(Map<String, dynamic>) f) =>
    v is List ? v.whereType<Map<String, dynamic>>().map(f).toList() : <T>[];

class User {
  final int id;
  final String name;
  final bool admin;
  final bool totp;
  User({required this.id, required this.name, required this.admin, required this.totp});
  factory User.fromJson(Map<String, dynamic> j) => User(id: _i(j['id']), name: _s(j['name']), admin: _b(j['admin']), totp: _b(j['totp']));
}

class Track {
  final int id;
  final String title, artist, album, albumArtist, codec, ext, mime, genre, quality;
  final int artistId, albumId, trackNo, discNo, year, bitrate, sampleRate, bitDepth, channels, size, cv;
  final double duration;
  final bool hasLyrics;
  bool fav;
  int plays;
  final double? lufs, albumLufs;

  Track({
    required this.id,
    required this.title,
    required this.artist,
    required this.album,
    required this.albumArtist,
    required this.codec,
    required this.ext,
    required this.mime,
    required this.genre,
    required this.quality,
    required this.artistId,
    required this.albumId,
    required this.trackNo,
    required this.discNo,
    required this.year,
    required this.bitrate,
    required this.sampleRate,
    required this.bitDepth,
    required this.channels,
    required this.size,
    required this.cv,
    required this.duration,
    required this.hasLyrics,
    required this.fav,
    required this.plays,
    required this.lufs,
    required this.albumLufs,
  });

  factory Track.fromJson(Map<String, dynamic> j) => Track(
        id: _i(j['id']),
        title: _s(j['title']),
        artist: _s(j['artist']),
        album: _s(j['album']),
        albumArtist: _s(j['album_artist']),
        codec: _s(j['codec']),
        ext: _s(j['ext']),
        mime: _s(j['mime']),
        genre: _s(j['genre']),
        quality: _s(j['q']),
        artistId: _i(j['artist_id']),
        albumId: _i(j['album_id']),
        trackNo: _i(j['track_no']),
        discNo: _i(j['disc_no']),
        year: _i(j['year']),
        bitrate: _i(j['bitrate']),
        sampleRate: _i(j['sample_rate']),
        bitDepth: _i(j['bit_depth']),
        channels: _i(j['channels']),
        size: _i(j['size']),
        cv: _i(j['cv']),
        duration: _d(j['duration']),
        hasLyrics: _b(j['has_lyrics']),
        fav: _b(j['fav']),
        plays: _i(j['plays']),
        lufs: _dn(j['lufs']),
        albumLufs: _dn(j['album_lufs']),
      );

  static List<Track> list(Object? v) => _list(v, Track.fromJson);

  /// e.g. "FLAC 24bit/96kHz" or "MP3 320kbps"
  String get spec {
    final c = codec.isEmpty ? ext.toUpperCase() : codec.toUpperCase();
    if (bitDepth > 0 && sampleRate > 0) return '$c ${bitDepth}bit/${_khz(sampleRate)}kHz';
    if (bitrate > 0) return '$c ${bitrate}kbps';
    return c;
  }
}

String _khz(int hz) {
  final k = hz / 1000;
  return k == k.roundToDouble() ? k.toInt().toString() : k.toStringAsFixed(1);
}

String fmtSpec(int bits, int rate) => bits > 0 && rate > 0 ? '$bits/${_khz(rate)}' : '';

class Album {
  final int id, artistId, year, trackCount, cv, maxBits, maxRate;
  final String title, artist, genre, quality;
  final double duration;
  bool fav;
  Album({
    required this.id,
    required this.artistId,
    required this.year,
    required this.trackCount,
    required this.cv,
    required this.maxBits,
    required this.maxRate,
    required this.title,
    required this.artist,
    required this.genre,
    required this.quality,
    required this.duration,
    required this.fav,
  });
  factory Album.fromJson(Map<String, dynamic> j) => Album(
        id: _i(j['id']),
        artistId: _i(j['artist_id']),
        year: _i(j['year']),
        trackCount: _i(j['track_count']),
        cv: _i(j['cv']),
        maxBits: _i(j['max_bits']),
        maxRate: _i(j['max_rate']),
        title: _s(j['title']),
        artist: _s(j['artist']),
        genre: _s(j['genre']),
        quality: _s(j['q']),
        duration: _d(j['duration']),
        fav: _b(j['fav']),
      );
  static List<Album> list(Object? v) => _list(v, Album.fromJson);
}

class Artist {
  final int id, albumCount, trackCount, coverAlbum, cv;
  final String name;
  bool fav;
  Artist({required this.id, required this.albumCount, required this.trackCount, required this.coverAlbum, required this.cv, required this.name, required this.fav});
  factory Artist.fromJson(Map<String, dynamic> j) => Artist(
        id: _i(j['id']),
        albumCount: _i(j['album_count']),
        trackCount: _i(j['track_count']),
        coverAlbum: _i(j['cover_album']),
        cv: _i(j['cv']),
        name: _s(j['name']),
        fav: _b(j['fav']),
      );
  static List<Artist> list(Object? v) => _list(v, Artist.fromJson);
}

class Playlist {
  final int id, trackCount, updated;
  final String name;
  final double duration;
  final List<int> covers;
  Playlist({required this.id, required this.trackCount, required this.updated, required this.name, required this.duration, required this.covers});
  factory Playlist.fromJson(Map<String, dynamic> j) => Playlist(
        id: _i(j['id']),
        trackCount: _i(j['track_count']),
        updated: _i(j['updated']),
        name: _s(j['name']),
        duration: _d(j['duration']),
        covers: j['covers'] is List ? (j['covers'] as List).map(_i).toList() : <int>[],
      );
  static List<Playlist> list(Object? v) => _list(v, Playlist.fromJson);
}

class HomeData {
  final List<Album> recent, played, random;
  final List<Track> top;
  HomeData(this.recent, this.played, this.random, this.top);
  factory HomeData.fromJson(Map<String, dynamic> j) =>
      HomeData(Album.list(j['recent']), Album.list(j['played']), Album.list(j['random']), Track.list(j['top']));
}

class SearchResult {
  final List<Artist> artists;
  final List<Album> albums;
  final List<Track> tracks;
  SearchResult(this.artists, this.albums, this.tracks);
  factory SearchResult.fromJson(Map<String, dynamic> j) => SearchResult(Artist.list(j['artists']), Album.list(j['albums']), Track.list(j['tracks']));
  bool get isEmpty => artists.isEmpty && albums.isEmpty && tracks.isEmpty;
}

class AlbumDetail {
  final Album album;
  final List<Track> tracks;
  AlbumDetail(this.album, this.tracks);
  factory AlbumDetail.fromJson(Map<String, dynamic> j) => AlbumDetail(Album.fromJson(j['album'] as Map<String, dynamic>), Track.list(j['tracks']));
}

class ArtistDetail {
  final Artist artist;
  final List<Album> albums, appearsOn;
  final List<Track> top;
  final List<int> trackIds;
  ArtistDetail(this.artist, this.albums, this.appearsOn, this.top, this.trackIds);
  factory ArtistDetail.fromJson(Map<String, dynamic> j) => ArtistDetail(
        Artist.fromJson(j['artist'] as Map<String, dynamic>),
        Album.list(j['albums']),
        Album.list(j['appears_on']),
        Track.list(j['top']),
        j['track_ids'] is List ? (j['track_ids'] as List).map(_i).toList() : <int>[],
      );
}

class PlaylistDetail {
  final Playlist playlist;
  final List<Track> tracks;
  PlaylistDetail(this.playlist, this.tracks);
  factory PlaylistDetail.fromJson(Map<String, dynamic> j) =>
      PlaylistDetail(Playlist.fromJson(j['playlist'] as Map<String, dynamic>), Track.list(j['tracks']));
}

class Favorites {
  final List<Track> tracks;
  final List<Album> albums;
  final List<Artist> artists;
  Favorites(this.tracks, this.albums, this.artists);
  factory Favorites.fromJson(Map<String, dynamic> j) => Favorites(Track.list(j['tracks']), Album.list(j['albums']), Artist.list(j['artists']));
}

class LyricLine {
  final double t;
  final String text;
  LyricLine(this.t, this.text);
}

class Lyrics {
  final bool synced;
  final List<LyricLine> lines;
  Lyrics(this.synced, this.lines);
  factory Lyrics.fromJson(Map<String, dynamic> j) =>
      Lyrics(_b(j['synced']), _list(j['lines'], (l) => LyricLine(_d(l['t']), _s(l['text']))));
}

String fmtDur(double s) {
  if (s.isNaN || s < 0) s = 0;
  final t = s.floor();
  final h = t ~/ 3600, m = (t % 3600) ~/ 60, sec = t % 60;
  final ss = sec.toString().padLeft(2, '0');
  return h > 0 ? '$h:${m.toString().padLeft(2, '0')}:$ss' : '$m:$ss';
}

String fmtLong(double s) {
  final m = (s / 60).round();
  if (m < 60) return '$m分';
  return '${m ~/ 60}時間${m % 60}分';
}
