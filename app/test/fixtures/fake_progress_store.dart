import 'package:aidoku/data/progress_store.dart';

/// An in-memory ProgressStore for widget tests — avoids depending on
/// shared_preferences' platform channel mocking for tests that aren't
/// actually about persistence itself (see progress_store_test.dart for
/// those). Optionally seeded with existing progress, to test resuming.
class FakeProgressStore implements ProgressStore {
  final Map<int, int> _chunkIndexByBook;

  FakeProgressStore({Map<int, int>? seed})
    : _chunkIndexByBook = Map.of(seed ?? {});

  @override
  Future<int?> getChunkIndex(int bookId) async => _chunkIndexByBook[bookId];

  @override
  Future<void> saveChunkIndex(int bookId, int chunkIndex) async {
    _chunkIndexByBook[bookId] = chunkIndex;
  }

  @override
  Future<void> clearProgress(int bookId) async {
    _chunkIndexByBook.remove(bookId);
  }
}
