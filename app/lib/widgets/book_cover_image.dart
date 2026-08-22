import 'dart:typed_data';

import 'package:flutter/material.dart';

import '../data/book_content_repository.dart';

/// A book's cover, fetched as raw bytes through [repository]
/// (BookContentRepository.getBookImage — see its own doc comment for
/// why not a bare URL handed to Image.network) and rendered via
/// Image.memory. Falls back to a generic book icon while loading, on
/// error, or when the book has no stored cover (not every book does —
/// see db/schema.sql's book_image, nullable). Shared between
/// ShopScreen's list cards and BookDetailScreen's larger cover, just at
/// different sizes — the fetch/fallback logic doesn't otherwise change.
class BookCoverImage extends StatefulWidget {
  final int bookId;
  final BookContentRepository repository;
  final double width;
  final double height;
  final double borderRadius;
  final double placeholderIconSize;

  const BookCoverImage({
    super.key,
    required this.bookId,
    required this.repository,
    required this.width,
    required this.height,
    this.borderRadius = 4,
    this.placeholderIconSize = 24,
  });

  @override
  State<BookCoverImage> createState() => _BookCoverImageState();
}

class _BookCoverImageState extends State<BookCoverImage> {
  // Fetched once per widget instance, not on every rebuild.
  late final Future<Uint8List?> _imageFuture = widget.repository.getBookImage(
    widget.bookId,
  );

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return ClipRRect(
      borderRadius: BorderRadius.circular(widget.borderRadius),
      child: SizedBox(
        width: widget.width,
        height: widget.height,
        child: FutureBuilder<Uint8List?>(
          future: _imageFuture,
          builder: (context, snapshot) {
            // Covers both "no image stored" (null) and "the fetch
            // failed" (an error) the same way: plenty of books have no
            // cover yet, so neither is worth surfacing as a real error
            // here - same fallback while still loading, to avoid a
            // layout flash.
            final bytes = snapshot.data;
            if (snapshot.connectionState != ConnectionState.done ||
                snapshot.hasError ||
                bytes == null) {
              return Container(
                color: theme.colorScheme.surfaceContainerHighest,
                child: Icon(
                  Icons.menu_book,
                  size: widget.placeholderIconSize,
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              );
            }
            return Image.memory(bytes, fit: BoxFit.cover);
          },
        ),
      ),
    );
  }
}
