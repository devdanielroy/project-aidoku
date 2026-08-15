/// Where a reader's per-book question history lives — same
/// consumer-defined-interface pattern as ProgressStore (see its own doc
/// comment for the rationale): LocalScoreStore is the only
/// implementation for now, but the shape mirrors AIDOKU_DESIGN.md §4's
/// UserProgress.answers_history so an account-backed implementation can
/// swap in later without QuestionsView/ReadingSessionScreen changing.
///
/// Deliberately a separate interface from ProgressStore rather than
/// folded into it — "which chunk am I on" and "how did I do on past
/// questions" are read/written at different times and by different
/// widgets (ReadingSessionScreen vs. QuestionsView), and keeping them
/// apart means a future backing change to one doesn't drag the other
/// along.
abstract class ScoreStore {
  /// Records whether [questionId] (in [bookId]) was answered correctly.
  /// Called once per question, the moment it's answered — see
  /// QuestionsView._selectOption. Answering the same question again
  /// (currently only possible via restart) overwrites the earlier
  /// result rather than keeping both.
  Future<void> recordAnswer({
    required int bookId,
    required int questionId,
    required bool correct,
  });

  /// Every recorded answer for [bookId] so far, keyed by question id.
  /// Empty if the reader hasn't answered anything in this book yet.
  Future<Map<int, bool>> getAnswers(int bookId);

  /// Wipes [bookId]'s recorded answers — called on restart, so a fresh
  /// attempt's accuracy isn't diluted by a previous one.
  Future<void> clearScore(int bookId);
}

/// Whether every question in [questionIds] was answered correctly in
/// [answers] (see [ScoreStore.getAnswers]) — true if all were correct,
/// false if any wasn't, null if some haven't been answered at all.
/// Shared by ChunkListScreen (the pass/fail badge per chunk) and
/// ChunkReviewSessionScreen (whether to show a chunk's questions
/// pre-answered rather than asking the reader to redo something they
/// already got right).
bool? chunkPassed(Iterable<int> questionIds, Map<int, bool> answers) {
  final results = questionIds.map((id) => answers[id]).toList();
  if (results.contains(null)) return null;
  return results.every((correct) => correct!);
}
