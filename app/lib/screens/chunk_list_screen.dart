import 'package:flutter/material.dart';

import '../data/book_content_repository.dart';
import '../data/score_store.dart';
import '../models/book.dart';
import '../models/chunk_summary.dart';
import 'chunk_review_session_screen.dart';

/// Lets a reader browse every chunk they've cleared in [book] so far and
/// jump back into any of them for a full redo — see
/// ChunkReviewSessionScreen for what "redo" means. Opened from
/// ReadingSessionScreen's app bar once at least one chunk is cleared;
/// the caller decides what "cleared" means (see its `_clearedCount`)
/// and passes the resulting id list — this screen just displays it.
class ChunkListScreen extends StatefulWidget {
  final Book book;
  final BookContentRepository repository;
  final ScoreStore scoreStore;
  final List<int> chunkIds;

  const ChunkListScreen({
    super.key,
    required this.book,
    required this.repository,
    required this.scoreStore,
    required this.chunkIds,
  });

  @override
  State<ChunkListScreen> createState() => _ChunkListScreenState();
}

/// [passed] is null when some of the chunk's questions haven't been
/// answered — shouldn't happen for a chunk that's genuinely in
/// [ChunkListScreen.chunkIds] (ChunkPanel requires all 3 answered before
/// a chunk counts as cleared), but this is the honest fallback rather
/// than guessing true or false.
typedef _ChunkRow = ({ChunkSummary summary, bool? passed});

class _ChunkListScreenState extends State<ChunkListScreen> {
  // Mutable (not `late final`) and reassigned in _openSession once a
  // review session returns — a redo in that session can change
  // ScoreStore's answers, and this screen's own State instance is
  // never rebuilt from scratch by the pop (it was just covered, not
  // removed), so without this the badges shown here would go stale the
  // moment the reader comes back from redoing something.
  late Future<List<_ChunkRow>> _rowsFuture;

  @override
  void initState() {
    super.initState();
    _rowsFuture = _loadRows();
  }

  Future<List<_ChunkRow>> _loadRows() async {
    final allSummaries = await widget.repository.getChunkSummaries(
      widget.book.id,
    );
    final summaryById = {for (final s in allSummaries) s.id: s};

    // One lightweight (ids-only) request per reviewable chunk, in
    // parallel — cheap, and the only way to know which ScoreStore
    // answers belong to which chunk (ScoreStore itself is keyed by
    // question id, not chunk id).
    final questionIdsByChunk = await Future.wait(
      widget.chunkIds.map(widget.repository.getQuestionIds),
    );
    final answers = await widget.scoreStore.getAnswers(widget.book.id);

    return [
      for (var i = 0; i < widget.chunkIds.length; i++)
        (
          summary: summaryById[widget.chunkIds[i]]!,
          passed: chunkPassed(questionIdsByChunk[i], answers),
        ),
    ];
  }

  Future<void> _openSession(int startIndex) async {
    await Navigator.of(context).push(
      MaterialPageRoute(
        builder: (_) => ChunkReviewSessionScreen(
          book: widget.book,
          repository: widget.repository,
          scoreStore: widget.scoreStore,
          chunkIds: widget.chunkIds,
          startIndex: startIndex,
        ),
      ),
    );
    // Refresh badges regardless of how the session ended (manual back,
    // or its own auto-pop on finishing the last reviewable chunk) — see
    // _rowsFuture's doc comment.
    if (!mounted) return;
    // Not `setState(() => _rowsFuture = _loadRows())` — an assignment
    // expression evaluates to its RHS, so that arrow form would make
    // this callback itself return a Future, which setState rejects.
    setState(() {
      _rowsFuture = _loadRows();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text('Review: ${widget.book.title}')),
      body: FutureBuilder<List<_ChunkRow>>(
        future: _rowsFuture,
        builder: (context, snapshot) {
          if (snapshot.connectionState != ConnectionState.done) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snapshot.hasError) {
            return Center(
              child: Text('Failed to load chunks: ${snapshot.error}'),
            );
          }
          final rows = snapshot.data!;
          return ListView.separated(
            padding: const EdgeInsets.all(16),
            itemCount: rows.length,
            separatorBuilder: (_, _) => const SizedBox(height: 8),
            itemBuilder: (context, i) =>
                _ChunkListTile(row: rows[i], onTap: () => _openSession(i)),
          );
        },
      ),
    );
  }
}

/// One row: chunk number, its preview text, and a pass/fail indicator
/// that's both a soft background tint and an icon — color alone isn't
/// enough (colorblind-unfriendly, and easy to miss at a glance), so the
/// icon carries the same information redundantly.
class _ChunkListTile extends StatelessWidget {
  final _ChunkRow row;
  final VoidCallback onTap;

  const _ChunkListTile({required this.row, required this.onTap});

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final passed = row.passed;
    final background = switch (passed) {
      true => theme.colorScheme.primaryContainer.withValues(alpha: 0.80),
      false => theme.colorScheme.errorContainer.withValues(alpha: 0.80),
      null => null,
    };
    final icon = switch (passed) {
      true => Icon(Icons.check_circle, color: theme.colorScheme.primary),
      false => Icon(Icons.cancel, color: theme.colorScheme.error),
      null => null,
    };

    return Card(
      color: background,
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Chunk ${row.summary.index + 1}',
                      style: theme.textTheme.labelLarge,
                    ),
                    const SizedBox(height: 4),
                    Text(
                      row.summary.preview,
                      maxLines: 2,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ],
                ),
              ),
              if (icon != null) ...[const SizedBox(width: 12), icon],
            ],
          ),
        ),
      ),
    );
  }
}
