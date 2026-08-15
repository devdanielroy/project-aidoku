/// One row of a book's chunk review list — a teaser, not the full
/// chunk. Mirrors book-content's chunk summary response exactly (GET
/// /aidoku/book/{book_id}/chunks/summary) — see book-content/internal/db's
/// ChunkSummary struct for why this exists as its own lightweight shape
/// instead of reusing Chunk (which carries full text, expensive to fetch
/// one-per-row for a whole book's worth of rows).
class ChunkSummary {
  final int id;
  final int index;
  final String preview;

  const ChunkSummary({
    required this.id,
    required this.index,
    required this.preview,
  });

  factory ChunkSummary.fromJson(Map<String, dynamic> json) {
    return ChunkSummary(
      id: json['id'] as int,
      index: json['index'] as int,
      preview: json['preview'] as String,
    );
  }
}
