// Exercises the core loop end to end (AIDOKU_DESIGN.md §2): pick a book,
// read a chunk unassisted, answer its 3 questions, see the breakdown,
// advance to the next chunk. In its own file (rather than alongside
// library_screen_test.dart) because running both in one file/isolate hits
// an apparent flutter_test cross-test hang on the second test's initial
// pumpAndSettle — root cause not tracked down, but each test is solid in
// isolation and one-test-per-file is a reasonable structure regardless.

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:aidoku/data/book_content_repository.dart';
import 'package:aidoku/screens/library_screen.dart';

import 'fixtures/fake_book_content.dart';
import 'fixtures/fake_progress_store.dart';

void main() {
  testWidgets(
    'full core loop: read -> answer 3 questions -> breakdown -> next chunk',
    (WidgetTester tester) async {
      final progressStore = FakeProgressStore();
      await tester.pumpWidget(
        MaterialApp(
          home: LibraryScreen(
            repository: BookContentRepository(client: fakeBookContentClient()),
            progressStore: progressStore,
          ),
        ),
      );
      await tester.pumpAndSettle();

      // Pick the book.
      await tester.tap(find.text('Test Book'));
      await tester.pumpAndSettle();

      // Reading phase: chunk 1's text is shown, no questions yet.
      expect(find.text('CHUNK 1 OF 3'), findsOneWidget);
      // "must be in want of a wife" (rather than the opening clause) —
      // the breakdown's own explanation separately quotes "It is a truth
      // universally acknowledged, that ..." and "must", so searching for
      // either of those would ambiguously match both the passage and the
      // breakdown text once the breakdown is showing.
      expect(find.textContaining('must be in want of a wife'), findsOneWidget);
      await tester.tap(find.text("Continue"));
      await tester.pumpAndSettle();

      // The passage stays visible (shrunk into the top pane) once the
      // questions sheet is up — it's meant to never disappear.
      expect(find.textContaining('must be in want of a wife'), findsOneWidget);

      // Questions phase: answer all 3 questions for chunk 1. Options are
      // shuffled for display (see QuestionsView), so tap by the known
      // correct answer's text, not by position.
      for (var i = 0; i < 3; i++) {
        expect(find.text('Question ${i + 1} of 3'), findsOneWidget);
        // Options are shuffled for display (see QuestionsView), so the
        // correct answer can land anywhere — including scrolled out of
        // the sheet's fixed-height view — unlike always tapping the
        // first option.
        final correctOption = find.text(chunk101CorrectAnswers[i]);
        await tester.ensureVisible(correctOption);
        await tester.tap(correctOption);
        await tester.pumpAndSettle();
        final buttonText = i == 2 ? 'See the full breakdown' : 'Next question';
        await tester.tap(find.text(buttonText));
        await tester.pumpAndSettle();
      }

      // Breakdown phase — the passage is still visible here too, just
      // squeezed smaller, per the design ask that it never disappear.
      expect(find.text('BREAKDOWN'), findsOneWidget);
      expect(find.text('Next chunk'), findsOneWidget);
      expect(find.textContaining('must be in want of a wife'), findsOneWidget);
      await tester.tap(find.text('Next chunk'));
      await tester.pumpAndSettle();

      // Back to reading phase, now on chunk 2.
      expect(find.text('CHUNK 2 OF 3'), findsOneWidget);

      // Progress was saved on advancing — see progress_store_test.dart
      // for saveChunkIndex/getChunkIndex/clearProgress in isolation;
      // this just confirms ReadingSessionScreen actually calls it.
      expect(await progressStore.getChunkIndex(1), 1);
    },
  );
}
