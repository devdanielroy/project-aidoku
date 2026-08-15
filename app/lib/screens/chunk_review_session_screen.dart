import 'package:flutter/material.dart';

import '../data/book_content_repository.dart';
import '../data/score_store.dart';
import '../models/book.dart';
import '../models/loaded_chunk.dart';
import '../models/question.dart';
import 'session/chunk_panel.dart';

/// Lets a reader redo a chunk they've already cleared — the full read →
/// questions → breakdown loop, exactly like the main reading flow (via
/// the same ChunkPanel), just scoped to [chunkIds] rather than the whole
/// book. Opened from ChunkListScreen at whichever chunk was tapped.
///
/// Three things deliberately differ from ReadingSessionScreen:
/// - A chunk that was already fully passed opens with every question
///   pre-answered on its correct option (see QuestionsView.startAnswered)
///   rather than asking the reader to redo something they already got
///   right — only a chunk with at least one wrong (or unanswered)
///   question is a real do-over. Pre-filled answers aren't re-recorded
///   to ScoreStore (nothing was actually re-answered); genuine answers,
///   on a real redo, overwrite the previous result the same way
///   ScoreStore already does elsewhere.
/// - Nothing here ever touches ProgressStore — finishing a review chunk
///   must not move the reader's real resume bookmark backward.
/// - "Next chunk" advances to the next id in [chunkIds] (the reviewable
///   subset), not the next chunk in the whole book, and pops back to
///   ChunkListScreen once that subset is exhausted rather than trying to
///   continue into chunks that aren't cleared yet.
///
/// Chunk numbering shown here (app bar title, ChunkPanel's "CHUNK N OF
/// M") is the reader's position within *this review session*, not the
/// chunk's real position in the book — e.g. "Chunk 1 of 3" for the
/// first of 3 reviewable chunks, even if that's actually chunk 5 of a
/// 20-chunk book.
class ChunkReviewSessionScreen extends StatefulWidget {
  final Book book;
  final BookContentRepository repository;
  final ScoreStore scoreStore;
  final List<int> chunkIds;

  /// Which position in [chunkIds] to open on — the chunk that was
  /// actually tapped in ChunkListScreen, not necessarily the first.
  final int startIndex;

  const ChunkReviewSessionScreen({
    super.key,
    required this.book,
    required this.repository,
    required this.scoreStore,
    required this.chunkIds,
    required this.startIndex,
  });

  @override
  State<ChunkReviewSessionScreen> createState() =>
      _ChunkReviewSessionScreenState();
}

/// A loaded chunk plus whether it was already fully passed *before* this
/// screen opened — computed once per chunk load (see _load) rather than
/// derived at build time, so a genuine redo answered wrong mid-session
/// can't retroactively flip a chunk that started out passed.
typedef _ReviewChunk = ({LoadedChunk loaded, bool startAnswered});

class _ChunkReviewSessionScreenState extends State<ChunkReviewSessionScreen> {
  late int _index = widget.startIndex;
  late Future<_ReviewChunk> _chunkFuture = _load(widget.chunkIds[_index]);

  Future<_ReviewChunk> _load(int chunkId) async {
    // Concurrent, not sequential — loadChunk doesn't depend on
    // getAnswers or vice versa.
    final loadedFuture = widget.repository.loadChunk(chunkId);
    final answersFuture = widget.scoreStore.getAnswers(widget.book.id);
    final loaded = await loadedFuture;
    final answers = await answersFuture;
    final passed = chunkPassed(loaded.questions.map((q) => q.id), answers);
    return (loaded: loaded, startAnswered: passed == true);
  }

  void _onAnswered(Question question, bool correct) {
    widget.scoreStore.recordAnswer(
      bookId: widget.book.id,
      questionId: question.id,
      correct: correct,
    );
  }

  void _onChunkComplete() {
    final nextIndex = _index + 1;
    if (nextIndex >= widget.chunkIds.length) {
      // Nothing further cleared to review yet — back to the list, same
      // as pressing the app bar's back button mid-chunk.
      Navigator.of(context).pop();
      return;
    }
    setState(() {
      _index = nextIndex;
      _chunkFuture = _load(widget.chunkIds[_index]);
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text('Chunk ${_index + 1} of ${widget.chunkIds.length}'),
      ),
      body: FutureBuilder<_ReviewChunk>(
        future: _chunkFuture,
        builder: (context, snapshot) {
          if (snapshot.connectionState != ConnectionState.done) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snapshot.hasError) {
            return Center(
              child: Text('Failed to load chunk: ${snapshot.error}'),
            );
          }
          final review = snapshot.data!;
          return ChunkPanel(
            key: ValueKey('review-chunk-${review.loaded.chunk.id}'),
            loadedChunk: review.loaded,
            chunkNumber: _index + 1,
            totalChunks: widget.chunkIds.length,
            isLastChunk: _index + 1 >= widget.chunkIds.length,
            onChunkComplete: _onChunkComplete,
            onAnswered: _onAnswered,
            startAnswered: review.startAnswered,
          );
        },
      ),
    );
  }
}
