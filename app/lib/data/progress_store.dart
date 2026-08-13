/// Where a reader's per-book progress (currently: just which chunk
/// they're on) lives. Deliberately an interface, not a concrete class —
/// LocalProgressStore (device-local, via shared_preferences) is the only
/// implementation for now, but the shape is chosen so a future
/// account-backed implementation (talking to book-content's not-yet-built
/// UserProgress endpoints — see AIDOKU_DESIGN.md §4's UserProgress
/// sketch, which this deliberately mirrors) can be swapped in later
/// without LibraryScreen/ReadingSessionScreen changing at all. Same
/// consumer-defined-interface pattern as BookContentRepository's own
/// injectability.
abstract class ProgressStore {
  /// The chunk index the reader was last on in [bookId], or null if
  /// they've never started it (or already finished it — see
  /// clearProgress).
  Future<int?> getChunkIndex(int bookId);

  /// Records [chunkIndex] as [bookId]'s current position. Called
  /// whenever the reader moves to a new chunk, not just once a chunk is
  /// fully completed — so quitting mid-chunk (still reading, or partway
  /// through its questions) still resumes at that same chunk next time,
  /// not the one before it.
  Future<void> saveChunkIndex(int bookId, int chunkIndex);

  /// Clears [bookId]'s saved position — called once the book is
  /// finished (nothing left to resume) or on an explicit restart.
  Future<void> clearProgress(int bookId);
}
