import 'package:flutter/material.dart';

import 'data/book_content_repository.dart';
import 'data/local_settings_store.dart';
import 'data/progress_store.dart';
import 'data/score_store.dart';
import 'data/settings_store.dart';
import 'models/user_settings.dart';
import 'screens/library_screen.dart';

void main() {
  runApp(const AidokuApp());
}

/// Minimal reading-app palette being tried out — near-black/near-white
/// backgrounds, grey cards, and exactly one accent color per mode
/// (mirrored: dark mode's accent is the brightest tone against a near-
/// black surface, light mode's is the darkest tone against a near-white
/// one) — the same restrained look most reading apps use so color is
/// reserved for book covers rather than competing with them. See
/// _buildTheme for where these actually get used.
///
/// Dark mode's values reflect active hand-tuning in the running app —
/// treat them as the current experiment, not a locked-in design. Light
/// mode hasn't been hand-tuned the same way yet; it mirrors dark mode's
/// structure (same roles, inverted tone) as a reasonable starting point.
class _DarkPalette {
  static const background = Color.fromARGB(255, 35, 35, 35);
  static const surfaceElement = Color.fromARGB(255, 80, 80, 80);
  static const text = Color(0xFFECECEC);
  static const mutedText = Color(0xFFB0B0B0);
  static const accent = Color.fromARGB(255, 255, 255, 255);
}

class _LightPalette {
  static const background = Color(0xFFFAFAFA);
  static const surfaceElement = Color(0xFFE8E8E8);
  static const text = Color(0xFF1A1A1A);
  static const mutedText = Color(0xFF5A5A5A);
  static const accent = Color(0xFF1A1A1A);
}

/// Builds one brightness's ThemeData from its palette — shared between
/// light/dark so the two can't silently drift in which ColorScheme
/// roles get pinned. onAccent is always that same palette's background:
/// dark mode's bright accent gets dark text, light mode's dark accent
/// gets light text — same mirrored structure as the palettes themselves.
ThemeData _buildTheme({
  required Color background,
  required Color surfaceElement,
  required Color text,
  required Color mutedText,
  required Color accent,
  required Brightness brightness,
}) {
  return ThemeData(
    colorScheme: ColorScheme.fromSeed(seedColor: accent, brightness: brightness)
        .copyWith(
          primary: accent,
          onPrimary: background,
          secondary: mutedText,
          onSecondary: background,
          surface: background,
          onSurface: text,
          surfaceContainerHigh: surfaceElement,
          surfaceContainerHighest: surfaceElement,
          // error/onError/errorContainer intentionally left as
          // Material3's own computed red — that's a semantic
          // wrong-answer signal (see QuestionsView/ChunkListScreen),
          // not part of the visual palette.
        ),
    useMaterial3: true,
  );
}

final _darkTheme = _buildTheme(
  background: _DarkPalette.background,
  surfaceElement: _DarkPalette.surfaceElement,
  text: _DarkPalette.text,
  mutedText: _DarkPalette.mutedText,
  accent: _DarkPalette.accent,
  brightness: Brightness.dark,
);

final _lightTheme = _buildTheme(
  background: _LightPalette.background,
  surfaceElement: _LightPalette.surfaceElement,
  text: _LightPalette.text,
  mutedText: _LightPalette.mutedText,
  accent: _LightPalette.accent,
  brightness: Brightness.light,
);

ThemeMode _toFlutterThemeMode(ThemeModeSetting mode) => switch (mode) {
  ThemeModeSetting.system => ThemeMode.system,
  ThemeModeSetting.light => ThemeMode.light,
  ThemeModeSetting.dark => ThemeMode.dark,
};

class AidokuApp extends StatefulWidget {
  /// Overridable for tests — defaults to this device's local storage.
  /// See SettingsStore's own doc comment for why this is an interface.
  final SettingsStore? settingsStore;

  /// Forwarded straight through to LibraryScreen — same overridable-
  /// for-tests pattern as every other dependency here. Without these,
  /// a widget test driving AidokuApp end to end (Library -> Settings ->
  /// save -> live re-theme) would hit LibraryScreen's real defaults —
  /// an actual network call for the book list, this device's real
  /// storage for progress/score — which is exactly what every other
  /// screen's own tests avoid.
  final BookContentRepository? repository;
  final ProgressStore? progressStore;
  final ScoreStore? scoreStore;

  const AidokuApp({
    super.key,
    this.settingsStore,
    this.repository,
    this.progressStore,
    this.scoreStore,
  });

  @override
  State<AidokuApp> createState() => _AidokuAppState();
}

class _AidokuAppState extends State<AidokuApp> {
  late final SettingsStore _settingsStore =
      widget.settingsStore ?? LocalSettingsStore();

  // Starts at the type's own default (system) while the real saved
  // value loads — a brief flash of the wrong mode on cold start is a
  // fair trade against blocking the very first frame on disk I/O.
  ThemeModeSetting _themeMode = ThemeModeSetting.system;

  @override
  void initState() {
    super.initState();
    _loadThemeMode();
  }

  Future<void> _loadThemeMode() async {
    final settings = await _settingsStore.getSettings();
    if (!mounted) return;
    setState(() => _themeMode = settings.themeMode);
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'HonTawny',
      theme: _lightTheme,
      darkTheme: _darkTheme,
      themeMode: _toFlutterThemeMode(_themeMode),
      home: LibraryScreen(
        repository: widget.repository,
        progressStore: widget.progressStore,
        scoreStore: widget.scoreStore,
        settingsStore: _settingsStore,
        // SettingsScreen (opened from here) is where the reader
        // actually changes this — see its own onThemeModeChanged doc
        // comment for why this can't just be a normal setState() a few
        // widgets down.
        onThemeModeChanged: (mode) => setState(() => _themeMode = mode),
      ),
    );
  }
}
