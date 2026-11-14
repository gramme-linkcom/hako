import 'package:flutter/material.dart';

import '../api.dart';
import '../models.dart';
import '../player.dart';
import '../widgets.dart';

class LibraryPage extends StatelessWidget {
  const LibraryPage({super.key});

  @override
  Widget build(BuildContext context) {
    void go(Widget p) => Navigator.of(context).push(MaterialPageRoute(builder: (_) => p));
    Widget item(IconData icon, String label, Widget page) =>
        ListTile(leading: Icon(icon, color: accent), title: Text(label), trailing: const Icon(Icons.chevron_right), onTap: () => go(page));
    return Scaffold(
      appBar: AppBar(title: const Text('ライブラリ')),
      body: ListView(children: [
        item(Icons.album_outlined, 'アルバム', const AlbumsPage()),
        item(Icons.person_outline, 'アーティスト', const ArtistsPage()),
        item(Icons.music_note_outlined, '曲', const TracksPage(title: '曲')),
        item(Icons.queue_music, 'プレイリスト', const PlaylistsPage()),
        item(Icons.favorite_border, 'お気に入り', const FavoritesPage()),
        const Divider(),
        item(Icons.history, '最近再生した曲', const TracksPage(title: '最近再生した曲', sort: 'recent')),
        item(Icons.trending_up, 'よく聴く曲', const TracksPage(title: 'よく聴く曲', sort: 'plays')),
        item(Icons.new_releases_outlined, '最近追加された曲', const TracksPage(title: '最近追加された曲', sort: 'added')),
        const Divider(),
        item(Icons.high_quality_outlined, 'ハイレゾのアルバム', const AlbumsPage(quality: 'hires', title: 'ハイレゾ')),
        item(Icons.graphic_eq, 'ロスレスのアルバム', const AlbumsPage(quality: 'lossless', title: 'ロスレス')),
      ]),
    );
  }
}

// ---------------- albums ----------------

class AlbumsPage extends StatefulWidget {
  final String? quality;
  final String title;
  const AlbumsPage({super.key, this.quality, this.title = 'アルバム'});
  @override
  State<AlbumsPage> createState() => _AlbumsPageState();
}

class _AlbumsPageState extends State<AlbumsPage> {
  String _sort = 'added';
  static const _sorts = {'added': '追加日', 'title': 'タイトル', 'artist': 'アーティスト', 'year': '年', 'random': 'ランダム'};

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(widget.title), actions: [
        PopupMenuButton<String>(
          icon: const Icon(Icons.sort),
          initialValue: _sort,
          onSelected: (v) => setState(() => _sort = v),
          itemBuilder: (_) => [for (final e in _sorts.entries) PopupMenuItem(value: e.key, child: Text(e.value))],
        ),
      ]),
      body: Loader<List<Album>>(
        key: ValueKey(_sort),
        load: () => api.albums(sort: _sort, quality: widget.quality),
        builder: (context, albums, reload) => RefreshIndicator(
          onRefresh: reload,
          child: albums.isEmpty
              ? ListView(children: const [EmptyNote('アルバムがありません')])
              : CustomScrollView(slivers: [
                  SliverToBoxAdapter(
                    child: Padding(
                      padding: const EdgeInsets.fromLTRB(16, 4, 16, 0),
                      child: Text('${albums.length}枚', style: const TextStyle(color: muted)),
                    ),
                  ),
                  AlbumGrid(albums),
                ]),
        ),
      ),
    );
  }
}

// ---------------- artists ----------------

class ArtistTile extends StatelessWidget {
  final Artist a;
  const ArtistTile(this.a, {super.key});
  @override
  Widget build(BuildContext context) => ListTile(
        leading: ClipOval(child: Cover(albumId: a.coverAlbum, cv: a.cv, title: a.name, size: 44, radius: 0, width: 96)),
        title: Text(a.name, maxLines: 1, overflow: TextOverflow.ellipsis),
        subtitle: Text('アルバム ${a.albumCount}枚 · ${a.trackCount}曲', style: const TextStyle(color: muted, fontSize: 12)),
        onTap: () => openArtist(context, a.id),
      );
}

class ArtistsPage extends StatefulWidget {
  const ArtistsPage({super.key});
  @override
  State<ArtistsPage> createState() => _ArtistsPageState();
}

class _ArtistsPageState extends State<ArtistsPage> {
  String _filter = '';
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('アーティスト')),
      body: Loader<List<Artist>>(
        load: api.artists,
        builder: (context, all, reload) {
          final f = _filter.toLowerCase();
          final list = f.isEmpty ? all : all.where((a) => a.name.toLowerCase().contains(f)).toList();
          return RefreshIndicator(
            onRefresh: reload,
            child: ListView.builder(
              keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
              itemCount: list.length + 1,
              itemBuilder: (context, i) {
                if (i == 0) {
                  return Padding(
                    padding: const EdgeInsets.fromLTRB(16, 4, 16, 8),
                    child: TextField(
                      onChanged: (v) => setState(() => _filter = v.trim()),
                      decoration: InputDecoration(
                        hintText: '${all.length}人から絞り込み',
                        prefixIcon: const Icon(Icons.filter_list),
                        isDense: true,
                        filled: true,
                        fillColor: surface,
                        border: OutlineInputBorder(borderRadius: BorderRadius.circular(24), borderSide: BorderSide.none),
                      ),
                    ),
                  );
                }
                return ArtistTile(list[i - 1]);
              },
            ),
          );
        },
      ),
    );
  }
}

// ---------------- tracks (paged) ----------------

class TracksPage extends StatefulWidget {
  final String title;
  final String sort;
  final String? quality;
  const TracksPage({super.key, required this.title, this.sort = 'title', this.quality});
  @override
  State<TracksPage> createState() => _TracksPageState();
}

class _TracksPageState extends State<TracksPage> {
  final List<Track> _tracks = [];
  int _total = -1;
  bool _loading = false;
  String? _error;
  static const _page = 300;

  @override
  void initState() {
    super.initState();
    _more();
  }

  Future<void> _more({bool reset = false}) async {
    if (_loading) return;
    if (reset) {
      _tracks.clear();
      _total = -1;
    }
    if (_total >= 0 && _tracks.length >= _total) return;
    setState(() => _loading = true);
    try {
      final (l, total) = await api.tracks(sort: widget.sort, quality: widget.quality, limit: _page, offset: _tracks.length);
      _tracks.addAll(l);
      _total = l.isEmpty ? _tracks.length : total;
      _error = null;
    } catch (e) {
      _error = '$e';
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<List<Track>> _all() async {
    // play buttons use up to 2000 tracks
    if (_total > _tracks.length && _tracks.length < 2000) {
      final (l, _) = await api.tracks(sort: widget.sort, quality: widget.quality, limit: 2000);
      return l;
    }
    return _tracks;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(widget.title)),
      body: _tracks.isEmpty && _loading
          ? const Center(child: CircularProgressIndicator())
          : _tracks.isEmpty && _error != null
              ? Center(child: Text(_error!))
              : RefreshIndicator(
                  onRefresh: () => _more(reset: true),
                  child: NotificationListener<ScrollNotification>(
                    onNotification: (n) {
                      if (n.metrics.extentAfter < 800) _more();
                      return false;
                    },
                    child: ListView.builder(
                      itemCount: _tracks.length + 2,
                      itemBuilder: (context, i) {
                        if (i == 0) {
                          return Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
                            if (_tracks.isNotEmpty) PlayButtons(tracks: _all),
                            Padding(
                              padding: const EdgeInsets.fromLTRB(16, 0, 16, 4),
                              child: Text(_total >= 0 ? '$_total曲' : '', style: const TextStyle(color: muted)),
                            ),
                            if (_tracks.isEmpty) const EmptyNote('曲がありません'),
                          ]);
                        }
                        if (i == _tracks.length + 1) {
                          return _loading ? const Padding(padding: EdgeInsets.all(16), child: Center(child: CircularProgressIndicator())) : const SizedBox(height: 16);
                        }
                        return TrackTile(list: _tracks, index: i - 1);
                      },
                    ),
                  ),
                ),
    );
  }
}

// ---------------- favorites ----------------

class FavoritesPage extends StatelessWidget {
  const FavoritesPage({super.key});
  @override
  Widget build(BuildContext context) {
    return DefaultTabController(
      length: 3,
      child: Scaffold(
        appBar: AppBar(
          title: const Text('お気に入り'),
          bottom: const TabBar(tabs: [Tab(text: '曲'), Tab(text: 'アルバム'), Tab(text: 'アーティスト')]),
        ),
        body: Loader<Favorites>(
          load: api.favorites,
          builder: (context, f, reload) => TabBarView(children: [
            RefreshIndicator(
              onRefresh: reload,
              child: f.tracks.isEmpty
                  ? ListView(children: const [EmptyNote('お気に入りの曲はまだありません', icon: Icons.favorite_border)])
                  : ListView.builder(
                      itemCount: f.tracks.length + 1,
                      itemBuilder: (_, i) => i == 0 ? PlayButtons(tracks: () async => f.tracks) : TrackTile(list: f.tracks, index: i - 1),
                    ),
            ),
            RefreshIndicator(
              onRefresh: reload,
              child: f.albums.isEmpty
                  ? ListView(children: const [EmptyNote('お気に入りのアルバムはまだありません', icon: Icons.favorite_border)])
                  : CustomScrollView(slivers: [AlbumGrid(f.albums)]),
            ),
            RefreshIndicator(
              onRefresh: reload,
              child: f.artists.isEmpty
                  ? ListView(children: const [EmptyNote('お気に入りのアーティストはまだありません', icon: Icons.favorite_border)])
                  : ListView(children: [for (final a in f.artists) ArtistTile(a)]),
            ),
          ]),
        ),
      ),
    );
  }
}

// ---------------- playlists ----------------

class PlaylistCover extends StatelessWidget {
  final Playlist p;
  final double size;
  const PlaylistCover(this.p, {super.key, this.size = 52});
  @override
  Widget build(BuildContext context) {
    if (p.covers.length >= 4) {
      final s = size / 2;
      return ClipRRect(
        borderRadius: BorderRadius.circular(6),
        child: SizedBox(
          width: size,
          height: size,
          child: Wrap(children: [for (final id in p.covers.take(4)) Cover(albumId: id, size: s, radius: 0, width: 96)]),
        ),
      );
    }
    return Cover(albumId: p.covers.isNotEmpty ? p.covers.first : 0, title: p.name, size: size, width: size > 100 ? 500 : 96);
  }
}

class PlaylistsPage extends StatefulWidget {
  const PlaylistsPage({super.key});
  @override
  State<PlaylistsPage> createState() => _PlaylistsPageState();
}

class _PlaylistsPageState extends State<PlaylistsPage> {
  int _gen = 0;

  Future<void> _create() async {
    final name = await promptText(context, '新しいプレイリスト', hint: '名前');
    if (name == null || name.trim().isEmpty) return;
    try {
      await api.createPlaylist(name.trim());
      setState(() => _gen++);
    } catch (e) {
      toast('$e');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('プレイリスト'), actions: [IconButton(icon: const Icon(Icons.add), tooltip: '新規作成', onPressed: _create)]),
      body: Loader<List<Playlist>>(
        key: ValueKey(_gen),
        load: api.playlists,
        builder: (context, pls, reload) => RefreshIndicator(
          onRefresh: reload,
          child: pls.isEmpty
              ? ListView(children: const [EmptyNote('プレイリストはまだありません。\n右上の＋から作成できます', icon: Icons.queue_music)])
              : ListView(children: [
                  for (final p in pls)
                    ListTile(
                      leading: PlaylistCover(p),
                      title: Text(p.name),
                      subtitle: Text('${p.trackCount}曲 · ${fmtLong(p.duration)}', style: const TextStyle(color: muted, fontSize: 12)),
                      onTap: () async {
                        await Navigator.of(context).push(MaterialPageRoute(builder: (_) => PlaylistPage(id: p.id)));
                        if (mounted) setState(() => _gen++);
                      },
                    ),
                ]),
        ),
      ),
    );
  }
}

class PlaylistPage extends StatefulWidget {
  final int id;
  const PlaylistPage({super.key, required this.id});
  @override
  State<PlaylistPage> createState() => _PlaylistPageState();
}

class _PlaylistPageState extends State<PlaylistPage> {
  int _gen = 0;

  @override
  Widget build(BuildContext context) {
    return Loader<PlaylistDetail>(
      key: ValueKey(_gen),
      load: () => api.playlist(widget.id),
      builder: (context, d, reload) {
        final p = d.playlist;
        final tracks = d.tracks;
        Future<void> save(List<Track> l) async {
          try {
            await api.updatePlaylist(p.id, trackIds: l.map((t) => t.id).toList());
          } catch (e) {
            toast('$e');
          }
          if (mounted) setState(() => _gen++);
        }

        return Scaffold(
          appBar: AppBar(title: Text(p.name), actions: [
            PopupMenuButton<String>(
              onSelected: (v) async {
                if (v == 'rename') {
                  final n = await promptText(context, '名前を変更', initial: p.name);
                  if (n == null || n.trim().isEmpty) return;
                  await api.updatePlaylist(p.id, name: n.trim()).catchError((e) => toast('$e'));
                  if (mounted) setState(() => _gen++);
                } else if (v == 'queue') {
                  player.addToQueue(tracks);
                  toast('キューに追加しました');
                } else if (v == 'delete') {
                  if (!await confirm(context, '「${p.name}」を削除しますか？', ok: '削除')) return;
                  await api.deletePlaylist(p.id).catchError((e) => toast('$e'));
                  if (context.mounted) Navigator.of(context).pop();
                }
              },
              itemBuilder: (_) => const [
                PopupMenuItem(value: 'queue', child: Text('キューに追加')),
                PopupMenuItem(value: 'rename', child: Text('名前を変更')),
                PopupMenuItem(value: 'delete', child: Text('削除')),
              ],
            ),
          ]),
          body: RefreshIndicator(
            onRefresh: reload,
            child: ListView.builder(
              itemCount: tracks.length + 1,
              itemBuilder: (context, i) {
                if (i == 0) {
                  return Column(children: [
                    const SizedBox(height: 8),
                    PlaylistCover(p, size: 160),
                    const SizedBox(height: 10),
                    Text('${p.trackCount}曲 · ${fmtLong(p.duration)}', style: const TextStyle(color: muted)),
                    if (tracks.isNotEmpty) PlayButtons(tracks: () async => tracks) else const EmptyNote('曲がありません。曲のメニューから追加できます'),
                  ]);
                }
                return TrackTile(
                  list: tracks,
                  index: i - 1,
                  onRemove: () => save(List.of(tracks)..removeAt(i - 1)),
                );
              },
            ),
          ),
        );
      },
    );
  }
}
