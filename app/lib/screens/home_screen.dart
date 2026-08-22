import 'package:flutter/material.dart';

import '../data/book_content_repository.dart';
import '../data/progress_store.dart';
import '../data/score_store.dart';
import '../data/settings_store.dart';
import '../models/user_settings.dart';
import 'library_screen.dart';
import 'shop_screen.dart';

/// The app's root screen: a bottom NavigationBar switching between
/// ShopScreen ("Store") and LibraryScreen ("My Library") — Store first,
/// matching the order they were asked for. Each tab keeps its own
/// Scaffold/AppBar (nested Scaffolds are a normal, supported Flutter
/// pattern) rather than this screen owning one shared AppBar, so
/// neither tab's internals needed restructuring to gain a bottom nav.
///
/// Uses IndexedStack, not a plain switch on the selected index — a
/// switch would tear down and rebuild the non-selected tab's State
/// every time, re-fetching its books and losing scroll position each
/// time the reader switches tabs; IndexedStack keeps both alive,
/// just paints one.
class HomeScreen extends StatefulWidget {
  /// Overridable for tests — forwarded to both tabs; see each screen's
  /// own doc comment for why these are interfaces at all.
  final BookContentRepository? repository;
  final ProgressStore? progressStore;
  final ScoreStore? scoreStore;
  final SettingsStore? settingsStore;

  /// Forwarded to LibraryScreen (Store has no theme setting of its own)
  /// — see SettingsScreen's own doc comment for why AidokuApp needs
  /// this callback threaded all the way down rather than a plain
  /// setState() partway.
  final ValueChanged<ThemeModeSetting>? onThemeModeChanged;

  const HomeScreen({
    super.key,
    this.repository,
    this.progressStore,
    this.scoreStore,
    this.settingsStore,
    this.onThemeModeChanged,
  });

  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  int _selectedIndex = 0;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: IndexedStack(
        index: _selectedIndex,
        children: [
          ShopScreen(repository: widget.repository),
          LibraryScreen(
            repository: widget.repository,
            progressStore: widget.progressStore,
            scoreStore: widget.scoreStore,
            settingsStore: widget.settingsStore,
            onThemeModeChanged: widget.onThemeModeChanged,
          ),
        ],
      ),
      bottomNavigationBar: NavigationBar(
        selectedIndex: _selectedIndex,
        onDestinationSelected: (index) =>
            setState(() => _selectedIndex = index),
        destinations: const [
          NavigationDestination(icon: Icon(Icons.storefront), label: 'Store'),
          NavigationDestination(
            icon: Icon(Icons.library_books),
            label: 'My Library',
          ),
        ],
      ),
    );
  }
}
