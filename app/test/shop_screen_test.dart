import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:aidoku/data/book_content_repository.dart';
import 'package:aidoku/screens/shop_screen.dart';

import 'fixtures/fake_book_content.dart';

void main() {
  testWidgets('lists every published book, unfiltered by language pair', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: ShopScreen(
          repository: BookContentRepository(
            client: fakeBookContentClient(books: [testBook, testBookOtherPair]),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    // testBook is en/ja, testBookOtherPair is ja/en - both should show
    // up regardless, unlike LibraryScreen's active-study-language filter.
    expect(find.text('Test Book'), findsOneWidget);
    expect(find.text('Other Pair Book'), findsOneWidget);
  });

  testWidgets('shows author and reading level per book', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: ShopScreen(
          repository: BookContentRepository(client: fakeBookContentClient()),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.textContaining('Test Author'), findsOneWidget);
    expect(find.textContaining('Level'), findsOneWidget);
  });

  testWidgets('a book with no cover image falls back to a placeholder icon', (
    WidgetTester tester,
  ) async {
    // fakeBookContentClient 404s /aidoku/book/{id}/image when [images]
    // has no entry for that id - same as a real book with no stored
    // cover (db/schema.sql's book_image is nullable).
    await tester.pumpWidget(
      MaterialApp(
        home: ShopScreen(
          repository: BookContentRepository(client: fakeBookContentClient()),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byIcon(Icons.menu_book), findsOneWidget);
  });

  testWidgets('a book with a stored cover renders it, fetched as bytes', (
    WidgetTester tester,
  ) async {
    // Proves the cover comes from BookContentRepository.getBookImage
    // (raw bytes over the repository's own client) rather than a URL
    // handed to Image.network - fakeBookContentClient only serves image
    // bytes through that same fake client, so a real network image
    // widget wouldn't see them here.
    await tester.pumpWidget(
      MaterialApp(
        home: ShopScreen(
          repository: BookContentRepository(
            client: fakeBookContentClient(
              images: {testBook['id'] as int: testImageBytes},
            ),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byType(Image), findsOneWidget);
    expect(find.byIcon(Icons.menu_book), findsNothing);
  });

  testWidgets('shows a corner ribbon naming each book\'s language', (
    WidgetTester tester,
  ) async {
    // Store's unfiltered by language pair (see the first test above),
    // so the ribbon is what tells a shopper which language a book is
    // actually written in - checked via Banner's own message property,
    // not find.text, since Banner paints its message directly rather
    // than via a Text widget.
    await tester.pumpWidget(
      MaterialApp(
        home: ShopScreen(
          repository: BookContentRepository(
            client: fakeBookContentClient(books: [testBook, testBookOtherPair]),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    final messages = tester
        .widgetList<Banner>(find.byType(Banner))
        .map((b) => b.message)
        // MaterialApp's own debug-mode banner ("DEBUG", top-right of the
        // whole window - see the screenshot this was modeled on) is
        // also a Banner; only the per-card ones are under test here.
        .where((m) => m != 'DEBUG')
        .toSet();
    // testBook's target_language is 'en', testBookOtherPair's is 'ja'.
    expect(messages, {'ENGLISH', 'JAPANESE'});
  });

  testWidgets('empty state when no books are published', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: ShopScreen(
          repository: BookContentRepository(
            client: fakeBookContentClient(books: []),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('No books published yet.'), findsOneWidget);
  });
}
