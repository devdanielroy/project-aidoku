// LocalScoreStore in isolation, against real shared_preferences (its
// platform channel mocked via setMockInitialValues — same approach as
// progress_store_test.dart, its sibling). See core_loop_test.dart /
// resume_test.dart for ScoreStore actually wired into the reading flow.

import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:aidoku/data/local_score_store.dart';

void main() {
  setUp(() {
    SharedPreferences.setMockInitialValues({});
  });

  test('getAnswers is empty when nothing has been recorded', () async {
    final store = LocalScoreStore();
    expect(await store.getAnswers(1), isEmpty);
  });

  test('recordAnswer then getAnswers round-trips', () async {
    final store = LocalScoreStore();
    await store.recordAnswer(bookId: 1, questionId: 101, correct: true);
    expect(await store.getAnswers(1), {101: true});
  });

  test(
    'recordAnswer accumulates multiple questions for the same book',
    () async {
      final store = LocalScoreStore();
      await store.recordAnswer(bookId: 1, questionId: 101, correct: true);
      await store.recordAnswer(bookId: 1, questionId: 102, correct: false);
      expect(await store.getAnswers(1), {101: true, 102: false});
    },
  );

  test(
    'recordAnswer overwrites a previous result for the same question',
    () async {
      final store = LocalScoreStore();
      await store.recordAnswer(bookId: 1, questionId: 101, correct: false);
      await store.recordAnswer(bookId: 1, questionId: 101, correct: true);
      expect(await store.getAnswers(1), {101: true});
    },
  );

  test('answers for different books are independent', () async {
    final store = LocalScoreStore();
    await store.recordAnswer(bookId: 1, questionId: 101, correct: true);
    await store.recordAnswer(bookId: 2, questionId: 201, correct: false);
    expect(await store.getAnswers(1), {101: true});
    expect(await store.getAnswers(2), {201: false});
  });

  test('clearScore removes all recorded answers for a book', () async {
    final store = LocalScoreStore();
    await store.recordAnswer(bookId: 1, questionId: 101, correct: true);
    await store.recordAnswer(bookId: 1, questionId: 102, correct: false);
    await store.clearScore(1);
    expect(await store.getAnswers(1), isEmpty);
  });

  test(
    'clearScore on a book with nothing recorded is a no-op, not an error',
    () async {
      final store = LocalScoreStore();
      await store.clearScore(999);
      expect(await store.getAnswers(999), isEmpty);
    },
  );
}
