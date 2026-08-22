import 'package:flutter/material.dart';

import '../data/book_content_repository.dart';
import '../models/book.dart';
import '../models/language_pair.dart';
import '../theme/accent_colors.dart';
import '../widgets/book_cover_image.dart';

/// A single book's page in the Store — opened by tapping a book in
/// ShopScreen's list, same "shopping site product page" shape: a big
/// cover, title/author, and a price + buy affordance up top, book
/// metadata and a summary below. Genre tags and the summary are real,
/// hand-curated per-book data (Book.genres/summary, sourced from the
/// catalog — see their own doc comments); everything to do with actually
/// buying is still a placeholder (see README's Shop milestone): no real
/// price, no purchase flow yet, called out inline below.
class BookDetailScreen extends StatefulWidget {
  final Book book;

  /// Overridable for tests — defaults to the real thing, same pattern
  /// as every other screen that talks to book-content.
  final BookContentRepository? repository;

  const BookDetailScreen({super.key, required this.book, this.repository});

  @override
  State<BookDetailScreen> createState() => _BookDetailScreenState();
}

class _BookDetailScreenState extends State<BookDetailScreen> {
  late final BookContentRepository _repository =
      widget.repository ?? BookContentRepository();

  // Only the chunk count is needed here (not the chunks themselves),
  // so this reuses the same id-list endpoint LibraryScreen's progress
  // bar does rather than fetching every chunk's full text.
  late final Future<int> _chunkCountFuture = _repository
      .getChunkIds(widget.book.id)
      .then((ids) => ids.length);

  @override
  Widget build(BuildContext context) {
    final book = widget.book;
    return Scaffold(
      appBar: AppBar(),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                BookCoverImage(
                  bookId: book.id,
                  repository: _repository,
                  width: 140,
                  height: 200,
                  borderRadius: 8,
                  placeholderIconSize: 48,
                ),
                const SizedBox(width: 16),
                Expanded(child: _TitleAuthor(book: book)),
                const SizedBox(width: 16),
                const _PurchaseColumn(),
              ],
            ),
            const SizedBox(height: 24),
            _MetadataSection(book: book, chunkCountFuture: _chunkCountFuture),
            const SizedBox(height: 24),
            _SummarySection(book: book),
          ],
        ),
      ),
    );
  }
}

class _TitleAuthor extends StatelessWidget {
  final Book book;

  const _TitleAuthor({required this.book});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        Text(book.title, style: theme.textTheme.headlineSmall),
        const SizedBox(height: 8),
        Text(
          'By ${book.author}',
          style: theme.textTheme.titleMedium?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }
}

/// Price + buy affordances, top right — everything here is a
/// placeholder (see BookDetailScreen's own doc comment): a fixed
/// $0.00 (currency/pricing isn't designed yet) and three buttons that
/// all do nothing when tapped, Free Sample included — this is still
/// just the shop's facade, real payment (and the free-sample reading
/// flow it'll actually open) comes later.
class _PurchaseColumn extends StatelessWidget {
  const _PurchaseColumn();

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return SizedBox(
      width: 140,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            '\$0.00',
            textAlign: TextAlign.end,
            style: theme.textTheme.titleLarge,
          ),
          const SizedBox(height: 12),
          FilledButton(onPressed: () {}, child: const Text('Buy Now')),
          const SizedBox(height: 8),
          OutlinedButton(onPressed: () {}, child: const Text('Add to Cart')),
          const SizedBox(height: 8),
          FilledButton(
            onPressed: () {},
            style: FilledButton.styleFrom(
              backgroundColor: babyBlueAccent,
              foregroundColor: Colors.black87,
            ),
            child: const Text('Free Sample'),
          ),
        ],
      ),
    );
  }
}

/// Language, reading level, chunk count, and genre tags — all real,
/// already-known book data (see Book.genres's own doc comment for where
/// the tags come from).
class _MetadataSection extends StatelessWidget {
  final Book book;
  final Future<int> chunkCountFuture;

  const _MetadataSection({required this.book, required this.chunkCountFuture});

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: [
        Chip(
          avatar: const Icon(Icons.language, size: 18),
          label: Text(languageDisplayName(book.targetLanguage)),
        ),
        Chip(
          avatar: const Icon(Icons.school, size: 18),
          label: Text(book.readingLevelName),
        ),
        FutureBuilder<int>(
          future: chunkCountFuture,
          builder: (context, snapshot) {
            final count = snapshot.data;
            if (count == null) return const SizedBox.shrink();
            return Chip(
              avatar: const Icon(Icons.menu_book, size: 18),
              label: Text('$count chunks'),
            );
          },
        ),
        for (final genre in book.genres) Chip(label: Text(genre)),
      ],
    );
  }
}

/// The book's real, hand-curated synopsis (see Book.summary's own doc
/// comment).
class _SummarySection extends StatelessWidget {
  final Book book;

  const _SummarySection({required this.book});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Summary', style: theme.textTheme.titleMedium),
        const SizedBox(height: 8),
        Text(
          book.summary,
          style: theme.textTheme.bodyMedium?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }
}
