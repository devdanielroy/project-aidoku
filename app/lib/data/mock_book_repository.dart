import 'dart:convert';

import 'package:flutter/services.dart' show rootBundle;

import '../models/book.dart';

/// Loads books from bundled mock JSON assets. Stands in for a real backend
/// call for this vertical slice — see AIDOKU_DESIGN.md §8: validate the
/// loop and data model with mock content before spending on the real
/// content pipeline / Claude API calls.
class MockBookRepository {
  const MockBookRepository();

  static const _assetPaths = ['assets/mock/pride_and_prejudice.json'];

  Future<List<Book>> loadBooks() async {
    final books = <Book>[];
    for (final path in _assetPaths) {
      final raw = await rootBundle.loadString(path);
      final json = jsonDecode(raw) as Map<String, dynamic>;
      books.add(Book.fromJson(json));
    }
    return books;
  }
}
