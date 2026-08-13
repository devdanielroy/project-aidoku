import 'package:shared_preferences/shared_preferences.dart';

import 'progress_store.dart';

/// ProgressStore backed by shared_preferences — this device's local
/// storage, not synced anywhere. The obvious stand-in until there's a
/// real user account to hang progress off; see ProgressStore's own doc
/// comment for the intended migration path.
class LocalProgressStore implements ProgressStore {
  static const _keyPrefix = 'progress.chunk_index.book_';

  @override
  Future<int?> getChunkIndex(int bookId) async {
    final prefs = await SharedPreferences.getInstance();
    return prefs.getInt('$_keyPrefix$bookId');
  }

  @override
  Future<void> saveChunkIndex(int bookId, int chunkIndex) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setInt('$_keyPrefix$bookId', chunkIndex);
  }

  @override
  Future<void> clearProgress(int bookId) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove('$_keyPrefix$bookId');
  }
}
