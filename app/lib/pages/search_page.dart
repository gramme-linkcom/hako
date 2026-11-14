import 'dart:async';

import 'package:flutter/material.dart';

import '../api.dart';
import '../models.dart';
import '../widgets.dart';
import 'library_page.dart';

class SearchPage extends StatefulWidget {
  const SearchPage({super.key});
  @override
  State<SearchPage> createState() => _SearchPageState();
}

class _SearchPageState extends State<SearchPage> {
  final _c = TextEditingController();
  Timer? _debounce;
  SearchResult? _r;
  String _q = '';
  bool _loading = false;
  String? _error;

  void _onChanged(String v) {
    _debounce?.cancel();
    _debounce = Timer(const Duration(milliseconds: 300), () => _run(v.trim()));
  }

  Future<void> _run(String q) async {
    _q = q;
    if (q.isEmpty) {
      setState(() => _r = null);
      return;
    }
    setState(() => _loading = true);
    try {
      final r = await api.search(q);
      if (mounted && q == _q) {
        setState(() {
          _r = r;
          _error = null;
        });
      }
    } catch (e) {
      if (mounted) setState(() => _error = '$e');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  void dispose() {
    _debounce?.cancel();
    _c.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final r = _r;
    return Scaffold(
      appBar: AppBar(
        titleSpacing: 12,
        title: TextField(
          controller: _c,
          autofocus: false,
          textInputAction: TextInputAction.search,
          onChanged: _onChanged,
          onSubmitted: (v) => _run(v.trim()),
          decoration: InputDecoration(
            hintText: '曲・アルバム・アーティスト',
            prefixIcon: const Icon(Icons.search),
            suffixIcon: _c.text.isEmpty
                ? null
                : IconButton(
                    icon: const Icon(Icons.close),
                    onPressed: () {
                      _c.clear();
                      _run('');
                    }),
            filled: true,
            fillColor: surface,
            isDense: true,
            border: OutlineInputBorder(borderRadius: BorderRadius.circular(24), borderSide: BorderSide.none),
          ),
        ),
        // always present: adding/removing `bottom` rebuilds the AppBar's layout, which recreates
        // the TextField and closes the keyboard while typing
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(2),
          child: _loading ? const LinearProgressIndicator(minHeight: 2) : const SizedBox(height: 2),
        ),
      ),
      body: _error != null
          ? Center(child: Text(_error!))
          : r == null
              ? const EmptyNote('キーワードを入力してください', icon: Icons.search)
              : r.isEmpty
                  ? const EmptyNote('見つかりませんでした', icon: Icons.search_off)
                  : ListView(
                      keyboardDismissBehavior: ScrollViewKeyboardDismissBehavior.onDrag,
                      padding: const EdgeInsets.only(bottom: 24),
                      children: [
                        if (r.artists.isNotEmpty) ...[
                          const SectionTitle('アーティスト'),
                          for (final a in r.artists.take(8)) ArtistTile(a),
                        ],
                        AlbumRow('アルバム', r.albums),
                        if (r.tracks.isNotEmpty) ...[
                          const SectionTitle('曲'),
                          for (var i = 0; i < r.tracks.length; i++) TrackTile(list: r.tracks, index: i),
                        ],
                      ],
                    ),
    );
  }
}
