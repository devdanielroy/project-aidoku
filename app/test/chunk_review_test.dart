// Verifies the chunk review flow end to end: the entry point stays
// hidden until something's cleared, the list shows only cleared chunks
// with a pass/fail badge, a failed chunk is a real interactive redo
// (and flips the badge on success) while a passed chunk instead opens
// pre-answered — nothing to retap, see QuestionsView.startAnswered — and
// none of this ever touches ProgressStore. See chunk_list_screen_test.dart
// for the list screen in isolation and questions_view_test.dart for the
// pre-answered rendering in isolation. In its own file — see
// core_loop_test.dart's header comment re: multiple tests in one file
// hitting an apparent flutter_test hang.

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:aidoku/data/book_content_repository.dart';
import 'package:aidoku/screens/library_screen.dart';

import 'fixtures/fake_book_content.dart';
import 'fixtures/fake_progress_store.dart';
import 'fixtures/fake_score_store.dart';

const _wrongAnswers = ['denied', 'obligation', 'a legal requirement'];

void main() {
  testWidgets(
    'review: gated until cleared; a failed chunk redoes interactively and flips to passed; '
    'a passed chunk then opens pre-answered instead of asking for a retap',
    (WidgetTester tester) async {
      final progressStore = FakeProgressStore();
      final scoreStore = FakeScoreStore();

      await tester.pumpWidget(
        MaterialApp(
          home: LibraryScreen(
            repository: BookContentRepository(client: fakeBookContentClient()),
            progressStore: progressStore,
            scoreStore: scoreStore,
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('Test Book'));
      await tester.pumpAndSettle();

      // Still on chunk 1, nothing cleared yet - no review entry point.
      expect(find.byTooltip('Review cleared chunks'), findsNothing);

      // Clear chunk 1, answering everything wrong on purpose.
      await tester.tap(find.text('Continue'));
      await tester.pumpAndSettle();
      for (var i = 0; i < 3; i++) {
        final wrongOption = find.text(_wrongAnswers[i]);
        await tester.ensureVisible(wrongOption);
        await tester.tap(wrongOption);
        await tester.pumpAndSettle();
        final buttonText = i == 2 ? 'See the full breakdown' : 'Next question';
        await tester.tap(find.text(buttonText));
        await tester.pumpAndSettle();
      }
      await tester.tap(find.text('Next chunk'));
      await tester.pumpAndSettle();

      // Now on chunk 2, with chunk 1 cleared (but failed) - entry point
      // available.
      expect(find.text('CHUNK 2 OF 3'), findsOneWidget);
      expect(find.byTooltip('Review cleared chunks'), findsOneWidget);

      await tester.tap(find.byTooltip('Review cleared chunks'));
      await tester.pumpAndSettle();

      expect(find.text('Chunk 1'), findsOneWidget);
      expect(find.byIcon(Icons.cancel), findsOneWidget);
      expect(find.byIcon(Icons.check_circle), findsNothing);

      // Redo it - a failed chunk is a real interactive retry, not
      // pre-filled.
      await tester.tap(find.text('Chunk 1'));
      await tester.pumpAndSettle();
      expect(find.text('Chunk 1 of 1'), findsOneWidget);

      await tester.tap(find.text('Continue'));
      await tester.pumpAndSettle();
      // Nothing pre-selected - the advance button isn't there yet.
      expect(find.text('Next question'), findsNothing);

      for (var i = 0; i < 3; i++) {
        final correctOption = find.text(chunk101CorrectAnswers[i]);
        await tester.ensureVisible(correctOption);
        await tester.tap(correctOption);
        await tester.pumpAndSettle();
        final buttonText = i == 2 ? 'See the full breakdown' : 'Next question';
        await tester.tap(find.text(buttonText));
        await tester.pumpAndSettle();
      }
      await tester.tap(find.text('Finish this excerpt'));
      await tester.pumpAndSettle();

      // Back on the list automatically - the redo flipped the badge.
      expect(find.text('Review: Test Book'), findsOneWidget);
      expect(find.byIcon(Icons.check_circle), findsOneWidget);
      expect(find.byIcon(Icons.cancel), findsNothing);

      // Open it again - now passed, so this time it should be
      // pre-answered rather than asking for a retap.
      await tester.tap(find.text('Chunk 1'));
      await tester.pumpAndSettle();

      await tester.tap(find.text('Continue'));
      await tester.pumpAndSettle();
      // Already answered, with no tap on any option - the advance
      // button is there immediately.
      expect(find.text('Next question'), findsOneWidget);

      for (var i = 0; i < 3; i++) {
        final buttonText = i == 2 ? 'See the full breakdown' : 'Next question';
        await tester.tap(find.text(buttonText));
        await tester.pumpAndSettle();
      }
      await tester.tap(find.text('Finish this excerpt'));
      await tester.pumpAndSettle();

      // Still back on the list, badge unchanged - nothing was actually
      // re-answered, so nothing was re-recorded.
      expect(find.text('Review: Test Book'), findsOneWidget);
      expect(find.byIcon(Icons.check_circle), findsOneWidget);
      expect(find.byIcon(Icons.cancel), findsNothing);

      // None of this ever touched the real resume bookmark.
      expect(await progressStore.getChunkIndex(1), 1);
    },
  );
}
