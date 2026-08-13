import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:aidoku/data/book_content_repository.dart';
import 'package:aidoku/screens/library_screen.dart';

import 'fixtures/fake_book_content.dart';
import 'fixtures/fake_progress_store.dart';

void main() {
  testWidgets('library screen shows a published book from book-content', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: LibraryScreen(
          repository: BookContentRepository(client: fakeBookContentClient()),
          progressStore: FakeProgressStore(),
        ),
      ),
    );
    await tester.pumpAndSettle(); // wait for the fake async "network" load

    expect(find.text('Test Book'), findsOneWidget);
  });

  testWidgets('a book with no saved progress shows no progress indicator', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: LibraryScreen(
          repository: BookContentRepository(client: fakeBookContentClient()),
          progressStore: FakeProgressStore(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.textContaining('Chunk'), findsNothing);
    expect(find.byType(LinearProgressIndicator), findsNothing);
  });

  testWidgets('a book with saved progress shows a chunk N of M indicator', (
    WidgetTester tester,
  ) async {
    // testBook's id is 1, with 3 chunks (see fake_book_content.dart) -
    // seeded partway through, at chunk index 1 (the 2nd of 3).
    await tester.pumpWidget(
      MaterialApp(
        home: LibraryScreen(
          repository: BookContentRepository(client: fakeBookContentClient()),
          progressStore: FakeProgressStore(seed: {1: 1}),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Chunk 2 of 3'), findsOneWidget);
    final progressBar = tester.widget<LinearProgressIndicator>(
      find.byType(LinearProgressIndicator),
    );
    expect(progressBar.value, 1 / 3);
  });
}
