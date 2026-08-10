// Exercises the core loop end to end (AIDOKU_DESIGN.md §2): pick a book,
// read a chunk unassisted, answer its 3 questions, see the breakdown,
// advance to the next chunk. In its own file (rather than alongside
// library_screen_test.dart) because running both in one file/isolate hits
// an apparent flutter_test cross-test hang on the second test's initial
// pumpAndSettle — root cause not tracked down, but each test is solid in
// isolation and one-test-per-file is a reasonable structure regardless.

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:aidoku/main.dart';

void main() {
  testWidgets(
    'full core loop: read -> answer 3 questions -> breakdown -> next chunk',
    (WidgetTester tester) async {
      await tester.pumpWidget(const AidokuApp());
      await tester.pumpAndSettle();

      // Pick the book.
      await tester.tap(find.text('Pride and Prejudice'));
      await tester.pumpAndSettle();

      // Reading phase: chunk 1's text is shown, no questions yet.
      expect(find.text('CHUNK 1 OF 3'), findsOneWidget);
      // "must be in want of a wife" (rather than the opening clause) —
      // the breakdown's own explanation separately quotes "It is a truth
      // universally acknowledged, that ..." and "must", so searching for
      // either of those would ambiguously match both the passage and the
      // breakdown text once the breakdown is showing.
      expect(find.textContaining('must be in want of a wife'), findsOneWidget);
      await tester.tap(find.text("I've read it — continue"));
      await tester.pumpAndSettle();

      // The passage stays visible (shrunk into the top pane) once the
      // questions sheet is up — it's meant to never disappear.
      expect(find.textContaining('must be in want of a wife'), findsOneWidget);

      // Questions phase: answer all 3 questions for chunk 1. The correct
      // option is always index 0 in the mock data, so tapping the first
      // option tile answers correctly every time.
      for (var i = 0; i < 3; i++) {
        expect(find.text('Question ${i + 1} of 3'), findsOneWidget);
        await tester.tap(find.byType(InkWell).first);
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
    },
  );
}
