import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:aidoku/data/book_content_repository.dart';
import 'package:aidoku/models/user_settings.dart';
import 'package:aidoku/screens/settings_screen.dart';

import 'fixtures/fake_book_content.dart';
import 'fixtures/fake_settings_store.dart';

void main() {
  // testBook is en/ja, testBookOtherPair is ja/en — together they give
  // every test here two native-language options (en, ja) to work with,
  // not just the one pair most other screens' tests get away with.
  BookContentRepository twoPairRepository() => BookContentRepository(
    client: fakeBookContentClient(books: [testBook, testBookOtherPair]),
  );

  testWidgets('shows every native language available across published books', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: SettingsScreen(
          repository: twoPairRepository(),
          settingsStore: FakeSettingsStore(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byType(DropdownButtonFormField<String>));
    await tester.pumpAndSettle();

    expect(find.text('English').hitTestable(), findsOneWidget);
    expect(find.text('Japanese').hitTestable(), findsOneWidget);
  });

  testWidgets(
    'picking a native language shows only that pair\'s study language',
    (WidgetTester tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: SettingsScreen(
            repository: twoPairRepository(),
            settingsStore: FakeSettingsStore(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      // Nothing to study yet - no native language picked.
      expect(find.text('Pick a native language first.'), findsOneWidget);

      await tester.tap(find.byType(DropdownButtonFormField<String>));
      await tester.pumpAndSettle();
      await tester.tap(find.text('Japanese').last);
      await tester.pumpAndSettle();

      // Native = Japanese -> the only matching pair studies English.
      expect(find.text('English'), findsOneWidget);
      expect(
        find.text('Japanese'),
        findsOneWidget,
      ); // the dropdown's own selection
    },
  );

  testWidgets('enrolling in a study language auto-activates it', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: SettingsScreen(
          repository: twoPairRepository(),
          settingsStore: FakeSettingsStore(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byType(DropdownButtonFormField<String>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Japanese').last);
    await tester.pumpAndSettle();

    expect(find.text('Active'), findsNothing);
    await tester.tap(find.byType(Checkbox));
    await tester.pumpAndSettle();
    expect(find.text('Active'), findsOneWidget);
  });

  testWidgets('save persists username, native, and active language', (
    WidgetTester tester,
  ) async {
    final store = FakeSettingsStore();
    await tester.pumpWidget(
      MaterialApp(
        home: SettingsScreen(
          repository: twoPairRepository(),
          settingsStore: store,
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField), 'Roi');
    await tester.tap(find.byType(DropdownButtonFormField<String>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Japanese').last);
    await tester.pumpAndSettle();
    await tester.tap(find.byType(Checkbox));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    final saved = await store.getSettings();
    expect(saved.username, 'Roi');
    expect(saved.nativeLanguage, 'ja');
    expect(saved.studyLanguages, ['en']);
    expect(saved.activeStudyLanguage, 'en');
  });

  testWidgets('save pops the screen', (WidgetTester tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Builder(
          builder: (context) => Scaffold(
            body: Center(
              child: FilledButton(
                onPressed: () => Navigator.of(context).push(
                  MaterialPageRoute(
                    builder: (_) => SettingsScreen(
                      repository: twoPairRepository(),
                      settingsStore: FakeSettingsStore(),
                    ),
                  ),
                ),
                child: const Text('open'),
              ),
            ),
          ),
        ),
      ),
    );
    await tester.tap(find.text('open'));
    await tester.pumpAndSettle();
    expect(find.byType(SettingsScreen), findsOneWidget);
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect(find.byType(SettingsScreen), findsNothing);
  });

  testWidgets('pre-fills from existing settings', (WidgetTester tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: SettingsScreen(
          repository: twoPairRepository(),
          settingsStore: FakeSettingsStore(
            seed: const UserSettings(
              username: 'Roi',
              nativeLanguage: 'ja',
              studyLanguages: ['en'],
              activeStudyLanguage: 'en',
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Roi'), findsOneWidget);
    expect(find.text('Active'), findsOneWidget);
  });

  testWidgets('defaults to the System theme segment selected', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: SettingsScreen(
          repository: twoPairRepository(),
          settingsStore: FakeSettingsStore(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final segmented = tester.widget<SegmentedButton<ThemeModeSetting>>(
      find.byType(SegmentedButton<ThemeModeSetting>),
    );
    expect(segmented.selected, {ThemeModeSetting.system});
  });

  testWidgets('pre-fills the theme segment from existing settings', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: SettingsScreen(
          repository: twoPairRepository(),
          settingsStore: FakeSettingsStore(
            seed: const UserSettings(themeMode: ThemeModeSetting.dark),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final segmented = tester.widget<SegmentedButton<ThemeModeSetting>>(
      find.byType(SegmentedButton<ThemeModeSetting>),
    );
    expect(segmented.selected, {ThemeModeSetting.dark});
  });

  testWidgets('picking a theme segment and saving persists it', (
    WidgetTester tester,
  ) async {
    final store = FakeSettingsStore();
    await tester.pumpWidget(
      MaterialApp(
        home: SettingsScreen(
          repository: twoPairRepository(),
          settingsStore: store,
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Dark'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect((await store.getSettings()).themeMode, ThemeModeSetting.dark);
  });

  testWidgets('save calls onThemeModeChanged with the saved mode', (
    WidgetTester tester,
  ) async {
    ThemeModeSetting? notified;
    await tester.pumpWidget(
      MaterialApp(
        home: SettingsScreen(
          repository: twoPairRepository(),
          settingsStore: FakeSettingsStore(),
          onThemeModeChanged: (mode) => notified = mode,
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Light'));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect(notified, ThemeModeSetting.light);
  });

  testWidgets('reading level defaults to unset', (WidgetTester tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: SettingsScreen(
          repository: twoPairRepository(),
          settingsStore: FakeSettingsStore(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final dropdown = tester.widget<DropdownButtonFormField<int?>>(
      find.byType(DropdownButtonFormField<int?>),
    );
    expect(dropdown.initialValue, isNull);
  });

  testWidgets('pre-fills the reading level from existing settings', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: SettingsScreen(
          repository: twoPairRepository(),
          settingsStore: FakeSettingsStore(
            seed: const UserSettings(readingLevel: 5),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final dropdown = tester.widget<DropdownButtonFormField<int?>>(
      find.byType(DropdownButtonFormField<int?>),
    );
    expect(dropdown.initialValue, 5);
  });

  testWidgets('picking a reading level and saving persists it', (
    WidgetTester tester,
  ) async {
    final store = FakeSettingsStore();
    await tester.pumpWidget(
      MaterialApp(
        home: SettingsScreen(
          repository: twoPairRepository(),
          settingsStore: store,
        ),
      ),
    );
    await tester.pumpAndSettle();

    // ensureVisible can leave a widget this close to the reading-level
    // dropdown's own height (56px) right at the clipped viewport edge,
    // still not hit-testable — a full drag to the scrollable's actual
    // end is unambiguous.
    await tester.drag(find.byType(ListView), const Offset(0, -1000));
    await tester.pumpAndSettle();
    await tester.tap(find.byType(DropdownButtonFormField<int?>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Bookworm (5)').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect((await store.getSettings()).readingLevel, 5);
  });

  testWidgets('reading level can be set back to unset after being picked', (
    WidgetTester tester,
  ) async {
    final store = FakeSettingsStore();
    await tester.pumpWidget(
      MaterialApp(
        home: SettingsScreen(
          repository: twoPairRepository(),
          settingsStore: store,
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.drag(find.byType(ListView), const Offset(0, -1000));
    await tester.pumpAndSettle();
    await tester.tap(find.byType(DropdownButtonFormField<int?>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Bookworm (5)').last);
    await tester.pumpAndSettle();

    // "Prefer not to say" is a real item, not just a hint — it should
    // still be choosable after a real level was already picked.
    await tester.tap(find.byType(DropdownButtonFormField<int?>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Prefer not to say').last);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();

    expect((await store.getSettings()).readingLevel, isNull);
  });

  testWidgets(
    'changing native language clears previously enrolled study languages',
    (WidgetTester tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: SettingsScreen(
            repository: twoPairRepository(),
            settingsStore: FakeSettingsStore(
              seed: const UserSettings(
                nativeLanguage: 'ja',
                studyLanguages: ['en'],
                activeStudyLanguage: 'en',
              ),
            ),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(find.text('Active'), findsOneWidget);

      // Switch native language from Japanese to English.
      await tester.tap(find.byType(DropdownButtonFormField<String>));
      await tester.pumpAndSettle();
      await tester.tap(find.text('English').last);
      await tester.pumpAndSettle();

      // Now studying Japanese is the only option, and it's unenrolled -
      // the prior English enrollment doesn't carry over.
      expect(find.text('Active'), findsNothing);
      final checkbox = tester.widget<Checkbox>(find.byType(Checkbox));
      expect(checkbox.value, isFalse);
    },
  );
}
