/// The full post-answer explanation of a chunk (vocab, grammar, meaning),
/// written in the learner's L1. Mirrors book-content's breakdown response
/// exactly (GET /aidoku/chunk/{chunk_id}/breakdown) — see
/// book-content/internal/db's Breakdown struct and AIDOKU_DESIGN.md §2
/// step 5 / §4.
class Breakdown {
  final int id;
  final int chunkId;
  final String content;

  const Breakdown({
    required this.id,
    required this.chunkId,
    required this.content,
  });

  factory Breakdown.fromJson(Map<String, dynamic> json) {
    return Breakdown(
      id: json['id'] as int,
      chunkId: json['chunk_id'] as int,
      content: json['content'] as String,
    );
  }
}
