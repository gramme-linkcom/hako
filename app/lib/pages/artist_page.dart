import 'package:flutter/material.dart';

import '../api.dart';
import '../models.dart';
import '../widgets.dart';

class ArtistPage extends StatelessWidget {
  final int id;
  const ArtistPage({super.key, required this.id});

  @override
  Widget build(BuildContext context) {
    return Loader<ArtistDetail>(
      load: () => api.artist(id),
      builder: (context, d, reload) => _ArtistView(d),
    );
  }
}

class _ArtistView extends StatefulWidget {
  final ArtistDetail d;
  const _ArtistView(this.d);
  @override
  State<_ArtistView> createState() => _ArtistViewState();
}

class _ArtistViewState extends State<_ArtistView> {
  List<Track>? _all;

  Future<List<Track>> _allTracks() async => _all ??= await api.trackBatch(widget.d.trackIds);

  @override
  Widget build(BuildContext context) {
    final d = widget.d;
    final a = d.artist;
    return Scaffold(
      appBar: AppBar(title: Text(a.name), actions: [
        IconButton(
          icon: Icon(a.fav ? Icons.favorite : Icons.favorite_border, color: a.fav ? accent : null),
          onPressed: () {
            setState(() => a.fav = !a.fav);
            toggleFavorite('artist', a.id, a.fav);
          },
        ),
      ]),
      body: CustomScrollView(slivers: [
        SliverToBoxAdapter(
          child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 0, 16, 0),
              child: Text('アルバム ${a.albumCount}枚 · ${a.trackCount}曲', style: const TextStyle(color: muted)),
            ),
            PlayButtons(tracks: _allTracks),
            if (d.top.isNotEmpty) ...[
              const SectionTitle('人気の曲'),
              for (var i = 0; i < d.top.length; i++) TrackTile(list: d.top, index: i),
            ],
            if (d.albums.isNotEmpty) const SectionTitle('アルバム'),
          ]),
        ),
        if (d.albums.isNotEmpty) AlbumGrid(d.albums),
        if (d.appearsOn.isNotEmpty) ...[
          const SliverToBoxAdapter(child: SectionTitle('参加作品')),
          AlbumGrid(d.appearsOn),
        ],
        const SliverToBoxAdapter(child: SizedBox(height: 24)),
      ]),
    );
  }
}
