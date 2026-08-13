/// One reading chunk of a book. Mirrors book-content's chunk response
/// exactly (GET /aidoku/chunk/{chunk_id}) — see book-content/internal/db's
/// Chunk struct. Doesn't carry its questions or breakdown: the real data
/// model (AIDOKU_DESIGN.md §4) keeps those as separate entities joined
/// by chunk_id, and book-content's API mirrors that (separate endpoints,
/// fetched separately) rather than nesting everything under one
/// response — see LoadedChunk for the client-side bundle that reunites
/// them for the reading screens, once all three are actually needed
/// together.
class Chunk {
  final int id;
  final int bookId;
  final int index;
  final String text;
  final int charCount;

  const Chunk({
    required this.id,
    required this.bookId,
    required this.index,
    required this.text,
    required this.charCount,
  });

  factory Chunk.fromJson(Map<String, dynamic> json) {
    return Chunk(
      id: json['id'] as int,
      bookId: json['book_id'] as int,
      index: json['index'] as int,
      text: json['text'] as String,
      charCount: json['char_count'] as int,
    );
  }
}
