import 'package:flutter/material.dart';

import '../../models/chunk.dart';

/// Step 1 of the core loop (AIDOKU_DESIGN.md §2): show the chunk text and
/// nothing else — no dictionary, no hints, no translation. The reader has
/// to get through this unassisted before any help is offered; the
/// friction is the pedagogy, not a UX flaw to smooth away.
class ReadingView extends StatelessWidget {
  final Chunk chunk;
  final int chunkNumber;
  final int totalChunks;
  final VoidCallback onDone;

  /// False once the questions panel has been revealed — the continue
  /// button no longer makes sense once the reader is already answering
  /// questions below. See ChunkPanel.
  final bool showContinueButton;

  /// The current question's `Question.highlight` span, underlined in the
  /// text in place of that question re-quoting it. Null outside the
  /// questions phase, and for comprehension questions (which aren't about
  /// one specific word or phrase).
  final String? highlightText;

  /// True during the breakdown phase, when the passage is squeezed into a
  /// small strip at the top rather than filling most of the screen — see
  /// ChunkPanel. Shrinks the text and tightens spacing accordingly; the
  /// passage stays fully visible (just smaller), never hidden.
  final bool compact;

  const ReadingView({
    super.key,
    required this.chunk,
    required this.chunkNumber,
    required this.totalChunks,
    required this.onDone,
    this.showContinueButton = true,
    this.highlightText,
    this.compact = false,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final textStyle = compact
        ? theme.textTheme.bodyLarge?.copyWith(height: 1.4)
        : theme.textTheme.headlineSmall?.copyWith(height: 1.5);
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              'CHUNK $chunkNumber OF $totalChunks',
              style: theme.textTheme.labelLarge?.copyWith(
                color: theme.colorScheme.secondary,
              ),
            ),
            SizedBox(height: compact ? 8 : 24),
            Expanded(
              child: Center(
                child: SingleChildScrollView(
                  child: Text.rich(
                    TextSpan(
                      children: _highlightedSpans(
                        text: chunk.text,
                        highlight: highlightText,
                        baseStyle: textStyle,
                        underlineColor: theme.colorScheme.primary,
                      ),
                    ),
                  ),
                ),
              ),
            ),
            if (showContinueButton) ...[
              const SizedBox(height: 24),
              FilledButton(
                onPressed: onDone,
                child: const Text("I've read it — continue"),
              ),
            ],
          ],
        ),
      ),
    );
  }
}

/// Splits [text] into spans, underlining the first occurrence of
/// [highlight] (if any). Falls back to plain, unstyled spans when there's
/// nothing to highlight or [highlight] isn't actually found verbatim in
/// [text] — a mismatch there is a content bug (see Question.highlight),
/// not something to crash or silently blank the passage over.
List<InlineSpan> _highlightedSpans({
  required String text,
  required String? highlight,
  required TextStyle? baseStyle,
  required Color underlineColor,
}) {
  if (highlight == null || highlight.isEmpty) {
    return [TextSpan(text: text, style: baseStyle)];
  }
  final start = text.indexOf(highlight);
  if (start < 0) {
    return [TextSpan(text: text, style: baseStyle)];
  }
  final end = start + highlight.length;
  return [
    TextSpan(text: text.substring(0, start), style: baseStyle),
    TextSpan(
      text: text.substring(start, end),
      style: baseStyle?.copyWith(
        decoration: TextDecoration.underline,
        decorationColor: underlineColor,
        decorationThickness: 2,
        fontWeight: FontWeight.w700,
      ),
    ),
    TextSpan(text: text.substring(end), style: baseStyle),
  ];
}
