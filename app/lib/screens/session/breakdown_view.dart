import 'package:flutter/material.dart';

import '../../models/chunk.dart';

/// Step 3 of the core loop (AIDOKU_DESIGN.md §2 step 5): the full
/// breakdown — vocab, grammar, meaning, written in the learner's L1 — is
/// shown only after all three questions are answered, never before. Lives
/// inside the same bottom sheet as QuestionsView (see ChunkPanel), with
/// the passage still visible — now squeezed smaller — above it.
class BreakdownView extends StatelessWidget {
  final Chunk chunk;
  final bool isLastChunk;
  final VoidCallback onNext;

  const BreakdownView({
    super.key,
    required this.chunk,
    required this.isLastChunk,
    required this.onNext,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    // top: false — embedded in the sheet, not flush against the physical
    // top of the screen. See QuestionsView for the same reasoning.
    return SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              'BREAKDOWN',
              style: theme.textTheme.labelLarge?.copyWith(
                color: theme.colorScheme.secondary,
              ),
            ),
            const SizedBox(height: 16),
            Expanded(
              child: SingleChildScrollView(
                child: Text(
                  chunk.breakdown.content,
                  style: theme.textTheme.bodyLarge?.copyWith(height: 1.7),
                ),
              ),
            ),
            const SizedBox(height: 16),
            FilledButton(
              onPressed: onNext,
              child: Text(isLastChunk ? 'Finish this excerpt' : 'Next chunk'),
            ),
          ],
        ),
      ),
    );
  }
}
