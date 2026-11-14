import 'dart:async';
import 'dart:convert';
import 'dart:math' as math;

import 'package:flutter/foundation.dart';
import 'package:just_audio/just_audio.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'api.dart';
import 'models.dart';

enum Repeat { off, all, one }

/// User settings for playback.
class PlayPrefs {
  String quality = 'orig'; // orig | 320 | 192 | 128
  String norm = 'auto'; // auto | track | album | off
  double target = -14; // LUFS

  static const _k = 'play_prefs';

  Future<void> load() async {
    final p = await SharedPreferences.getInstance();
    try {
      final j = jsonDecode(p.getString(_k) ?? '{}') as Map<String, dynamic>;
      quality = j['quality'] as String? ?? quality;
      norm = j['norm'] as String? ?? norm;
      target = (j['target'] as num?)?.toDouble() ?? target;
    } catch (_) {}
  }

  Future<void> save() async {
    final p = await SharedPreferences.getInstance();
    await p.setString(_k, jsonEncode({'quality': quality, 'norm': norm, 'target': target}));
  }
}

/// The playback engine. Wraps a just_audio player (ExoPlayer on Android). The media
/// notification / lock screen / foreground service is handled by HakoAudioHandler (audio_handler.dart).
///
/// The queue is kept in play order: shuffle reorders [queue] itself (like the web app), so
/// "play next" always means the track after the current one.
class PlayerController extends ChangeNotifier {
  PlayerController._();
  static final PlayerController instance = PlayerController._();

  final _enhancer = AndroidLoudnessEnhancer();
  late final AudioPlayer audio = AudioPlayer(
    // ExoPlayer sends the session cookie itself (no local proxy needed)
    useProxyForRequestHeaders: false,
    audioPipeline: AudioPipeline(androidAudioEffects: [_enhancer]),
  );

  final prefs = PlayPrefs();

  /// tracks in the same order as the player's audio sources
  List<Track> queue = [];
  List<Track>? _orig; // order before shuffling
  bool shuffle = false;
  Repeat repeat = Repeat.off;
  final Set<int> _forceTx = {}; // tracks the device couldn't decode directly
  final Set<int> _directIds = {}; // tracks currently loaded as original files

  int? get index => audio.currentIndex;
  Track? get current {
    final i = index;
    return i != null && i >= 0 && i < queue.length ? queue[i] : null;
  }

  bool get canSeek {
    final t = current;
    return t != null && _directIds.contains(t.id);
  }

  int _retries = 0;
  Timer? _retryTimer;
  int? _lastIndex;
  bool _scrobbled = false;
  Duration _listened = Duration.zero;
  Duration _lastPos = Duration.zero;
  Timer? _saveTimer;

  Future<void> init() async {
    await prefs.load();
    await _enhancer.setEnabled(false);
    audio.currentIndexStream.listen((i) {
      if (i != _lastIndex) {
        _lastIndex = i;
        _scrobbled = false;
        _listened = Duration.zero;
        _lastPos = Duration.zero;
        _applyGain();
        _saveSoon();
        notifyListeners();
      }
    });
    audio.positionStream.listen(_onPosition);
    audio.playbackEventStream.listen((_) {}, onError: (Object e, StackTrace st) => _onError(e));
    _saveTimer = Timer.periodic(const Duration(seconds: 15), (_) => _save());
  }

  @override
  void dispose() {
    _saveTimer?.cancel();
    super.dispose();
  }

  // ---------------- sources ----------------

  bool _canDirect(Track t) {
    if (_forceTx.contains(t.id)) return false;
    final c = t.codec.toLowerCase();
    final e = t.ext.toLowerCase().replaceAll('.', '');
    final ok = c == 'mp3' || c == 'flac' || c == 'aac' || c == 'vorbis' || c == 'opus' || c.startsWith('pcm_') ||
        (c.isEmpty && const {'mp3', 'flac', 'ogg', 'opus', 'wav'}.contains(e));
    if (!ok) return false;
    if (prefs.quality != 'orig' && t.bitrate > (int.tryParse(prefs.quality) ?? 320) + 40) return false;
    return true;
  }

  AudioSource _source(Track t) {
    final direct = _canDirect(t);
    if (direct) {
      _directIds.add(t.id);
    } else {
      _directIds.remove(t.id);
    }
    final br = prefs.quality == 'orig' ? 320 : (int.tryParse(prefs.quality) ?? 320);
    return ProgressiveAudioSource(
      api.streamUri(t, direct: direct, bitrate: br),
      headers: api.authHeaders,
      duration: t.duration > 0 ? Duration(milliseconds: (t.duration * 1000).round()) : null,
      options: const ProgressiveAudioSourceOptions(
        androidExtractorOptions: AndroidExtractorOptions(constantBitrateSeekingEnabled: true),
      ),
      tag: t,
    );
  }

  Future<void> _load(int index, {Duration position = Duration.zero, bool play = true}) async {
    _retryTimer?.cancel();
    try {
      await audio.setAudioSources(queue.map(_source).toList(), initialIndex: index, initialPosition: position, preload: play);
      await _applyLoop();
      if (play) unawaited(audio.play());
    } catch (e) {
      // errors are also delivered through playbackEventStream → _onError
      debugPrint('load failed: $e');
    }
    _saveSoon();
    notifyListeners();
  }

  // ---------------- queue operations ----------------

  Future<void> playTracks(List<Track> tracks, {int index = 0, bool shuffle = false}) async {
    if (tracks.isEmpty) return;
    _retries = 0;
    index = index.clamp(0, tracks.length - 1);
    this.shuffle = shuffle;
    _orig = List.of(tracks);
    if (shuffle) {
      final first = index >= 0 ? tracks[index] : null;
      final rest = List.of(tracks)..removeAt(index);
      rest.shuffle();
      queue = first == null ? rest : [first, ...rest];
      index = 0;
    } else {
      queue = List.of(tracks);
    }
    await _load(index);
  }

  Future<void> shuffleAll(List<Track> tracks) async {
    if (tracks.isEmpty) return;
    await playTracks(tracks, index: math.Random().nextInt(tracks.length), shuffle: true);
  }

  Future<void> playNext(List<Track> tracks) async {
    if (tracks.isEmpty) return;
    if (queue.isEmpty) return playTracks(tracks);
    final at = (index ?? -1) + 1;
    queue.insertAll(at, tracks);
    _orig?.addAll(tracks);
    await audio.insertAudioSources(at, tracks.map(_source).toList());
    _saveSoon();
    notifyListeners();
  }

  Future<void> addToQueue(List<Track> tracks) async {
    if (tracks.isEmpty) return;
    if (queue.isEmpty) {
      queue = List.of(tracks);
      _orig = List.of(tracks);
      await _load(0, play: false);
      return;
    }
    queue.addAll(tracks);
    _orig?.addAll(tracks);
    await audio.addAudioSources(tracks.map(_source).toList());
    _saveSoon();
    notifyListeners();
  }

  Future<void> removeAt(int i) async {
    if (i < 0 || i >= queue.length) return;
    final t = queue.removeAt(i);
    _orig?.remove(t);
    if (queue.isEmpty) {
      await stop();
      return;
    }
    notifyListeners(); // update the UI right away (swipe-to-remove needs it in the same frame)
    await audio.removeAudioSourceAt(i);
    _saveSoon();
    notifyListeners();
  }

  Future<void> move(int from, int to) async {
    if (from == to || from < 0 || from >= queue.length || to < 0 || to >= queue.length) return;
    final t = queue.removeAt(from);
    queue.insert(to, t);
    notifyListeners();
    await audio.moveAudioSource(from, to);
    _saveSoon();
    notifyListeners();
  }

  Future<void> jumpTo(int i) async {
    if (i < 0 || i >= queue.length) return;
    await audio.seek(Duration.zero, index: i);
    if (!audio.playing) unawaited(audio.play());
  }

  Future<void> stop() async {
    queue = [];
    _orig = null;
    await audio.stop();
    await audio.clearAudioSources();
    _saveSoon();
    notifyListeners();
  }

  // ---------------- transport ----------------

  Future<void> togglePlay() async {
    if (queue.isEmpty) return;
    if (audio.playing) {
      _retryTimer?.cancel();
      await audio.pause();
    } else {
      if (audio.processingState == ProcessingState.idle) {
        await _load(index ?? 0, position: audio.position);
      } else {
        unawaited(audio.play());
      }
    }
  }

  Future<void> next() async {
    if (audio.hasNext) {
      await audio.seekToNext();
    } else if (queue.isNotEmpty && repeat == Repeat.one) {
      await audio.seek(Duration.zero, index: 0);
    }
  }

  Future<void> previous() async {
    if (audio.position > const Duration(seconds: 3) || !audio.hasPrevious) {
      await seek(Duration.zero);
    } else {
      await audio.seekToPrevious();
    }
  }

  Future<void> seek(Duration d) async {
    if (!canSeek && d != Duration.zero) return;
    await audio.seek(d);
  }

  Future<void> toggleShuffle() async {
    if (queue.length < 2) {
      shuffle = !shuffle;
      notifyListeners();
      return;
    }
    final i = index ?? 0;
    final cur = queue[i];
    // reorder everything except the playing track, without interrupting it
    List<Track> before, after;
    if (!shuffle) {
      _orig = List.of(queue);
      final rest = List.of(queue)..removeAt(i);
      rest.shuffle();
      before = [];
      after = rest;
    } else {
      final base = _orig ?? queue;
      final extra = queue.where((t) => !base.contains(t)).toList();
      final full = [...base, ...extra];
      final k = full.indexOf(cur);
      before = k > 0 ? full.sublist(0, k) : [];
      after = k >= 0 ? full.sublist(k + 1) : full.where((t) => t != cur).toList();
    }
    shuffle = !shuffle;
    if (i + 1 < queue.length) await audio.removeAudioSourceRange(i + 1, queue.length);
    if (i > 0) await audio.removeAudioSourceRange(0, i);
    queue = [...before, cur, ...after];
    if (before.isNotEmpty) await audio.insertAudioSources(0, before.map(_source).toList());
    if (after.isNotEmpty) await audio.addAudioSources(after.map(_source).toList());
    _saveSoon();
    notifyListeners();
  }

  Future<void> cycleRepeat() async {
    repeat = Repeat.values[(repeat.index + 1) % Repeat.values.length];
    await _applyLoop();
    _saveSoon();
    notifyListeners();
  }

  Future<void> _applyLoop() => audio.setLoopMode(switch (repeat) {
        Repeat.off => LoopMode.off,
        Repeat.all => LoopMode.all,
        Repeat.one => LoopMode.one,
      });

  /// re-create sources after settings change (quality), keeping position
  Future<void> reloadSources() async {
    if (queue.isEmpty) return;
    final wasPlaying = audio.playing;
    await _load(index ?? 0, position: audio.position, play: wasPlaying);
  }

  void markFavorite(int trackId, bool fav) {
    for (final t in queue) {
      if (t.id == trackId) t.fav = fav;
    }
    notifyListeners();
  }

  // ---------------- loudness normalisation ----------------

  ({double db, bool album})? normInfo(Track? t) {
    if (t == null || prefs.norm == 'off') return null;
    var useAlbum = prefs.norm == 'album';
    if (prefs.norm == 'auto') {
      final i = queue.indexOf(t);
      final p = i > 0 ? queue[i - 1] : null;
      final n = i >= 0 && i + 1 < queue.length ? queue[i + 1] : null;
      useAlbum = t.albumId > 0 && ((p != null && p.albumId == t.albumId) || (n != null && n.albumId == t.albumId));
    }
    final l = useAlbum && t.albumLufs != null ? t.albumLufs : t.lufs;
    if (l == null) return null;
    return (db: (prefs.target - l).clamp(-30.0, 12.0), album: useAlbum && t.albumLufs != null);
  }

  /// Below the target level: boost with Android's LoudnessEnhancer (it limits peaks itself).
  /// Above: attenuate with the player volume.
  Future<void> _applyGain() async {
    final n = normInfo(current);
    final db = n?.db ?? 0;
    try {
      if (db < 0) {
        await audio.setVolume(math.pow(10, db / 20).toDouble());
        await _enhancer.setEnabled(false);
      } else {
        await audio.setVolume(1);
        final boost = math.min(db, 9.0);
        await _enhancer.setTargetGain(boost);
        await _enhancer.setEnabled(boost > 0.1);
      }
    } catch (e) {
      debugPrint('gain: $e');
    }
  }

  Future<void> applyPrefs({bool reload = false}) async {
    await prefs.save();
    await _applyGain();
    if (reload) await reloadSources();
    notifyListeners();
  }

  // ---------------- play counts ----------------

  void _onPosition(Duration p) {
    final d = p - _lastPos;
    if (audio.playing && d > Duration.zero && d < const Duration(seconds: 2)) _listened += d;
    // replayed from the start (repeat one): count again
    if (p < const Duration(seconds: 2) && _lastPos > const Duration(seconds: 10)) {
      _scrobbled = false;
      _listened = Duration.zero;
    }
    _lastPos = p;
    final t = current;
    if (t == null || _scrobbled) return;
    final need = math.min(240.0, math.max(10.0, (t.duration > 0 ? t.duration : 60) * 0.5));
    if (_listened.inMilliseconds / 1000 >= need) {
      _scrobbled = true;
      t.plays++;
      api.scrobble(t.id).catchError((_) {});
    }
    if (audio.playing && _retries > 0 && _listened > const Duration(seconds: 30)) _retries = 0;
  }

  // ---------------- error recovery ----------------

  void _onError(Object e) {
    final t = current;
    if (t == null) return;
    debugPrint('playback error: $e');
    // the stream may have failed because the session expired: check once per burst of errors
    // (a 401 there signs the user out via api.onUnauthorized, which also stops the player)
    if (_retries == 0) api.fetchMe().then((_) {}, onError: (_) {});
    _retries++;
    if (_retries == 3 && _directIds.contains(t.id)) {
      _forceTx.add(t.id); // repeated failure on the original file: let the server convert it
    }
    if (_retries > 6) {
      _retries = 0;
      final i = index ?? 0;
      if (i + 1 < queue.length) _load(i + 1);
      return;
    }
    final pos = canSeek ? audio.position : Duration.zero;
    final delay = Duration(milliseconds: math.min(8000, 500 * (1 << (_retries - 1))));
    _retryTimer?.cancel();
    _retryTimer = Timer(delay, () {
      if (current == t) _load(index ?? 0, position: pos);
    });
  }

  // ---------------- persistence ----------------

  Timer? _saveDebounce;
  void _saveSoon() {
    _saveDebounce?.cancel();
    _saveDebounce = Timer(const Duration(seconds: 1), _save);
  }

  Future<void> _save() async {
    final p = await SharedPreferences.getInstance();
    await p.setString(
      'queue',
      jsonEncode({
        'ids': queue.map((t) => t.id).toList(),
        'orig': _orig?.map((t) => t.id).toList(),
        'idx': index ?? 0,
        'pos': audio.position.inMilliseconds,
        'shuffle': shuffle,
        'repeat': repeat.index,
      }),
    );
  }

  /// restore the last queue (paused) after the app was closed
  Future<void> restore() async {
    if (queue.isNotEmpty) return;
    try {
      final p = await SharedPreferences.getInstance();
      final s = p.getString('queue');
      if (s == null) return;
      final j = jsonDecode(s) as Map<String, dynamic>;
      final ids = (j['ids'] as List).map((e) => (e as num).toInt()).toList();
      if (ids.isEmpty) return;
      final tracks = await api.trackBatch(ids);
      if (tracks.isEmpty) return;
      final byId = {for (final t in tracks) t.id: t};
      queue = ids.map((id) => byId[id]).whereType<Track>().toList();
      final origIds = (j['orig'] as List?)?.map((e) => (e as num).toInt()).toList();
      _orig = origIds?.map((id) => byId[id]).whereType<Track>().toList();
      shuffle = j['shuffle'] == true;
      repeat = Repeat.values[((j['repeat'] as num?)?.toInt() ?? 0).clamp(0, 2)];
      final idx = ((j['idx'] as num?)?.toInt() ?? 0).clamp(0, queue.length - 1);
      final pos = Duration(milliseconds: (j['pos'] as num?)?.toInt() ?? 0);
      await _load(idx, position: _canDirect(queue[idx]) ? pos : Duration.zero, play: false);
    } catch (e) {
      debugPrint('restore failed: $e');
    }
  }
}

final player = PlayerController.instance;
