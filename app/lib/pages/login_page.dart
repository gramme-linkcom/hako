import 'package:flutter/material.dart';

import '../api.dart';
import '../widgets.dart';

class LoginPage extends StatefulWidget {
  final VoidCallback onSignedIn;
  const LoginPage({super.key, required this.onSignedIn});
  @override
  State<LoginPage> createState() => _LoginPageState();
}

class _LoginPageState extends State<LoginPage> {
  final _server = TextEditingController(text: api.server);
  final _name = TextEditingController();
  final _pass = TextEditingController();
  final _code = TextEditingController();
  String? _ticket; // set when the account uses two-step verification
  String? _error;
  bool _busy = false;

  Future<void> _submit() async {
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      if (_ticket == null) {
        if (_server.text.trim().isEmpty) throw ApiException(0, 'サーバーのアドレスを入力してください');
        await api.setServer(_server.text);
        _server.text = api.server;
        if (await api.checkServer()) {
          throw ApiException(0, 'サーバーの初期設定がまだです。先にブラウザで開いて管理者アカウントを作成してください');
        }
        final r = await api.login(_name.text.trim(), _pass.text);
        if (r.mfaTicket != null) {
          setState(() => _ticket = r.mfaTicket);
          return;
        }
      } else {
        await api.loginMfa(_ticket!, _code.text.trim());
      }
      _pass.clear();
      widget.onSignedIn();
    } catch (e) {
      setState(() => _error = '$e');
      // the 2FA ticket expires after a few minutes: start over on failure that says so
      if (e is ApiException && e.status == 401 && _ticket != null && e.message.contains('期限')) _ticket = null;
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final mfa = _ticket != null;
    return Scaffold(
      body: SafeArea(
        child: Center(
          child: SingleChildScrollView(
            padding: const EdgeInsets.all(24),
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 420),
              child: AutofillGroup(
                child: Column(crossAxisAlignment: CrossAxisAlignment.stretch, children: [
                  ClipRRect(borderRadius: BorderRadius.circular(18), child: Image.asset('assets/icon.png', width: 72, height: 72)),
                  const SizedBox(height: 12),
                  const Text('Hako', textAlign: TextAlign.center, style: TextStyle(fontSize: 28, fontWeight: FontWeight.w800)),
                  const SizedBox(height: 28),
                  if (!mfa) ...[
                    TextField(
                      controller: _server,
                      keyboardType: TextInputType.url,
                      autocorrect: false,
                      decoration: const InputDecoration(
                        labelText: 'サーバー',
                        hintText: 'https://hako.xxxx.ts.net',
                        helperText: 'Tailscale のアドレス、または http://192.168.x.x:8096',
                        border: OutlineInputBorder(),
                      ),
                    ),
                    const SizedBox(height: 16),
                    TextField(
                      controller: _name,
                      autocorrect: false,
                      autofillHints: const [AutofillHints.username],
                      decoration: const InputDecoration(labelText: 'ユーザー名', border: OutlineInputBorder()),
                    ),
                    const SizedBox(height: 16),
                    TextField(
                      controller: _pass,
                      obscureText: true,
                      autofillHints: const [AutofillHints.password],
                      decoration: const InputDecoration(labelText: 'パスワード', border: OutlineInputBorder()),
                      onSubmitted: (_) => _submit(),
                    ),
                  ] else ...[
                    const Text('2段階認証', style: TextStyle(fontSize: 18, fontWeight: FontWeight.w700)),
                    const SizedBox(height: 6),
                    const Text('認証アプリの6桁のコード、または回復コードを入力してください', style: TextStyle(color: muted)),
                    const SizedBox(height: 16),
                    TextField(
                      controller: _code,
                      autofocus: true,
                      keyboardType: TextInputType.visiblePassword,
                      autofillHints: const [AutofillHints.oneTimeCode],
                      decoration: const InputDecoration(labelText: 'コード', border: OutlineInputBorder()),
                      onSubmitted: (_) => _submit(),
                    ),
                  ],
                  if (_error != null) ...[
                    const SizedBox(height: 14),
                    Text(_error!, style: TextStyle(color: Theme.of(context).colorScheme.error)),
                  ],
                  const SizedBox(height: 20),
                  FilledButton(
                    onPressed: _busy ? null : _submit,
                    style: FilledButton.styleFrom(padding: const EdgeInsets.symmetric(vertical: 14)),
                    child: _busy
                        ? const SizedBox(width: 20, height: 20, child: CircularProgressIndicator(strokeWidth: 2))
                        : Text(mfa ? '確認' : 'ログイン'),
                  ),
                  if (mfa)
                    TextButton(
                      onPressed: () => setState(() {
                        _ticket = null;
                        _code.clear();
                        _error = null;
                      }),
                      child: const Text('戻る'),
                    ),
                ]),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
