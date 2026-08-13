import 'package:flutter/material.dart';

import '../data/book_content_repository.dart';
import '../data/local_progress_store.dart';
import '../data/progress_store.dart';
import '../models/book.dart';
import 'reading_session_screen.dart';

/// Step 1 of the core loop (AIDOKU_DESIGN.md §2): pick a book, filtered by
/// level. Only one real book exists so far, so this is a minimal
/// stand-in for a real library/catalog screen — enough to prove the
/// "pick a book, then read it" flow exists against the real
/// book-content service.
class LibraryScreen extends StatefulWidget {
  /// Overridable for tests (a BookContentRepository wired to a fake HTTP
  /// transport — see test/fixtures/) — defaults to the real thing,
  /// talking to whatever AppConfig.bookContentBaseUrl points at.
  final BookContentRepository? repository;

  /// Overridable for tests (an in-memory fake — see test/fixtures/) —
  /// defaults to this device's local storage. See ProgressStore's own
  /// doc comment for why this is an interface at all.
  final ProgressStore? progressStore;

  const LibraryScreen({super.key, this.repository, this.progressStore});

  @override
  State<LibraryScreen> createState() => _LibraryScreenState();
}

class _LibraryScreenState extends State<LibraryScreen> {
  late final BookContentRepository _repository =
      widget.repository ?? BookContentRepository();
  late final ProgressStore _progressStore =
      widget.progressStore ?? LocalProgressStore();
  late final Future<List<Book>> _booksFuture = _repository.getBooks();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text("Aidoku")),
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
          if (books.isEmpty) {
            return const Center(child: Text('No books published yet.'));
          }
          return ListView.separated(
            padding: const EdgeInsets.all(16),
            itemCount: books.length,
            separatorBuilder: (_, _) => const SizedBox(height: 12),
            itemBuilder: (context, i) => _BookCard(
              book: books[i],
              repository: _repository,
              progressStore: _progressStore,
            ),
          );
        },
      ),
    );
  }
}

/// [index] is the last-saved chunk index (0-based), [total] is the
/// book's chunk count — together enough to render both the fraction bar
/// and the "chunk N of M" label. A record rather than a class: pure data,
/// no behavior, doesn't escape this file.
typedef _BookProgress = ({int index, int total});

class _BookCard extends StatefulWidget {
  final Book book;
  final BookContentRepository repository;
  final ProgressStore progressStore;

  const _BookCard({
    required this.book,
    required this.repository,
    required this.progressStore,
  });

  @override
  State<_BookCard> createState() => _BookCardState();
}

class _BookCardState extends State<_BookCard> {
  // Cached rather than looked up inline in build() — a FutureBuilder
  // whose `future:` is a fresh call on every build re-triggers on every
  // rebuild (e.g. a scroll), not just once. Matches how the books list
  // itself is cached in _LibraryScreenState.
  late final Future<_BookProgress?> _progressFuture = _loadProgress();

  // The chunk-id-count fetch (needed for the "of M" / fraction) only
  // happens for books that actually have saved progress — a book never
  // started costs nothing beyond the local progress lookup itself.
  Future<_BookProgress?> _loadProgress() async {
    final savedIndex = await widget.progressStore.getChunkIndex(widget.book.id);
    if (savedIndex == null) return null;
    final chunkIds = await widget.repository.getChunkIds(widget.book.id);
    if (chunkIds.isEmpty) {
      return null; // shouldn't happen, but don't divide by 0
    }
    return (index: savedIndex, total: chunkIds.length);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: () => Navigator.of(context).push(
          MaterialPageRoute(
            builder: (_) => ReadingSessionScreen(
              book: widget.book,
              repository: widget.repository,
              progressStore: widget.progressStore,
            ),
          ),
        ),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          widget.book.title,
                          style: theme.textTheme.titleMedium,
                        ),
                        const SizedBox(height: 4),
                        Text(
                          'By ${widget.book.author} · Level: ${widget.book.readingLevelName}',
                          style: theme.textTheme.bodyMedium?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                        ),
                      ],
                    ),
                  ),
                  const SizedBox(width: 8),
                  const Icon(Icons.chevron_right),
                ],
              ),
              FutureBuilder<_BookProgress?>(
                future: _progressFuture,
                builder: (context, snapshot) {
                  final progress = snapshot.data;
                  if (progress == null) return const SizedBox.shrink();
                  final fraction = progress.index / progress.total;
                  return Padding(
                    padding: const EdgeInsets.only(top: 12),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        ClipRRect(
                          borderRadius: BorderRadius.circular(4),
                          child: LinearProgressIndicator(
                            value: fraction,
                            minHeight: 6,
                          ),
                        ),
                        const SizedBox(height: 4),
                        Text(
                          'Chunk ${progress.index + 1} of ${progress.total}',
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                        ),
                      ],
                    ),
                  );
                },
              ),
            ],
          ),
        ),
      ),
    );
  }
}
