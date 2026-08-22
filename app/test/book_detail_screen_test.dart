// BookDetailScreen in isolation - the Store's per-book page. See its
// own doc comment for which parts are real data vs. placeholders
// (genre tags and summary are real, hand-curated book data; price and
// the buy buttons are still placeholders).

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:aidoku/data/book_content_repository.dart';
import 'package:aidoku/models/book.dart';
import 'package:aidoku/screens/book_detail_screen.dart';
import 'package:aidoku/theme/accent_colors.dart';

import 'fixtures/fake_book_content.dart';

void main() {
  final book = Book.fromJson(testBook);

  Future<void> pump(WidgetTester tester) => tester.pumpWidget(
    MaterialApp(
      home: BookDetailScreen(
        book: book,
        repository: BookContentRepository(client: fakeBookContentClient()),
      ),
    ),
  );

  testWidgets('shows title and author', (WidgetTester tester) async {
    await pump(tester);
    await tester.pumpAndSettle();

    // "Test Book" appears twice - once in the AppBar, once as the
    // headline next to the cover - so this only checks the latter.
    expect(find.text('Test Book'), findsWidgets);
    expect(find.text('By Test Author'), findsOneWidget);
  });

  testWidgets(
    'shows a placeholder price and three inert acquisition buttons',
    (WidgetTester tester) async {
      // Pricing/purchasing aren't built yet (see README's Shop
      // milestone) - $0.00 and three buttons that exist but do nothing
      // when tapped, Free Sample included (still just the shop's
      // facade - see _PurchaseColumn's own doc comment).
      await pump(tester);
      await tester.pumpAndSettle();

      expect(find.text('\$0.00'), findsOneWidget);
      expect(find.widgetWithText(FilledButton, 'Buy Now'), findsOneWidget);
      expect(
        find.widgetWithText(OutlinedButton, 'Add to Cart'),
        findsOneWidget,
      );
      expect(
        find.widgetWithText(FilledButton, 'Free Sample'),
        findsOneWidget,
      );
      // Free Sample is styled distinctly from Buy Now (same baby blue
      // as ShopScreen's language ribbon - see accent_colors.dart), not
      // just plain-text, so it doesn't disappear next to the other two.
      final freeSampleButton = tester.widget<FilledButton>(
        find.widgetWithText(FilledButton, 'Free Sample'),
      );
      expect(
        freeSampleButton.style?.backgroundColor?.resolve({}),
        babyBlueAccent,
      );

      // Tapping doesn't throw and doesn't navigate anywhere.
      await tester.tap(find.text('Buy Now'));
      await tester.tap(find.text('Add to Cart'));
      await tester.tap(find.text('Free Sample'));
      await tester.pumpAndSettle();
      expect(find.byType(BookDetailScreen), findsOneWidget);
    },
  );

  testWidgets('shows language, reading level, and chunk count', (
    WidgetTester tester,
  ) async {
    await pump(tester);
    await tester.pumpAndSettle();

    // testBook's target_language is 'en', level 5 - see fake_book_content.
    expect(find.text('English'), findsOneWidget);
    expect(find.text(book.readingLevelName), findsOneWidget);
    // testChunks has 3 entries.
    expect(find.text('3 chunks'), findsOneWidget);
  });

  testWidgets("shows the book's real genre tags", (WidgetTester tester) async {
    // testBook's genres (see fake_book_content.dart): "Fiction, Gothic,
    // Horror, Classic" - already split/trimmed by Book.fromJson.
    await pump(tester);
    await tester.pumpAndSettle();

    expect(find.text('Fiction'), findsOneWidget);
    expect(find.text('Gothic'), findsOneWidget);
    expect(find.text('Horror'), findsOneWidget);
    expect(find.text('Classic'), findsOneWidget);
  });

  testWidgets("shows the book's real summary", (WidgetTester tester) async {
    await pump(tester);
    await tester.pumpAndSettle();

    expect(find.text('Summary'), findsOneWidget);
    // testBook's summary (see fake_book_content.dart).
    expect(find.text('A test summary.'), findsOneWidget);
  });
}
