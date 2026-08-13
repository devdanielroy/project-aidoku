import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:aidoku/data/book_content_repository.dart';
import 'package:aidoku/screens/library_screen.dart';

import 'fixtures/fake_book_content.dart';

void main() {
  testWidgets('library screen shows a published book from book-content', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: LibraryScreen(
          repository: BookContentRepository(client: fakeBookContentClient()),
        ),
      ),
    );
    await tester.pumpAndSettle(); // wait for the fake async "network" load

    expect(find.text('Test Book'), findsOneWidget);
  });
}
