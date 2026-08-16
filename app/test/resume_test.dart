// Verifies ReadingSessionScreen actually uses ProgressStore to resume at
// a previously-saved chunk instead of always starting over, and clears
// progress once the book is finished. See progress_store_test.dart for
// LocalProgressStore's get/save/clear behavior in isolation; this is the
// screen-level wiring on top of it. In its own file — see
// core_loop_test.dart's header comment re: multiple tests in one file
// hitting an apparent flutter_test hang.

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:aidoku/data/book_content_repository.dart';
import 'package:aidoku/screens/library_screen.dart';

import 'fixtures/fake_book_content.dart';
import 'fixtures/fake_progress_store.dart';
import 'fixtures/fake_score_store.dart';
import 'fixtures/fake_settings_store.dart';

void main() {
  testWidgets('resumes at the saved chunk instead of starting over', (
    WidgetTester tester,
  ) async {
    // testBook's id is 1 (see fake_book_content.dart) - seed it as
    // already on chunk index 1 (the 2nd of the 3 test chunks).
    final progressStore = FakeProgressStore(seed: {1: 1});

    await tester.pumpWidget(
      MaterialApp(
        home: LibraryScreen(
          repository: BookContentRepository(client: fakeBookContentClient()),
          progressStore: progressStore,
          scoreStore: FakeScoreStore(),
          settingsStore: FakeSettingsStore(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Test Book'));
    await tester.pumpAndSettle();

    // Straight to chunk 2, not chunk 1.
    expect(find.text('CHUNK 2 OF 3'), findsOneWidget);
    expect(find.text('CHUNK 1 OF 3'), findsNothing);
  });

  testWidgets(
    'a saved index past the end of the book falls back to the start',
    (WidgetTester tester) async {
      // The book only has 3 chunks (indices 0-2) - a stale/bad save
      // shouldn't crash on an out-of-range list index.
      final progressStore = FakeProgressStore(seed: {1: 99});

      await tester.pumpWidget(
        MaterialApp(
          home: LibraryScreen(
            repository: BookContentRepository(client: fakeBookContentClient()),
            progressStore: progressStore,
            scoreStore: FakeScoreStore(),
            settingsStore: FakeSettingsStore(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('Test Book'));
      await tester.pumpAndSettle();

      expect(find.text('CHUNK 1 OF 3'), findsOneWidget);
    },
  );

  testWidgets('finishing the book clears its saved progress', (
    WidgetTester tester,
  ) async {
    // Resume straight onto the last chunk, so finishing it only takes
    // one chunk's worth of interaction.
    final progressStore = FakeProgressStore(seed: {1: 2});

    await tester.pumpWidget(
      MaterialApp(
        home: LibraryScreen(
          repository: BookContentRepository(client: fakeBookContentClient()),
          progressStore: progressStore,
          scoreStore: FakeScoreStore(),
          settingsStore: FakeSettingsStore(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('Test Book'));
    await tester.pumpAndSettle();
    expect(find.text('CHUNK 3 OF 3'), findsOneWidget);

    await tester.tap(find.text('Continue'));
    await tester.pumpAndSettle();

    for (var i = 0; i < 3; i++) {
      final correctOption = find.text(chunk103CorrectAnswers[i]);
      await tester.ensureVisible(correctOption);
      await tester.tap(correctOption);
      await tester.pumpAndSettle();
      final buttonText = i == 2 ? 'See the full breakdown' : 'Next question';
      await tester.tap(find.text(buttonText));
      await tester.pumpAndSettle();
    }

    await tester.tap(find.text('Finish this excerpt'));
    await tester.pumpAndSettle();

    expect(await progressStore.getChunkIndex(1), isNull);
  });
}
