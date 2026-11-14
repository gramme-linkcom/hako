import 'package:audio_service/audio_service.dart';
import 'package:just_audio/just_audio.dart';

import 'api.dart';
import 'models.dart';
import 'player.dart';

/// Connects the player to Android's media session: the notification, lock screen,
/// Bluetooth / headset buttons and the foreground service that keeps playback alive.
///
/// The notification always shows the same three buttons: previous / play-pause / next
/// (no stop button, and "previous" stays available on the first track — it restarts it).
class HakoAudioHandler extends BaseAudioHandler with SeekHandler {
  HakoAudioHandler() {
    final a = player.audio;
    a.playbackEventStream.listen((_) => _broadcast(), onError: (Object _, StackTrace _) => _broadcast());
    a.playingStream.listen((_) => _broadcast());
    player.addListener(_onPlayerChange);
  }

  int? _itemFor; // track id currently published as mediaItem
  bool? _seekable;

  static MediaItem itemFor(Track t) => MediaItem(
        id: '${t.id}',
        title: t.title.isEmpty ? '(無題)' : t.title,
        artist: t.artist,
        album: t.album,
        duration: t.duration > 0 ? Duration(milliseconds: (t.duration * 1000).round()) : null,
        artUri: t.albumId > 0 ? Uri.parse(api.coverUrl(t.albumId, w: 500, cv: t.cv)) : null,
        artHeaders: api.authHeaders,
      );

  void _onPlayerChange() {
    final t = player.current;
    if (t == null) {
      if (_itemFor != null) {
        _itemFor = null;
        mediaItem.add(null);
      }
    } else if (t.id != _itemFor) {
      _itemFor = t.id;
      mediaItem.add(itemFor(t));
    }
    if (player.canSeek != _seekable) _broadcast();
  }

  void _broadcast() {
    final a = player.audio;
    final playing = a.playing;
    _seekable = player.canSeek;
    playbackState.add(PlaybackState(
      controls: [
        MediaControl.skipToPrevious,
        if (playing) MediaControl.pause else MediaControl.play,
        MediaControl.skipToNext,
      ],
      systemActions: {
        if (player.canSeek) MediaAction.seek,
        MediaAction.skipToPrevious,
        MediaAction.skipToNext,
        MediaAction.playPause,
      },
      androidCompactActionIndices: const [0, 1, 2],
      processingState: switch (a.processingState) {
        ProcessingState.idle => player.queue.isEmpty ? AudioProcessingState.idle : AudioProcessingState.ready,
        ProcessingState.loading => AudioProcessingState.loading,
        ProcessingState.buffering => AudioProcessingState.buffering,
        ProcessingState.ready => AudioProcessingState.ready,
        ProcessingState.completed => AudioProcessingState.completed,
      },
      playing: playing,
      updatePosition: a.position,
      bufferedPosition: a.bufferedPosition,
      speed: a.speed,
      queueIndex: a.currentIndex,
    ));
  }

  @override
  Future<void> play() async {
    if (!player.audio.playing) await player.togglePlay();
  }

  @override
  Future<void> pause() async {
    if (player.audio.playing) await player.togglePlay();
  }

  @override
  Future<void> stop() => pause();

  @override
  Future<void> seek(Duration position) => player.seek(position);

  @override
  Future<void> skipToNext() => player.next();

  @override
  Future<void> skipToPrevious() => player.previous();

  @override
  Future<void> skipToQueueItem(int index) => player.jumpTo(index);

  // swiping the app away from recents keeps the music playing (like other music apps);
  // when paused, the service stops by itself
  @override
  Future<void> onTaskRemoved() async {}
}
