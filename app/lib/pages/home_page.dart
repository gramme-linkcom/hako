import 'package:flutter/material.dart';

import '../api.dart';
import '../models.dart';
import '../player.dart';
import '../widgets.dart';
import 'library_page.dart';

class HomePage extends StatelessWidget {
  const HomePage({super.key});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Hako', style: TextStyle(fontWeight: FontWeight.w800))),
      body: Loader<HomeData>(
        load: api.home,
        builder: (context, d, reload) {
          final empty = d.recent.isEmpty && d.top.isEmpty;
          return RefreshIndicator(
            onRefresh: reload,
            child: ListView(
              padding: const EdgeInsets.only(bottom: 24),
              children: [
                if (empty) const EmptyNote('ライブラリが空です。\nサーバーの設定で音楽フォルダを追加してください'),
                if (!empty)
                  Padding(
                    padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
                    child: Row(children: [
                      Expanded(
                        child: FilledButton.tonalIcon(
                          icon: const Icon(Icons.shuffle),
                          label: const Text('すべてシャッフル'),
                          onPressed: () async {
                            try {
                              final (l, _) = await api.tracks(sort: 'random', limit: 500);
                              player.playTracks(l);
                            } catch (e) {
                              toast('$e');
                            }
                          },
                        ),
                      ),
                    ]),
                  ),
                AlbumRow('最近聴いたアルバム', d.played),
                if (d.top.isNotEmpty) ...[
                  SectionTitle('よく聴く曲',
                      trailing: TextButton(
                        onPressed: () => Navigator.of(context).push(MaterialPageRoute(builder: (_) => const TracksPage(title: 'よく聴く曲', sort: 'plays'))),
                        child: const Text('すべて'),
                      )),
                  for (var i = 0; i < d.top.length && i < 5; i++) TrackTile(list: d.top, index: i),
                ],
                AlbumRow('最近追加されたアルバム', d.recent),
                AlbumRow('ランダム', d.random),
              ],
            ),
          );
        },
      ),
    );
  }
}
