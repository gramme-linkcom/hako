import 'package:flutter/material.dart';
import 'package:just_audio/just_audio.dart';

import '../api.dart';
import '../models.dart';
import '../player.dart';
import '../widgets.dart';

/// play/pause button that shows a spinner while buffering
class PlayPauseButton extends StatelessWidget {
  final double size;
  final bool filled;
  const PlayPauseButton({super.key, this.size = 32, this.filled = false});
  @override
  Widget build(BuildContext context) {
    return StreamBuilder<PlayerState>(
      stream: player.audio.playerStateStream,
      builder: (context, snap) {
        final s = snap.data;
        final playing = s?.playing ?? false;
        final buffering = playing && (s?.processingState == ProcessingState.loading || s?.processingState == ProcessingState.buffering);
        final icon = Icon(playing ? Icons.pause : Icons.play_arrow, size: size);
        final btn = filled
            ? IconButton.filled(
                iconSize: size,
                style: IconButton.styleFrom(backgroundColor: Colors.white, foregroundColor: Colors.black, minimumSize: Size(size * 2, size * 2)),
                onPressed: player.togglePlay,
                icon: icon,
              )
            : IconButton(iconSize: size, onPressed: player.togglePlay, icon: icon);
        if (!buffering) return btn;
        return Stack(alignment: Alignment.center, children: [
          btn,
          IgnorePointer(
            child: SizedBox(
              width: filled ? size * 2 : size + 12,
              height: filled ? size * 2 : size + 12,
              child: CircularProgressIndicator(strokeWidth: 2, color: filled ? accent : null),
            ),
          ),
        ]);
      },
    );
  }
}

void openNowPlaying(BuildContext context) {
  Navigator.of(context, rootNavigator: true).push(PageRouteBuilder(
    transitionDuration: const Duration(milliseconds: 280),
    reverseTransitionDuration: const Duration(milliseconds: 220),
    pageBuilder: (_, _, _) => const NowPlayingPage(),
    transitionsBuilder: (_, anim, _, child) => SlideTransition(
      position: Tween(begin: const Offset(0, 1), end: Offset.zero).chain(CurveTween(curve: Curves.easeOutCubic)).animate(anim),
      child: child,
    ),
  ));
}

class MiniPlayer extends StatelessWidget {
  const MiniPlayer({super.key});

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: player,
      builder: (context, _) {
        final t = player.current;
        if (t == null) return const SizedBox.shrink();
        return Material(
          color: surface,
          child: InkWell(
            onTap: () => openNowPlaying(context),
            child: Column(mainAxisSize: MainAxisSize.min, children: [
              _MiniProgress(t),
              Padding(
                padding: const EdgeInsets.fromLTRB(10, 6, 4, 6),
                child: Row(children: [
                  Cover(albumId: t.albumId, cv: t.cv, title: t.album, size: 44, radius: 5, width: 96),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                      Text(t.title, maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(fontWeight: FontWeight.w600)),
                      Text(t.artist, maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(color: muted, fontSize: 12)),
                    ]),
                  ),
                  const PlayPauseButton(size: 30),
                  IconButton(icon: const Icon(Icons.skip_next), onPressed: player.next),
                ]),
              ),
            ]),
          ),
        );
      },
    );
  }
}

class _MiniProgress extends StatelessWidget {
  final Track t;
  const _MiniProgress(this.t);
  @override
  Widget build(BuildContext context) {
    return StreamBuilder<Duration>(
      stream: player.audio.positionStream,
      builder: (context, snap) {
        final d = player.audio.duration?.inMilliseconds ?? (t.duration * 1000).round();
        final p = snap.data?.inMilliseconds ?? 0;
        return LinearProgressIndicator(value: d > 0 ? (p / d).clamp(0.0, 1.0) : 0, minHeight: 2, backgroundColor: Colors.white10);
      },
    );
  }
}

class NowPlayingPage extends StatefulWidget {
  const NowPlayingPage({super.key});
  @override
  State<NowPlayingPage> createState() => _NowPlayingPageState();
}

class _NowPlayingPageState extends State<NowPlayingPage> {
  bool _lyrics = false;
  double? _drag; // seconds while dragging the slider

  @override
  Widget build(BuildContext context) {
    return ListenableBuilder(
      listenable: player,
      builder: (context, _) {
        final t = player.current;
        if (t == null) {
          WidgetsBinding.instance.addPostFrameCallback((_) {
            if (mounted) Navigator.of(context).maybePop();
          });
          return const Scaffold();
        }
        final size = MediaQuery.sizeOf(context);
        final coverSize = (size.width - 48).clamp(120.0, size.height * 0.42);
        final norm = player.normInfo(t);
        return Scaffold(
          body: Container(
            decoration: BoxDecoration(
              gradient: LinearGradient(
                begin: Alignment.topCenter,
                end: Alignment.bottomCenter,
                colors: [accent.withValues(alpha: 0.22), bg, bg],
              ),
            ),
            child: SafeArea(
              child: GestureDetector(
                onVerticalDragEnd: (d) {
                  if ((d.primaryVelocity ?? 0) > 400) Navigator.of(context).pop();
                },
                child: Padding(
                  padding: const EdgeInsets.symmetric(horizontal: 24),
                  child: Column(children: [
                    Row(children: [
                      IconButton(icon: const Icon(Icons.keyboard_arrow_down, size: 30), onPressed: () => Navigator.of(context).pop()),
                      const Expanded(child: Text('再生中', textAlign: TextAlign.center, style: TextStyle(color: muted))),
                      IconButton(icon: const Icon(Icons.more_vert), onPressed: () => showTrackMenu(context, t)),
                    ]),
                    const Spacer(),
                    SizedBox(
                      height: coverSize,
                      child: _lyrics
                          ? LyricsView(key: ValueKey(t.id), track: t)
                          : Center(child: Cover(albumId: t.albumId, cv: t.cv, title: t.album, size: coverSize, radius: 12, width: 800)),
                    ),
                    const Spacer(),
                    Row(children: [
                      Expanded(
                        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                          Text(t.title, maxLines: 2, overflow: TextOverflow.ellipsis, style: const TextStyle(fontSize: 20, fontWeight: FontWeight.w800)),
                          const SizedBox(height: 2),
                          Text(t.artist, maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(color: muted, fontSize: 15)),
                        ]),
                      ),
                      IconButton(
                        icon: Icon(t.fav ? Icons.favorite : Icons.favorite_border, color: t.fav ? accent : null),
                        onPressed: () {
                          t.fav = !t.fav;
                          toggleFavorite('track', t.id, t.fav);
                        },
                      ),
                    ]),
                    const SizedBox(height: 6),
                    Row(children: [
                      QBadge(t.quality),
                      if (t.quality.isNotEmpty) const SizedBox(width: 6),
                      Flexible(
                        child: Text(
                          [t.spec, if (!player.canSeek) '変換再生', if (norm != null) '${norm.db >= 0 ? '+' : ''}${norm.db.toStringAsFixed(1)}dB${norm.album ? '（アルバム）' : ''}']
                              .join(' · '),
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(color: muted, fontSize: 11),
                        ),
                      ),
                    ]),
                    const SizedBox(height: 4),
                    _seekBar(t),
                    _controls(),
                    const SizedBox(height: 8),
                    Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
                      IconButton(
                        icon: Icon(Icons.lyrics_outlined, color: _lyrics ? accent : null),
                        tooltip: '歌詞',
                        onPressed: () => setState(() => _lyrics = !_lyrics),
                      ),
                      IconButton(
                        icon: const Icon(Icons.queue_music),
                        tooltip: 'キュー',
                        onPressed: () => showQueue(context),
                      ),
                    ]),
                    const SizedBox(height: 8),
                  ]),
                ),
              ),
            ),
          ),
        );
      },
    );
  }

  Widget _seekBar(Track t) {
    return StreamBuilder<Duration>(
      stream: player.audio.positionStream,
      builder: (context, snap) {
        final total = (player.audio.duration?.inMilliseconds ?? (t.duration * 1000).round()) / 1000;
        final pos = _drag ?? (snap.data?.inMilliseconds ?? 0) / 1000;
        final max = total > 0 ? total : 1.0;
        return Column(children: [
          SliderTheme(
            data: SliderTheme.of(context).copyWith(trackHeight: 3, thumbShape: const RoundSliderThumbShape(enabledThumbRadius: 6), overlayShape: SliderComponentShape.noOverlay),
            child: Slider(
              value: pos.clamp(0.0, max),
              max: max,
              onChanged: player.canSeek ? (v) => setState(() => _drag = v) : null,
              onChangeEnd: (v) {
                player.seek(Duration(milliseconds: (v * 1000).round()));
                setState(() => _drag = null);
              },
            ),
          ),
          const SizedBox(height: 6),
          Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
            Text(fmtDur(pos), style: const TextStyle(color: muted, fontSize: 12)),
            Text(fmtDur(total), style: const TextStyle(color: muted, fontSize: 12)),
          ]),
        ]);
      },
    );
  }

  Widget _controls() {
    final rep = player.repeat;
    return Row(mainAxisAlignment: MainAxisAlignment.spaceBetween, children: [
      IconButton(
        icon: Icon(Icons.shuffle, color: player.shuffle ? accent : null),
        onPressed: player.toggleShuffle,
      ),
      IconButton(iconSize: 38, icon: const Icon(Icons.skip_previous), onPressed: player.previous),
      const PlayPauseButton(size: 36, filled: true),
      IconButton(iconSize: 38, icon: const Icon(Icons.skip_next), onPressed: player.next),
      IconButton(
        icon: Icon(rep == Repeat.one ? Icons.repeat_one : Icons.repeat, color: rep == Repeat.off ? null : accent),
        onPressed: () {
          player.cycleRepeat();
          toast(switch (player.repeat) {
            Repeat.off => 'リピート: オフ',
            Repeat.all => 'リピート: すべて',
            Repeat.one => 'リピート: 1曲',
          });
        },
      ),
    ]);
  }
}

// ---------------- queue ----------------

void showQueue(BuildContext context) {
  showModalBottomSheet<void>(
    context: context,
    useRootNavigator: true,
    isScrollControlled: true,
    showDragHandle: true,
    builder: (ctx) => DraggableScrollableSheet(
      expand: false,
      initialChildSize: 0.75,
      maxChildSize: 0.95,
      builder: (ctx, scroll) => ListenableBuilder(
        listenable: player,
        builder: (ctx, _) {
          final q = player.queue;
          final cur = player.index ?? -1;
          return Column(children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 0, 8, 8),
              child: Row(children: [
                Expanded(child: Text('キュー（${q.length}曲）', style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w700))),
                TextButton(
                  onPressed: q.isEmpty ? null : () => addToPlaylistDialog(ctx, List.of(q)),
                  child: const Text('プレイリストに保存'),
                ),
              ]),
            ),
            Expanded(
              child: ReorderableListView.builder(
                scrollController: scroll,
                buildDefaultDragHandles: false,
                itemCount: q.length,
                onReorder: (from, to) {
                  if (to > from) to--;
                  player.move(from, to);
                },
                itemBuilder: (ctx, i) {
                  final t = q[i];
                  final isCur = i == cur;
                  return Dismissible(
                    key: ValueKey('q$i-${t.id}'), // same track may be queued twice
                    direction: isCur ? DismissDirection.none : DismissDirection.endToStart,
                    background: Container(color: Colors.red.withValues(alpha: 0.4), alignment: Alignment.centerRight, padding: const EdgeInsets.only(right: 20), child: const Icon(Icons.delete)),
                    onDismissed: (_) => player.removeAt(i),
                    child: ListTile(
                      tileColor: isCur ? accent.withValues(alpha: 0.12) : null,
                      leading: Cover(albumId: t.albumId, cv: t.cv, title: t.album, size: 40, radius: 4, width: 96),
                      title: Text(t.title, maxLines: 1, overflow: TextOverflow.ellipsis, style: TextStyle(color: isCur ? accent : (i < cur ? muted : null))),
                      subtitle: Text(t.artist, maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(color: muted, fontSize: 12)),
                      trailing: ReorderableDragStartListener(index: i, child: const Padding(padding: EdgeInsets.all(8), child: Icon(Icons.drag_handle))),
                      onTap: () => player.jumpTo(i),
                    ),
                  );
                },
              ),
            ),
          ]);
        },
      ),
    ),
  );
}

// ---------------- lyrics ----------------

class LyricsView extends StatefulWidget {
  final Track track;
  const LyricsView({super.key, required this.track});
  @override
  State<LyricsView> createState() => _LyricsViewState();
}

class _LyricsViewState extends State<LyricsView> {
  Lyrics? _l;
  String? _error;
  int _line = -1;
  final _keys = <int, GlobalKey>{};

  @override
  void initState() {
    super.initState();
    api.lyrics(widget.track.id).then((l) {
      if (mounted) setState(() => _l = l);
    }).catchError((Object e) {
      if (mounted) setState(() => _error = '$e');
    });
  }

  void _follow(Duration pos) {
    final l = _l;
    if (l == null || !l.synced) return;
    final s = pos.inMilliseconds / 1000 + 0.2;
    var i = -1;
    for (var k = 0; k < l.lines.length; k++) {
      if (l.lines[k].t <= s) {
        i = k;
      } else {
        break;
      }
    }
    if (i != _line) {
      _line = i;
      WidgetsBinding.instance.addPostFrameCallback((_) {
        final c = _keys[i]?.currentContext;
        if (c != null) Scrollable.ensureVisible(c, alignment: 0.4, duration: const Duration(milliseconds: 300));
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    final l = _l;
    if (_error != null) return Center(child: Text(_error!, style: const TextStyle(color: muted)));
    if (l == null) return const Center(child: CircularProgressIndicator());
    if (l.lines.isEmpty) return const Center(child: Text('歌詞はありません', style: TextStyle(color: muted)));
    return StreamBuilder<Duration>(
      stream: player.audio.positionStream,
      builder: (context, snap) {
        _follow(snap.data ?? Duration.zero);
        return ShaderMask(
          shaderCallback: (r) => const LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            colors: [Colors.transparent, Colors.white, Colors.white, Colors.transparent],
            stops: [0, 0.1, 0.9, 1],
          ).createShader(r),
          blendMode: BlendMode.dstIn,
          child: SingleChildScrollView(
            padding: const EdgeInsets.symmetric(vertical: 40),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              for (var i = 0; i < l.lines.length; i++)
                GestureDetector(
                  key: _keys.putIfAbsent(i, GlobalKey.new),
                  onTap: l.synced && player.canSeek ? () => player.seek(Duration(milliseconds: (l.lines[i].t * 1000).round())) : null,
                  child: Padding(
                    padding: const EdgeInsets.symmetric(vertical: 6),
                    child: Text(
                      l.lines[i].text.isEmpty ? '♪' : l.lines[i].text,
                      style: TextStyle(
                        fontSize: l.synced ? 20 : 16,
                        height: 1.35,
                        fontWeight: l.synced ? FontWeight.w700 : FontWeight.w400,
                        color: !l.synced || i == _line ? Colors.white : Colors.white38,
                      ),
                    ),
                  ),
                ),
            ]),
          ),
        );
      },
    );
  }
}
