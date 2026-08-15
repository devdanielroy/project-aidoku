import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:aidoku/data/book_content_repository.dart';
import 'package:aidoku/screens/library_screen.dart';

import 'fixtures/fake_book_content.dart';
import 'fixtures/fake_progress_store.dart';
import 'fixtures/fake_score_store.dart';

void main() {
  testWidgets('library screen shows a published book from book-content', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: LibraryScreen(
          repository: BookContentRepository(client: fakeBookContentClient()),
          progressStore: FakeProgressStore(),
          scoreStore: FakeScoreStore(),
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
          scoreStore: FakeScoreStore(),
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
          scoreStore: FakeScoreStore(),
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

  testWidgets('a book with no recorded answers shows no accuracy line', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: LibraryScreen(
          repository: BookContentRepository(client: fakeBookContentClient()),
          progressStore: FakeProgressStore(),
          scoreStore: FakeScoreStore(),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.textContaining('Accuracy'), findsNothing);
  });

  testWidgets('a book with recorded answers shows its accuracy', (
    WidgetTester tester,
  ) async {
    // testBook's id is 1 - 3 answers recorded, 2 correct.
    await tester.pumpWidget(
      MaterialApp(
        home: LibraryScreen(
          repository: BookContentRepository(client: fakeBookContentClient()),
          progressStore: FakeProgressStore(),
          scoreStore: FakeScoreStore(
            seed: {
              1: {101: true, 102: true, 103: false},
            },
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Accuracy: 67% (2/3)'), findsOneWidget);
  });

  testWidgets(
    'the card refreshes when the reader backs out of the book, not just on first load',
    (WidgetTester tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: LibraryScreen(
            repository: BookContentRepository(client: fakeBookContentClient()),
            progressStore: FakeProgressStore(),
            scoreStore: FakeScoreStore(),
          ),
        ),
      );
      await tester.pumpAndSettle();

      // Nothing yet - the card's futures were computed once, before any
      // of this happened.
      expect(find.textContaining('Chunk'), findsNothing);
      expect(find.textContaining('Accuracy'), findsNothing);

      await tester.tap(find.text('Test Book'));
      await tester.pumpAndSettle();

      await tester.tap(find.text('Continue'));
      await tester.pumpAndSettle();
      for (var i = 0; i < 3; i++) {
        final correctOption = find.text(chunk101CorrectAnswers[i]);
        await tester.ensureVisible(correctOption);
        await tester.tap(correctOption);
        await tester.pumpAndSettle();
        final buttonText = i == 2 ? 'See the full breakdown' : 'Next question';
        await tester.tap(find.text(buttonText));
        await tester.pumpAndSettle();
      }
      await tester.tap(find.text('Next chunk'));
      await tester.pumpAndSettle();

      // Back to the library via the app bar's back button, not a
      // programmatic pop - same as a reader would actually do it.
      await tester.pageBack();
      await tester.pumpAndSettle();

      expect(find.text('Chunk 2 of 3'), findsOneWidget);
      expect(find.text('Accuracy: 100% (3/3)'), findsOneWidget);
    },
  );
}
