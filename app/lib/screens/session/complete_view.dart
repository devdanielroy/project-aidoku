import 'package:flutter/material.dart';

import '../../models/book.dart';

/// Shown once every chunk in the book (or, for now, mock excerpt) has been
/// read, questioned, and broken down.
class CompleteView extends StatelessWidget {
  final Book book;
  final VoidCallback onRestart;

  const CompleteView({super.key, required this.book, required this.onRestart});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.celebration, size: 64, color: theme.colorScheme.primary),
            const SizedBox(height: 16),
            Text(
              'You finished this excerpt of "${book.title}"!',
              textAlign: TextAlign.center,
              style: theme.textTheme.titleLarge,
            ),
            const SizedBox(height: 8),
            Text(
              'Only ${book.chunks.length} chunk(s) exist so far — this is a mock vertical slice, not the full book.',
              textAlign: TextAlign.center,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
            const SizedBox(height: 24),
            OutlinedButton(
              onPressed: onRestart,
              child: const Text('Read again'),
            ),
          ],
        ),
      ),
    );
  }
}
