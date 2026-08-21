// AidokuApp's theme-mode wiring end to end: it loads the saved
// ThemeModeSetting on startup and reacts live when SettingsScreen saves
// a new one, without needing to be rebuilt from scratch — see its own
// doc comments (main.dart) for why that needs a callback threaded all
// the way down through LibraryScreen rather than a plain setState() a
// few widgets below the root.

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:aidoku/data/book_content_repository.dart';
import 'package:aidoku/main.dart';
import 'package:aidoku/models/user_settings.dart';

import 'fixtures/fake_book_content.dart';
import 'fixtures/fake_progress_store.dart';
import 'fixtures/fake_score_store.dart';
import 'fixtures/fake_settings_store.dart';

void main() {
  AidokuApp app({UserSettings? seed}) => AidokuApp(
    settingsStore: FakeSettingsStore(seed: seed),
    repository: BookContentRepository(client: fakeBookContentClient()),
    progressStore: FakeProgressStore(),
    scoreStore: FakeScoreStore(),
  );

  MaterialApp materialApp(WidgetTester tester) =>
      tester.widget<MaterialApp>(find.byType(MaterialApp));

  testWidgets('defaults to ThemeMode.system before any settings are saved', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(app());
    await tester.pumpAndSettle();

    expect(materialApp(tester).themeMode, ThemeMode.system);
  });

  testWidgets('loads a previously saved dark preference on startup', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(
      app(seed: const UserSettings(themeMode: ThemeModeSetting.dark)),
    );
    await tester.pumpAndSettle();

    expect(materialApp(tester).themeMode, ThemeMode.dark);
  });

  testWidgets(
    'saving a new theme mode from Settings re-themes the running app live',
    (WidgetTester tester) async {
      await tester.pumpWidget(app());
      await tester.pumpAndSettle();
      expect(materialApp(tester).themeMode, ThemeMode.system);

      await tester.tap(find.byIcon(Icons.settings));
      await tester.pumpAndSettle();

      await tester.tap(find.text('Dark'));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Save'));
      await tester.pumpAndSettle();

      // Back on Library, no restart — the same MaterialApp instance
      // now reports the new mode.
      expect(materialApp(tester).themeMode, ThemeMode.dark);
    },
  );
}
