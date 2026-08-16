// Locks in the underline-in-passage behavior added for vocab/grammar
// questions (see ReadingView._highlightedSpans / Question.highlight). In
// its own file — see core_loop_test.dart's header comment re: running
// multiple tests in one file hitting an apparent flutter_test hang.

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:aidoku/data/book_content_repository.dart';
import 'package:aidoku/screens/library_screen.dart';

import 'fixtures/fake_book_content.dart';
import 'fixtures/fake_progress_store.dart';
import 'fixtures/fake_score_store.dart';
import 'fixtures/fake_settings_store.dart';

void main() {
  testWidgets(
    'the first vocab question underlines its target word in the passage',
    (WidgetTester tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: LibraryScreen(
            repository: BookContentRepository(client: fakeBookContentClient()),
            progressStore: FakeProgressStore(),
            scoreStore: FakeScoreStore(),
            settingsStore: FakeSettingsStore(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      await tester.tap(find.text('Test Book'));
      await tester.pumpAndSettle();
      await tester.tap(find.text("Continue"));
      await tester.pumpAndSettle();

      // Chunk 101's first question is the vocab question, highlighting
      // "acknowledged" — see test/fixtures/fake_book_content.dart.
      //
      // Text.rich wraps the TextSpan it's given in an extra outer TextSpan
      // (RichText(text: TextSpan(children: [theSpanWePassed]))), so the
      // spans ReadingView actually built are one level deeper than
      // richText.text.children itself.
      final richText = tester.widget<RichText>(
        find.byWidgetPredicate(
          (w) => w is RichText && w.text.toPlainText().contains('acknowledged'),
        ),
      );
      final wrapper = (richText.text as TextSpan).children!.single as TextSpan;
      final spans = wrapper.children!.cast<TextSpan>();
      final highlighted = spans.firstWhere((s) => s.text == 'acknowledged');

      expect(highlighted.style?.decoration, TextDecoration.underline);

      // Sanity check the rest of the passage isn't also underlined.
      final plainSpans = spans.where((s) => s.text != 'acknowledged');
      for (final span in plainSpans) {
        expect(span.style?.decoration, isNot(TextDecoration.underline));
      }
    },
  );
}
