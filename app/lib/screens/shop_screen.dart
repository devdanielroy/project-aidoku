import 'dart:typed_data';

import 'package:flutter/material.dart';

import '../data/book_content_repository.dart';
import '../models/book.dart';
import '../models/language_pair.dart';

/// Browse every published book, regardless of language — unfiltered
/// (unlike LibraryScreen's active-study-language filter) for this first
/// pass, since there's nothing to actually buy yet (see README's Shop
/// milestone: payment is a separate, not-yet-built piece). Just a list,
/// each row showing the book's cover (if it has one — not every book
/// does; see BookContentRepository.getBookImage), title, author, and
/// level.
/// HomeScreen's other bottom-nav tab ("Store"), alongside LibraryScreen
/// ("My Library").
class ShopScreen extends StatefulWidget {
  /// Overridable for tests — defaults to the real thing, same pattern
  /// as every other screen that talks to book-content.
  final BookContentRepository? repository;

  const ShopScreen({super.key, this.repository});

  @override
  State<ShopScreen> createState() => _ShopScreenState();
}

class _ShopScreenState extends State<ShopScreen> {
  late final BookContentRepository _repository =
      widget.repository ?? BookContentRepository();
  late final Future<List<Book>> _booksFuture = _repository.getBooks();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Store')),
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
            itemBuilder: (context, i) =>
                _ShopBookCard(book: books[i], repository: _repository),
          );
        },
      ),
    );
  }
}

class _ShopBookCard extends StatefulWidget {
  final Book book;
  final BookContentRepository repository;

  const _ShopBookCard({required this.book, required this.repository});

  @override
  State<_ShopBookCard> createState() => _ShopBookCardState();
}

class _ShopBookCardState extends State<_ShopBookCard> {
  // Fetched once per card via the repository (raw bytes, not a URL —
  // see getBookImage's own doc comment for why), rather than every
  // rebuild.
  late final Future<Uint8List?> _imageFuture = widget.repository.getBookImage(
    widget.book.id,
  );

  // Soft baby blue for the corner ribbon — a fixed accent independent
  // of the light/dark palette (main.dart's _DarkPalette/_LightPalette),
  // same as the semantic red Material3 already reserves for wrong
  // answers elsewhere in the app.
  static const _languageRibbonColor = Color(0xFF89CFF0);

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final book = widget.book;
    return Card(
      // Store's unfiltered by language pair (unlike LibraryScreen), so
      // the ribbon is what tells a shopper which language a given book
      // is actually written in before they buy it.
      clipBehavior: Clip.antiAlias,
      child: Banner(
        message: languageDisplayName(book.targetLanguage).toUpperCase(),
        location: BannerLocation.topEnd,
        color: _languageRibbonColor,
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              ClipRRect(
                borderRadius: BorderRadius.circular(4),
                child: SizedBox(
                  width: 56,
                  height: 80,
                  child: FutureBuilder<Uint8List?>(
                    future: _imageFuture,
                    builder: (context, snapshot) {
                      final bytes = snapshot.data;
                      if (snapshot.connectionState != ConnectionState.done ||
                          snapshot.hasError ||
                          bytes == null) {
                        return Container(
                          color: theme.colorScheme.surfaceContainerHighest,
                          child: Icon(
                            Icons.menu_book,
                            color: theme.colorScheme.onSurfaceVariant,
                          ),
                        );
                      }
                      return Image.memory(bytes, fit: BoxFit.cover);
                    },
                  ),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(book.title, style: theme.textTheme.titleMedium),
                    const SizedBox(height: 4),
                    Text(
                      'By ${book.author} · Level: ${book.readingLevelName}',
                      style: theme.textTheme.bodyMedium?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
