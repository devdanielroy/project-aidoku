// ChunkListScreen in isolation: preview text + pass/fail badges derived
// from ScoreStore, and tapping a row opens a review session starting at
// that chunk specifically (not always the first). See chunk_review_test.dart
// for the full end-to-end flow (entry point gating, redoing a chunk,
// returning to the list).

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:aidoku/data/book_content_repository.dart';
import 'package:aidoku/models/book.dart';
import 'package:aidoku/screens/chunk_list_screen.dart';

import 'fixtures/fake_book_content.dart';
import 'fixtures/fake_score_store.dart';

void main() {
  final book = Book.fromJson(testBook);

  testWidgets("shows each reviewable chunk's preview and pass/fail badge", (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: ChunkListScreen(
          book: book,
          repository: BookContentRepository(client: fakeBookContentClient()),
          scoreStore: FakeScoreStore(
            seed: {
              1: {
                1001: true, 1002: true, 1003: true, // chunk 101: all correct
                1004: true, 1005: false, 1006: true, // chunk 102: one wrong
                // chunk 103's questions (1007-1009) are deliberately
                // absent - unanswered, not failed.
              },
            },
          ),
          chunkIds: const [101, 102, 103],
        ),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('Chunk 1'), findsOneWidget);
    expect(find.text('Chunk 2'), findsOneWidget);
    expect(find.text('Chunk 3'), findsOneWidget);
    expect(find.textContaining('must be in want of a wife'), findsOneWidget);

    // Exactly one pass, one fail - chunk 103 (unanswered) gets neither.
    expect(find.byIcon(Icons.check_circle), findsOneWidget);
    expect(find.byIcon(Icons.cancel), findsOneWidget);
  });

  testWidgets('tapping a chunk opens a review session starting at that chunk', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: ChunkListScreen(
          book: book,
          repository: BookContentRepository(client: fakeBookContentClient()),
          scoreStore: FakeScoreStore(
            seed: {
              1: {1004: true, 1005: true, 1006: true},
            },
          ),
          chunkIds: const [101, 102],
        ),
      ),
    );
    await tester.pumpAndSettle();

    // Tap chunk 2's row, not chunk 1's.
    await tester.tap(find.text('Chunk 2'));
    await tester.pumpAndSettle();

    // Opened at position 1 (chunk 102) - its app bar and passage, not
    // chunk 101's.
    expect(find.text('Chunk 2 of 2'), findsOneWidget);
    expect(find.textContaining('mean understanding'), findsOneWidget);
  });
}
