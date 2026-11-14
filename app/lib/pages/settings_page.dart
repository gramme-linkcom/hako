import 'package:flutter/material.dart';

import '../api.dart';
import '../main.dart';
import '../player.dart';
import '../widgets.dart';

class SettingsPage extends StatefulWidget {
  const SettingsPage({super.key});
  @override
  State<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends State<SettingsPage> {
  static const _qualities = {'orig': '元の音質（変換しない）', '320': '320kbps', '192': '192kbps', '128': '128kbps'};
  static const _norms = {'auto': '自動（アルバム連続再生時はアルバム単位）', 'track': '曲ごと', 'album': 'アルバム単位', 'off': 'オフ'};

  Future<void> _pick(String title, Map<String, String> options, String current, Future<void> Function(String) onPick) async {
    final v = await showDialog<String>(
      context: context,
      useRootNavigator: true,
      builder: (ctx) => SimpleDialog(
        title: Text(title),
        children: [
          RadioGroup<String>(
            groupValue: current,
            onChanged: (v) => Navigator.pop(ctx, v),
            child: Column(children: [
              for (final e in options.entries) RadioListTile<String>(value: e.key, title: Text(e.value)),
            ]),
          ),
        ],
      ),
    );
    if (v != null && v != current) {
      await onPick(v);
      if (mounted) setState(() {});
    }
  }

  @override
  Widget build(BuildContext context) {
    final p = player.prefs;
    final me = api.me;
    return Scaffold(
      appBar: AppBar(title: const Text('設定')),
      body: ListView(children: [
        const _Header('再生'),
        ListTile(
          leading: const Icon(Icons.high_quality_outlined),
          title: const Text('音質'),
          subtitle: Text(_qualities[p.quality] ?? p.quality),
          onTap: () => _pick('音質', _qualities, p.quality, (v) async {
            p.quality = v;
            await player.applyPrefs(reload: true);
          }),
        ),
        const Padding(
          padding: EdgeInsets.fromLTRB(72, 0, 16, 8),
          child: Text('モバイル回線で通信量を抑えたいときは 192kbps などに。それより高音質の曲はサーバーで MP3 に変換して送られます',
              style: TextStyle(color: muted, fontSize: 12)),
        ),
        ListTile(
          leading: const Icon(Icons.equalizer),
          title: const Text('音量の均一化'),
          subtitle: Text(_norms[p.norm] ?? p.norm),
          onTap: () => _pick('音量の均一化', _norms, p.norm, (v) async {
            p.norm = v;
            await player.applyPrefs();
          }),
        ),
        if (p.norm != 'off')
          Padding(
            padding: const EdgeInsets.fromLTRB(72, 0, 16, 8),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text('基準の音量: ${p.target.toStringAsFixed(0)} LUFS', style: const TextStyle(fontSize: 14)),
              Slider(
                value: p.target,
                min: -18,
                max: -10,
                divisions: 8,
                label: '${p.target.toStringAsFixed(0)} LUFS',
                onChanged: (v) => setState(() => p.target = v),
                onChangeEnd: (_) => player.applyPrefs(),
              ),
              const Text('アプリ版は静かな曲を持ち上げることもできます（-14 がおすすめ）', style: TextStyle(color: muted, fontSize: 12)),
            ]),
          ),
        const _Header('アカウント'),
        ListTile(leading: const Icon(Icons.dns_outlined), title: const Text('サーバー'), subtitle: Text(api.server)),
        ListTile(
          leading: const Icon(Icons.person_outline),
          title: const Text('ユーザー'),
          subtitle: Text(me == null ? '-' : '${me.name}${me.admin ? '（管理者）' : ''}${me.totp ? ' · 2段階認証オン' : ''}'),
        ),
        const Padding(
          padding: EdgeInsets.fromLTRB(72, 0, 16, 8),
          child: Text('音楽フォルダ・ユーザー・2段階認証の設定は、ブラウザ版の設定画面から行えます', style: TextStyle(color: muted, fontSize: 12)),
        ),
        ListTile(
          leading: const Icon(Icons.logout),
          title: const Text('ログアウト'),
          onTap: () async {
            if (!await confirm(context, 'ログアウトしますか？', ok: 'ログアウト')) return;
            await player.stop();
            await api.logout();
            signedIn.value = false;
          },
        ),
        const _Header('情報'),
        const ListTile(leading: Icon(Icons.info_outline), title: Text('Hako for Android'), subtitle: Text('バージョン 0.1.0')),
        const SizedBox(height: 24),
      ]),
    );
  }
}

class _Header extends StatelessWidget {
  final String text;
  const _Header(this.text);
  @override
  Widget build(BuildContext context) =>
      Padding(padding: const EdgeInsets.fromLTRB(16, 20, 16, 4), child: Text(text, style: const TextStyle(color: accent, fontWeight: FontWeight.w700)));
}
