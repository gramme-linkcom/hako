import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/material.dart';

import 'api.dart';
import 'models.dart';
import 'pages/album_page.dart';
import 'pages/artist_page.dart';
import 'player.dart';

const accent = Color(0xFFA78BFA);
const bg = Color(0xFF0E0E13);
const surface = Color(0xFF17171F);
const muted = Color(0xFF9A98A8);

final messengerKey = GlobalKey<ScaffoldMessengerState>();

void toast(String msg) {
  messengerKey.currentState
    ?..hideCurrentSnackBar()
    ..showSnackBar(SnackBar(content: Text(msg), behavior: SnackBarBehavior.floating, duration: const Duration(seconds: 2)));
}

/// placeholder colour derived from the title, so missing covers still look distinct
Color _tint(String s) {
  var h = 0;
  for (final c in s.codeUnits) {
    h = (h * 31 + c) & 0xffffff;
  }
  return HSLColor.fromAHSL(1, (h % 360).toDouble(), 0.35, 0.28).toColor();
}

class Cover extends StatelessWidget {
  final int albumId;
  final int cv;
  final String title;
  final double size;
  final double radius;
  final int width;
  const Cover({super.key, required this.albumId, this.cv = 0, this.title = '', this.size = 48, this.radius = 6, this.width = 200});

  @override
  Widget build(BuildContext context) {
    final ph = Container(
      color: _tint(title),
      alignment: Alignment.center,
      child: Text(title.isEmpty ? '♪' : title.characters.first.toUpperCase(),
          style: TextStyle(color: Colors.white70, fontSize: size * 0.4, fontWeight: FontWeight.w600)),
    );
    return ClipRRect(
      borderRadius: BorderRadius.circular(radius),
      child: SizedBox(
        width: size,
        height: size,
        child: albumId <= 0
            ? ph
            : CachedNetworkImage(
                imageUrl: api.coverUrl(albumId, w: width, cv: cv),
                httpHeaders: api.authHeaders,
                fit: BoxFit.cover,
                fadeInDuration: const Duration(milliseconds: 120),
                placeholder: (_, _) => ph,
                errorWidget: (_, _, _) => ph,
              ),
      ),
    );
  }
}

/// Hi-Res / Lossless badge
class QBadge extends StatelessWidget {
  final String q;
  final String spec;
  const QBadge(this.q, {super.key, this.spec = ''});
  @override
  Widget build(BuildContext context) {
    if (q != 'hires' && q != 'lossless') return const SizedBox.shrink();
    final hires = q == 'hires';
    final color = hires ? const Color(0xFFF5C451) : const Color(0xFF7DD3FC);
    final label = hires ? 'Hi-Res' : 'Lossless';
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1),
      decoration: BoxDecoration(border: Border.all(color: color.withValues(alpha: 0.8)), borderRadius: BorderRadius.circular(4)),
      child: Text(spec.isEmpty ? label : '$label $spec',
          style: TextStyle(color: color, fontSize: 10, fontWeight: FontWeight.w700, letterSpacing: 0.2)),
    );
  }
}

void openAlbum(BuildContext context, int id) {
  if (id <= 0) return;
  Navigator.of(context).push(MaterialPageRoute(builder: (_) => AlbumPage(id: id)));
}

void openArtist(BuildContext context, int id) {
  if (id <= 0) return;
  Navigator.of(context).push(MaterialPageRoute(builder: (_) => ArtistPage(id: id)));
}

Future<void> toggleFavorite(String kind, int id, bool fav) async {
  try {
    await api.setFavorite(kind, id, fav);
    if (kind == 'track') player.markFavorite(id, fav);
    toast(fav ? 'お気に入りに追加しました' : 'お気に入りから外しました');
  } catch (e) {
    toast('$e');
  }
}

Future<void> addToPlaylistDialog(BuildContext context, List<Track> tracks) async {
  if (tracks.isEmpty) return;
  List<Playlist> pls;
  try {
    pls = await api.playlists();
  } catch (e) {
    toast('$e');
    return;
  }
  if (!context.mounted) return;
  final choice = await showModalBottomSheet<Object>(
    context: context,
    useRootNavigator: true,
    showDragHandle: true,
    builder: (ctx) => SafeArea(
      child: ListView(shrinkWrap: true, children: [
        const Padding(padding: EdgeInsets.fromLTRB(20, 0, 20, 8), child: Text('プレイリストに追加', style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600))),
        ListTile(leading: const Icon(Icons.add), title: const Text('新しいプレイリスト'), onTap: () => Navigator.pop(ctx, 'new')),
        for (final p in pls)
          ListTile(leading: const Icon(Icons.queue_music), title: Text(p.name), subtitle: Text('${p.trackCount}曲'), onTap: () => Navigator.pop(ctx, p)),
      ]),
    ),
  );
  if (choice == null || !context.mounted) return;
  try {
    if (choice == 'new') {
      final name = await promptText(context, '新しいプレイリスト', hint: '名前');
      if (name == null || name.trim().isEmpty) return;
      await api.createPlaylist(name.trim(), tracks.map((t) => t.id).toList());
      toast('「${name.trim()}」を作成しました');
    } else if (choice is Playlist) {
      await api.appendToPlaylist(choice.id, tracks.map((t) => t.id).toList());
      toast('「${choice.name}」に追加しました');
    }
  } catch (e) {
    toast('$e');
  }
}

Future<String?> promptText(BuildContext context, String title, {String hint = '', String initial = ''}) {
  final c = TextEditingController(text: initial);
  return showDialog<String>(
    context: context,
    useRootNavigator: true,
    builder: (ctx) => AlertDialog(
      title: Text(title),
      content: TextField(controller: c, autofocus: true, decoration: InputDecoration(hintText: hint), onSubmitted: (v) => Navigator.pop(ctx, v)),
      actions: [
        TextButton(onPressed: () => Navigator.pop(ctx), child: const Text('キャンセル')),
        FilledButton(onPressed: () => Navigator.pop(ctx, c.text), child: const Text('OK')),
      ],
    ),
  );
}

Future<bool> confirm(BuildContext context, String msg, {String ok = 'OK'}) async {
  final r = await showDialog<bool>(
    context: context,
    useRootNavigator: true,
    builder: (ctx) => AlertDialog(
      content: Text(msg),
      actions: [
        TextButton(onPressed: () => Navigator.pop(ctx, false), child: const Text('キャンセル')),
        FilledButton(onPressed: () => Navigator.pop(ctx, true), child: Text(ok)),
      ],
    ),
  );
  return r == true;
}

/// long-press / ⋮ menu for a track
Future<void> showTrackMenu(BuildContext context, Track t, {VoidCallback? onRemove, String removeLabel = 'リストから削除'}) async {
  final nav = Navigator.of(context);
  await showModalBottomSheet<void>(
    context: context,
    useRootNavigator: true,
    showDragHandle: true,
    builder: (ctx) => SafeArea(
      child: SingleChildScrollView(
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          ListTile(
            leading: Cover(albumId: t.albumId, cv: t.cv, title: t.album, size: 44),
            title: Text(t.title, maxLines: 1, overflow: TextOverflow.ellipsis),
            subtitle: Text('${t.artist} · ${t.spec}', maxLines: 1, overflow: TextOverflow.ellipsis),
          ),
          const Divider(height: 1),
          ListTile(leading: const Icon(Icons.playlist_play), title: const Text('次に再生'), onTap: () {
            Navigator.pop(ctx);
            player.playNext([t]);
            toast('次に再生します');
          }),
          ListTile(leading: const Icon(Icons.queue_music), title: const Text('キューに追加'), onTap: () {
            Navigator.pop(ctx);
            player.addToQueue([t]);
            toast('キューに追加しました');
          }),
          ListTile(leading: const Icon(Icons.playlist_add), title: const Text('プレイリストに追加'), onTap: () {
            Navigator.pop(ctx);
            addToPlaylistDialog(context, [t]);
          }),
          ListTile(
            leading: Icon(t.fav ? Icons.favorite : Icons.favorite_border, color: t.fav ? accent : null),
            title: Text(t.fav ? 'お気に入りから外す' : 'お気に入りに追加'),
            onTap: () {
              Navigator.pop(ctx);
              t.fav = !t.fav;
              toggleFavorite('track', t.id, t.fav);
            },
          ),
          if (t.albumId > 0)
            ListTile(leading: const Icon(Icons.album), title: const Text('アルバムを開く'), onTap: () {
              Navigator.pop(ctx);
              nav.push(MaterialPageRoute(builder: (_) => AlbumPage(id: t.albumId)));
            }),
          if (t.artistId > 0)
            ListTile(leading: const Icon(Icons.person), title: const Text('アーティストを開く'), onTap: () {
              Navigator.pop(ctx);
              nav.push(MaterialPageRoute(builder: (_) => ArtistPage(id: t.artistId)));
            }),
          if (onRemove != null)
            ListTile(leading: const Icon(Icons.remove_circle_outline), title: Text(removeLabel), onTap: () {
              Navigator.pop(ctx);
              onRemove();
            }),
        ]),
      ),
    ),
  );
}

/// A row for a track. Tapping plays [list] starting at [index].
class TrackTile extends StatelessWidget {
  final List<Track> list;
  final int index;
  final bool showCover;
  final bool showNumber;
  final VoidCallback? onRemove;
  const TrackTile({super.key, required this.list, required this.index, this.showCover = true, this.showNumber = false, this.onRemove});

  @override
  Widget build(BuildContext context) {
    final t = list[index];
    return ListenableBuilder(
      listenable: player,
      builder: (context, _) {
        final playing = player.current?.id == t.id;
        return ListTile(
          contentPadding: const EdgeInsets.only(left: 16, right: 4),
          leading: showCover
              ? Cover(albumId: t.albumId, cv: t.cv, title: t.album, size: 46, radius: 5, width: 96)
              : SizedBox(
                  width: 28,
                  child: Center(
                    child: playing
                        ? const Icon(Icons.graphic_eq, color: accent, size: 20)
                        : Text(t.trackNo > 0 ? '${t.trackNo}' : '', style: const TextStyle(color: muted)),
                  ),
                ),
          title: Text(t.title.isEmpty ? '(無題)' : t.title,
              maxLines: 1, overflow: TextOverflow.ellipsis, style: TextStyle(color: playing ? accent : null, fontWeight: playing ? FontWeight.w600 : null)),
          subtitle: Row(children: [
            if (t.quality.isNotEmpty) ...[QBadge(t.quality), const SizedBox(width: 6)],
            Expanded(
              child: Text(showCover ? '${t.artist} · ${t.album}' : t.artist,
                  maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(color: muted, fontSize: 13)),
            ),
          ]),
          trailing: Row(mainAxisSize: MainAxisSize.min, children: [
            Text(fmtDur(t.duration), style: const TextStyle(color: muted, fontSize: 12)),
            IconButton(icon: const Icon(Icons.more_vert, size: 20), onPressed: () => showTrackMenu(context, t, onRemove: onRemove)),
          ]),
          onTap: () => player.playTracks(list, index: index),
          onLongPress: () => showTrackMenu(context, t, onRemove: onRemove),
        );
      },
    );
  }
}

class AlbumCard extends StatelessWidget {
  final Album album;
  final double size;
  const AlbumCard(this.album, {super.key, this.size = 150});
  @override
  Widget build(BuildContext context) {
    final a = album;
    return InkWell(
      borderRadius: BorderRadius.circular(8),
      onTap: () => openAlbum(context, a.id),
      child: SizedBox(
        width: size,
        child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
          Stack(children: [
            Cover(albumId: a.id, cv: a.cv, title: a.title, size: size, radius: 8, width: size > 160 ? 500 : 300),
            if (a.quality.isNotEmpty)
              Positioned(
                left: 6,
                bottom: 6,
                child: Container(
                  decoration: BoxDecoration(color: Colors.black.withValues(alpha: 0.65), borderRadius: BorderRadius.circular(4)),
                  child: QBadge(a.quality, spec: fmtSpec(a.maxBits, a.maxRate)),
                ),
              ),
          ]),
          const SizedBox(height: 6),
          Text(a.title, maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 13)),
          Text(a.year > 0 ? '${a.artist} · ${a.year}' : a.artist,
              maxLines: 1, overflow: TextOverflow.ellipsis, style: const TextStyle(color: muted, fontSize: 12)),
        ]),
      ),
    );
  }
}

class AlbumRow extends StatelessWidget {
  final String title;
  final List<Album> albums;
  const AlbumRow(this.title, this.albums, {super.key});
  @override
  Widget build(BuildContext context) {
    if (albums.isEmpty) return const SizedBox.shrink();
    return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
      SectionTitle(title),
      SizedBox(
        height: 196,
        child: ListView.separated(
          scrollDirection: Axis.horizontal,
          padding: const EdgeInsets.symmetric(horizontal: 16),
          itemCount: albums.length,
          separatorBuilder: (_, _) => const SizedBox(width: 12),
          itemBuilder: (_, i) => AlbumCard(albums[i]),
        ),
      ),
    ]);
  }
}

class SectionTitle extends StatelessWidget {
  final String text;
  final Widget? trailing;
  const SectionTitle(this.text, {super.key, this.trailing});
  @override
  Widget build(BuildContext context) => Padding(
        padding: const EdgeInsets.fromLTRB(16, 20, 8, 8),
        child: Row(children: [
          Expanded(child: Text(text, style: const TextStyle(fontSize: 18, fontWeight: FontWeight.w700))),
          ?trailing,
        ]),
      );
}

/// grid of albums that adapts the column count to the screen width
class AlbumGrid extends StatelessWidget {
  final List<Album> albums;
  const AlbumGrid(this.albums, {super.key});
  @override
  Widget build(BuildContext context) {
    return SliverPadding(
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 16),
      sliver: SliverLayoutBuilder(builder: (context, c) {
        final cols = (c.crossAxisExtent / 170).floor().clamp(2, 8);
        final w = (c.crossAxisExtent - (cols - 1) * 12) / cols;
        return SliverGrid(
          gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(crossAxisCount: cols, mainAxisSpacing: 14, crossAxisSpacing: 12, mainAxisExtent: w + 44),
          delegate: SliverChildBuilderDelegate((_, i) => AlbumCard(albums[i], size: w), childCount: albums.length),
        );
      }),
    );
  }
}

/// Loads data with [load] and shows a spinner / error with retry / the content.
class Loader<T> extends StatefulWidget {
  final Future<T> Function() load;
  final Widget Function(BuildContext context, T data, Future<void> Function() reload) builder;
  const Loader({super.key, required this.load, required this.builder});
  @override
  State<Loader<T>> createState() => _LoaderState<T>();
}

class _LoaderState<T> extends State<Loader<T>> {
  T? _data;
  Object? _error;
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _reload();
  }

  Future<void> _reload() async {
    setState(() {
      _loading = _data == null;
      _error = null;
    });
    try {
      final d = await widget.load();
      if (mounted) setState(() => _data = d);
    } catch (e) {
      if (mounted) setState(() => _error = e);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_data != null) return widget.builder(context, _data as T, _reload);
    // Material so the states also look right when used outside a Scaffold (detail pages)
    if (_loading) return const Material(color: bg, child: Center(child: CircularProgressIndicator()));
    return Material(
      color: bg,
      child: Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          const Icon(Icons.cloud_off, size: 40, color: muted),
          const SizedBox(height: 12),
          Text('$_error', textAlign: TextAlign.center),
          const SizedBox(height: 12),
          FilledButton.tonal(onPressed: _reload, child: const Text('再読み込み')),
          if (Navigator.of(context).canPop())
            TextButton(onPressed: () => Navigator.of(context).pop(), child: const Text('戻る')),
        ]),
      ),
      ),
    );
  }
}

class EmptyNote extends StatelessWidget {
  final String text;
  final IconData icon;
  const EmptyNote(this.text, {super.key, this.icon = Icons.library_music_outlined});
  @override
  Widget build(BuildContext context) => Padding(
        padding: const EdgeInsets.symmetric(vertical: 48, horizontal: 24),
        child: Column(children: [
          Icon(icon, size: 40, color: muted),
          const SizedBox(height: 10),
          Text(text, textAlign: TextAlign.center, style: const TextStyle(color: muted)),
        ]),
      );
}

/// play / shuffle buttons used at the top of lists
class PlayButtons extends StatelessWidget {
  final Future<List<Track>> Function() tracks;
  const PlayButtons({super.key, required this.tracks});
  @override
  Widget build(BuildContext context) => Padding(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
        child: Row(children: [
          Expanded(
            child: FilledButton.icon(
              onPressed: () async {
                final l = await tracks();
                if (l.isEmpty) return toast('曲がありません');
                player.playTracks(l);
              },
              icon: const Icon(Icons.play_arrow),
              label: const Text('再生'),
            ),
          ),
          const SizedBox(width: 12),
          Expanded(
            child: FilledButton.tonalIcon(
              onPressed: () async {
                final l = await tracks();
                if (l.isEmpty) return toast('曲がありません');
                player.shuffleAll(l);
              },
              icon: const Icon(Icons.shuffle),
              label: const Text('シャッフル'),
            ),
          ),
        ]),
      );
}
