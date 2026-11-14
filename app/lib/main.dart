import 'package:audio_service/audio_service.dart';
import 'package:audio_session/audio_session.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'api.dart';
import 'audio_handler.dart';
import 'pages/home_page.dart';
import 'pages/library_page.dart';
import 'pages/login_page.dart';
import 'pages/now_playing.dart';
import 'pages/search_page.dart';
import 'pages/settings_page.dart';
import 'player.dart';
import 'widgets.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final session = await AudioSession.instance;
  await session.configure(const AudioSessionConfiguration.music());
  await api.load();
  await player.init();
  await AudioService.init(
    builder: HakoAudioHandler.new,
    config: const AudioServiceConfig(
      androidNotificationChannelId: 'com.gramme.hako.playback',
      androidNotificationChannelName: '再生中の曲',
      androidNotificationIcon: 'drawable/ic_stat_hako',
      notificationColor: accent,
      androidStopForegroundOnPause: true,
    ),
  );
  runApp(const HakoApp());
}

/// true while signed in; flipped to false when the server rejects the session
final signedIn = ValueNotifier<bool>(api.signedIn);

class HakoApp extends StatelessWidget {
  const HakoApp({super.key});

  @override
  Widget build(BuildContext context) {
    final scheme = ColorScheme.fromSeed(seedColor: accent, brightness: Brightness.dark, surface: bg).copyWith(primary: accent);
    return MaterialApp(
      title: 'Hako',
      debugShowCheckedModeBanner: false,
      scaffoldMessengerKey: messengerKey,
      theme: ThemeData(
        colorScheme: scheme,
        scaffoldBackgroundColor: bg,
        appBarTheme: const AppBarTheme(backgroundColor: bg, surfaceTintColor: Colors.transparent, centerTitle: false),
        navigationBarTheme: NavigationBarThemeData(backgroundColor: surface, indicatorColor: accent.withValues(alpha: 0.25), height: 64),
        listTileTheme: const ListTileThemeData(iconColor: muted),
        useMaterial3: true,
      ),
      home: const Root(),
    );
  }
}

class Root extends StatefulWidget {
  const Root({super.key});
  @override
  State<Root> createState() => _RootState();
}

class _RootState extends State<Root> {
  @override
  void initState() {
    super.initState();
    api.onUnauthorized = () {
      player.stop();
      signedIn.value = false;
      toast('ログインの有効期限が切れました。もう一度ログインしてください');
    };
    if (api.signedIn) _afterSignIn();
  }

  Future<void> _afterSignIn() async {
    try {
      await api.fetchMe();
      await player.restore();
    } catch (_) {
      // offline: keep going, screens show their own errors
    }
  }

  @override
  Widget build(BuildContext context) {
    return ValueListenableBuilder<bool>(
      valueListenable: signedIn,
      builder: (context, ok, _) => ok
          ? const Shell()
          : LoginPage(onSignedIn: () {
              signedIn.value = true;
              _afterSignIn();
            }),
    );
  }
}

/// Bottom tabs, each with its own navigation stack, and the mini player above the tab bar.
class Shell extends StatefulWidget {
  const Shell({super.key});
  @override
  State<Shell> createState() => _ShellState();
}

class _ShellState extends State<Shell> {
  int _tab = 0;
  final _keys = List.generate(4, (_) => GlobalKey<NavigatorState>());

  static const _pages = <Widget>[HomePage(), SearchPage(), LibraryPage(), SettingsPage()];

  void _select(int i) {
    if (i == _tab) {
      // tapping the current tab goes back to its top
      _keys[i].currentState?.popUntil((r) => r.isFirst);
    } else {
      setState(() => _tab = i);
    }
  }

  @override
  Widget build(BuildContext context) {
    return PopScope(
      canPop: false,
      onPopInvokedWithResult: (didPop, _) {
        if (didPop) return;
        final nav = _keys[_tab].currentState;
        if (nav != null && nav.canPop()) {
          nav.pop();
        } else if (_tab != 0) {
          setState(() => _tab = 0);
        } else {
          // leave the app but keep the music playing
          SystemNavigator.pop();
        }
      },
      child: Scaffold(
        body: IndexedStack(
          index: _tab,
          children: [
            for (var i = 0; i < _pages.length; i++)
              Navigator(
                key: _keys[i],
                onGenerateRoute: (_) => MaterialPageRoute(builder: (_) => _pages[i]),
              ),
          ],
        ),
        bottomNavigationBar: Column(mainAxisSize: MainAxisSize.min, children: [
          const MiniPlayer(),
          NavigationBar(
            selectedIndex: _tab,
            onDestinationSelected: _select,
            labelBehavior: NavigationDestinationLabelBehavior.alwaysShow,
            destinations: const [
              NavigationDestination(icon: Icon(Icons.home_outlined), selectedIcon: Icon(Icons.home), label: 'ホーム'),
              NavigationDestination(icon: Icon(Icons.search), label: '検索'),
              NavigationDestination(icon: Icon(Icons.library_music_outlined), selectedIcon: Icon(Icons.library_music), label: 'ライブラリ'),
              NavigationDestination(icon: Icon(Icons.settings_outlined), selectedIcon: Icon(Icons.settings), label: '設定'),
            ],
          ),
        ]),
      ),
    );
  }
}
