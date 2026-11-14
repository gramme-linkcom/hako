import 'package:flutter/material.dart';

import '../api.dart';
import '../models.dart';
import '../player.dart';
import '../widgets.dart';

class AlbumPage extends StatelessWidget {
  final int id;
  const AlbumPage({super.key, required this.id});

  @override
  Widget build(BuildContext context) {
    return Loader<AlbumDetail>(
      load: () => api.album(id),
      builder: (context, d, reload) => _AlbumView(d),
    );
  }
}

class _AlbumView extends StatefulWidget {
  final AlbumDetail d;
  const _AlbumView(this.d);
  @override
  State<_AlbumView> createState() => _AlbumViewState();
}

class _AlbumViewState extends State<_AlbumView> {
  @override
  Widget build(BuildContext context) {
    final a = widget.d.album;
    final tracks = widget.d.tracks;
    final multiDisc = tracks.map((t) => t.discNo).toSet().length > 1;
    final width = MediaQuery.sizeOf(context).width;
    final coverSize = (width * 0.62).clamp(160.0, 320.0);
    final children = <Widget>[];
    var disc = -1;
    for (var i = 0; i < tracks.length; i++) {
      if (multiDisc && tracks[i].discNo != disc) {
        disc = tracks[i].discNo;
        children.add(Padding(
          padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
          child: Text('Disc ${disc > 0 ? disc : '?'}', style: const TextStyle(color: muted, fontWeight: FontWeight.w600)),
        ));
      }
      children.add(TrackTile(list: tracks, index: i, showCover: false));
    }

    return Scaffold(
      appBar: AppBar(actions: [
        IconButton(
          icon: Icon(a.fav ? Icons.favorite : Icons.favorite_border, color: a.fav ? accent : null),
          tooltip: 'お気に入り',
          onPressed: () {
            setState(() => a.fav = !a.fav);
            toggleFavorite('album', a.id, a.fav);
          },
        ),
        PopupMenuButton<String>(
          onSelected: (v) {
            switch (v) {
              case 'next':
                player.playNext(tracks);
                toast('次に再生します');
              case 'queue':
                player.addToQueue(tracks);
                toast('キューに追加しました');
              case 'playlist':
                addToPlaylistDialog(context, tracks);
              case 'artist':
                openArtist(context, a.artistId);
            }
          },
          itemBuilder: (_) => const [
            PopupMenuItem(value: 'next', child: Text('次に再生')),
            PopupMenuItem(value: 'queue', child: Text('キューに追加')),
            PopupMenuItem(value: 'playlist', child: Text('プレイリストに追加')),
            PopupMenuItem(value: 'artist', child: Text('アーティストを開く')),
          ],
        ),
      ]),
      body: ListView(padding: const EdgeInsets.only(bottom: 24), children: [
        Center(child: Cover(albumId: a.id, cv: a.cv, title: a.title, size: coverSize, radius: 10, width: 800)),
        const SizedBox(height: 16),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 20),
          child: Column(children: [
            Text(a.title, textAlign: TextAlign.center, style: const TextStyle(fontSize: 20, fontWeight: FontWeight.w800)),
            const SizedBox(height: 4),
            InkWell(
              onTap: () => openArtist(context, a.artistId),
              child: Text(a.artist, style: const TextStyle(color: accent, fontSize: 15)),
            ),
            const SizedBox(height: 6),
            Wrap(alignment: WrapAlignment.center, crossAxisAlignment: WrapCrossAlignment.center, spacing: 8, children: [
              Text(
                [if (a.year > 0) '${a.year}', if (a.genre.isNotEmpty) a.genre, '${a.trackCount}曲', fmtLong(a.duration)].join(' · '),
                style: const TextStyle(color: muted, fontSize: 13),
              ),
              QBadge(a.quality, spec: fmtSpec(a.maxBits, a.maxRate)),
            ]),
          ]),
        ),
        PlayButtons(tracks: () async => tracks),
        ...children,
      ]),
    );
  }
}
