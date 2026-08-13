import 'breakdown.dart';
import 'chunk.dart';
import 'question.dart';

/// Everything the reading screens need for one chunk, bundled together —
/// a client-side convenience only. book-content keeps Chunk/Question/
/// Breakdown as separate resources fetched separately (see their own
/// doc comments for why); BookContentRepository.loadChunk assembles
/// this from three-plus calls once, so ChunkPanel and its children can
/// go back to taking plain, already-loaded data the way they did before
/// there was a real backend, instead of each doing their own fetching.
class LoadedChunk {
  final Chunk chunk;

  /// Always exactly 3 (one vocab, one grammar, one comprehension) for
  /// any chunk that's actually been through the pipeline's question
  /// generation stage — see db/schema.sql's UNIQUE(chunk_id, type).
  final List<Question> questions;
  final Breakdown breakdown;

  const LoadedChunk({
    required this.chunk,
    required this.questions,
    required this.breakdown,
  });
}
