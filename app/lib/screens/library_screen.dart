import 'package:flutter/material.dart';

import '../data/mock_book_repository.dart';
import '../models/book.dart';
import 'reading_session_screen.dart';

/// Step 1 of the core loop (AIDOKU_DESIGN.md §2): pick a book, filtered by
/// level. Only one mock book exists so far, so this is a minimal
/// stand-in for a real library/catalog screen — enough to prove the
/// "pick a book, then read it" flow exists.
class LibraryScreen extends StatefulWidget {
  const LibraryScreen({super.key});

  @override
  State<LibraryScreen> createState() => _LibraryScreenState();
}

class _LibraryScreenState extends State<LibraryScreen> {
  final _repository = const MockBookRepository();
  late final Future<List<Book>> _booksFuture = _repository.loadBooks();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Aidoku 愛読')),
      body: FutureBuilder<List<Book>>(
        future: _booksFuture,
        builder: (context, snapshot) {
          if (snapshot.connectionState != ConnectionState.done) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snapshot.hasError) {
            return Center(
              child: Text('Failed to load books: ${snapshot.error}'),
            );
          }
          final books = snapshot.data!;
          return ListView.separated(
            padding: const EdgeInsets.all(16),
            itemCount: books.length,
            separatorBuilder: (_, _) => const SizedBox(height: 12),
            itemBuilder: (context, i) => _BookCard(book: books[i]),
          );
        },
      ),
    );
  }
}

class _BookCard extends StatelessWidget {
  final Book book;

  const _BookCard({required this.book});

  @override
  Widget build(BuildContext context) {
    return Card(
      child: ListTile(
        contentPadding: const EdgeInsets.all(16),
        title: Text(book.title, style: Theme.of(context).textTheme.titleMedium),
        subtitle: Padding(
          padding: const EdgeInsets.only(top: 4),
          child: Text(
            '${book.author} · ${book.levelTag} · ${book.chunks.length} chunks',
          ),
        ),
        trailing: const Icon(Icons.chevron_right),
        onTap: () => Navigator.of(context).push(
          MaterialPageRoute(builder: (_) => ReadingSessionScreen(book: book)),
        ),
      ),
    );
  }
}
