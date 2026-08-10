import 'package:flutter/material.dart';

import '../../models/chunk.dart';
import '../../models/question.dart';
import 'breakdown_view.dart';
import 'questions_view.dart';
import 'reading_view.dart';

/// The three phases of the core loop for a single chunk (AIDOKU_DESIGN.md
/// §2): read unassisted, answer three questions, see the breakdown.
enum _ChunkPhase { reading, questions, breakdown }

/// Owns the whole per-chunk loop as one persistent screen instead of a
/// stack of separate ones: the passage is always visible — "it is the
/// core of the lesson" — through all three phases, just given
/// progressively less of the screen as a bottom sheet grows to hold
/// whatever's currently needed below it:
///
/// - Reading: passage fills the screen, undisturbed, nothing else visible.
/// - Questions: a sheet slides up to cover half the screen; the passage
///   shrinks into the remaining top half and stays scrollable/readable.
/// - Breakdown: the sheet grows further (the breakdown is usually the
///   longest content of the three) and the passage is squeezed into a
///   small strip at the top, in smaller text, but never hidden entirely.
class ChunkPanel extends StatefulWidget {
  final Chunk chunk;
  final int chunkNumber;
  final int totalChunks;
  final bool isLastChunk;
  final VoidCallback onChunkComplete;

  const ChunkPanel({
    super.key,
    required this.chunk,
    required this.chunkNumber,
    required this.totalChunks,
    required this.isLastChunk,
    required this.onChunkComplete,
  });

  @override
  State<ChunkPanel> createState() => _ChunkPanelState();
}

class _ChunkPanelState extends State<ChunkPanel> {
  /// How much of the available height the sheet takes up in each phase —
  /// "the bottom half" for questions, and enough to comfortably fit the
  /// (usually longer) breakdown while still leaving the passage visible.
  static const _questionsPanelFraction = 0.5;
  static const _breakdownPanelFraction = 0.75;
  static const _revealDuration = Duration(milliseconds: 400);

  _ChunkPhase _phase = _ChunkPhase.reading;
  Question? _activeQuestion;

  double get _panelFraction => switch (_phase) {
    _ChunkPhase.reading => 0.0,
    _ChunkPhase.questions => _questionsPanelFraction,
    _ChunkPhase.breakdown => _breakdownPanelFraction,
  };

  void _revealQuestions() => setState(() => _phase = _ChunkPhase.questions);

  void _revealBreakdown() => setState(() => _phase = _ChunkPhase.breakdown);

  void _onActiveQuestionChanged(Question question) =>
      setState(() => _activeQuestion = question);

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        final targetPanelHeight = constraints.maxHeight * _panelFraction;
        // The sheet's *content* is always laid out at the height its
        // current occupant (QuestionsView pre-reveal and during
        // questions, BreakdownView once there) actually needs — not at
        // targetPanelHeight, which is 0 during the reading phase. That
        // keeps QuestionsView mounted and fully laid out (so its
        // initState fires and _activeQuestion is ready the instant the
        // sheet opens) even while it's invisible, clipped away below.
        final sheetContentHeight =
            constraints.maxHeight *
            (_phase == _ChunkPhase.breakdown
                ? _breakdownPanelFraction
                : _questionsPanelFraction);

        return TweenAnimationBuilder<double>(
          tween: Tween<double>(end: targetPanelHeight),
          duration: _revealDuration,
          curve: Curves.easeInOutCubic,
          builder: (context, panelHeight, child) {
            return Stack(
              children: [
                // The passage. Its box shrinks from the bottom as
                // panelHeight grows, so the (centered) text visibly rises
                // — "pushed up" — in lockstep with the sheet below.
                Positioned(
                  top: 0,
                  left: 0,
                  right: 0,
                  bottom: panelHeight,
                  child: ReadingView(
                    chunk: widget.chunk,
                    chunkNumber: widget.chunkNumber,
                    totalChunks: widget.totalChunks,
                    onDone: _revealQuestions,
                    showContinueButton: _phase == _ChunkPhase.reading,
                    highlightText: _phase == _ChunkPhase.questions
                        ? _activeQuestion?.highlight
                        : null,
                    compact: _phase == _ChunkPhase.breakdown,
                  ),
                ),
                // The sheet. Laid out once at sheetContentHeight via
                // OverflowBox (so nothing inside it reflows mid-animation)
                // and revealed upward through a clipped, growing window —
                // a curtain rising, not a resizing box.
                Positioned(
                  left: 0,
                  right: 0,
                  bottom: 0,
                  height: panelHeight,
                  child: ClipRect(
                    child: OverflowBox(
                      alignment: Alignment.bottomCenter,
                      maxHeight: sheetContentHeight,
                      minHeight: 0,
                      child: _Sheet(
                        child: _phase == _ChunkPhase.breakdown
                            ? BreakdownView(
                                key: const ValueKey('breakdown'),
                                chunk: widget.chunk,
                                isLastChunk: widget.isLastChunk,
                                onNext: widget.onChunkComplete,
                              )
                            : QuestionsView(
                                key: const ValueKey('questions'),
                                chunk: widget.chunk,
                                onComplete: _revealBreakdown,
                                onQuestionChanged: _onActiveQuestionChanged,
                              ),
                      ),
                    ),
                  ),
                ),
              ],
            );
          },
        );
      },
    );
  }
}

/// The bottom-sheet surface the questions/breakdown live in: rounded top
/// corners and a static drag-handle-style bar for the familiar "sheet"
/// look (it's not actually draggable — the sheet only opens and grows via
/// the flow's own buttons — but the affordance still reads correctly at a
/// glance), plus elevation to separate it visually from the passage
/// above. Cross-fades its content (AnimatedSwitcher keys off the child's
/// runtimeType, so QuestionsView <-> BreakdownView switches automatically)
/// rather than snapping instantly when the breakdown phase begins.
class _Sheet extends StatelessWidget {
  final Widget child;

  const _Sheet({required this.child});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Material(
      color: theme.colorScheme.surfaceContainerHigh,
      elevation: 8,
      shadowColor: Colors.black.withValues(alpha: 0.3),
      borderRadius: const BorderRadius.vertical(top: Radius.circular(28)),
      clipBehavior: Clip.antiAlias,
      child: Column(
        children: [
          const SizedBox(height: 10),
          Container(
            width: 40,
            height: 4,
            decoration: BoxDecoration(
              color: theme.colorScheme.onSurfaceVariant.withValues(alpha: 0.4),
              borderRadius: BorderRadius.circular(2),
            ),
          ),
          Expanded(
            child: AnimatedSwitcher(
              duration: const Duration(milliseconds: 200),
              child: child,
            ),
          ),
        ],
      ),
    );
  }
}
