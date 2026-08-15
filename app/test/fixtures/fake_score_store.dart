import 'package:aidoku/data/score_store.dart';

/// An in-memory ScoreStore for widget tests — same rationale as
/// FakeProgressStore (its sibling). Optionally seeded with existing
/// answers, to test score display without driving the whole
/// question-answering flow.
class FakeScoreStore implements ScoreStore {
  final Map<int, Map<int, bool>> _answersByBook;

  FakeScoreStore({Map<int, Map<int, bool>>? seed})
    : _answersByBook =
          seed?.map((bookId, answers) => MapEntry(bookId, Map.of(answers))) ??
          {};

  @override
  Future<void> recordAnswer({
    required int bookId,
    required int questionId,
    required bool correct,
  }) async {
    _answersByBook.putIfAbsent(bookId, () => {})[questionId] = correct;
  }

  @override
  Future<Map<int, bool>> getAnswers(int bookId) async =>
      Map.of(_answersByBook[bookId] ?? const {});

  @override
  Future<void> clearScore(int bookId) async {
    _answersByBook.remove(bookId);
  }
}
