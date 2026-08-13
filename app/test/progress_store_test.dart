// LocalProgressStore in isolation, against real shared_preferences (its
// platform channel mocked via setMockInitialValues — the package's own
// documented way to test code that uses it). See resume_test.dart for
// ReadingSessionScreen actually using a ProgressStore.

import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:aidoku/data/local_progress_store.dart';

void main() {
  setUp(() {
    SharedPreferences.setMockInitialValues({});
  });

  test('getChunkIndex returns null when nothing has been saved', () async {
    final store = LocalProgressStore();
    expect(await store.getChunkIndex(1), isNull);
  });

  test('saveChunkIndex then getChunkIndex round-trips', () async {
    final store = LocalProgressStore();
    await store.saveChunkIndex(1, 4);
    expect(await store.getChunkIndex(1), 4);
  });

  test(
    'saveChunkIndex overwrites a previous value for the same book',
    () async {
      final store = LocalProgressStore();
      await store.saveChunkIndex(1, 4);
      await store.saveChunkIndex(1, 7);
      expect(await store.getChunkIndex(1), 7);
    },
  );

  test('progress for different books is independent', () async {
    final store = LocalProgressStore();
    await store.saveChunkIndex(1, 4);
    await store.saveChunkIndex(2, 9);
    expect(await store.getChunkIndex(1), 4);
    expect(await store.getChunkIndex(2), 9);
  });

  test('clearProgress removes the saved value', () async {
    final store = LocalProgressStore();
    await store.saveChunkIndex(1, 4);
    await store.clearProgress(1);
    expect(await store.getChunkIndex(1), isNull);
  });

  test(
    'clearProgress on a book with nothing saved is a no-op, not an error',
    () async {
      final store = LocalProgressStore();
      await store.clearProgress(999);
      expect(await store.getChunkIndex(999), isNull);
    },
  );
}
