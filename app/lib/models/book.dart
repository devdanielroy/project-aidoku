import 'chunk.dart';

/// A book available to read, with its chunks already attached (a
/// mock-only convenience — see Chunk's doc comment). See
/// AIDOKU_DESIGN.md §4.
class Book {
  final String id;
  final String title;
  final String author;
  final String sourceUrl;
  final String levelTag;
  final String language;
  final String status;
  final List<Chunk> chunks;

  const Book({
    required this.id,
    required this.title,
    required this.author,
    required this.sourceUrl,
    required this.levelTag,
    required this.language,
    required this.status,
    required this.chunks,
  });

  factory Book.fromJson(Map<String, dynamic> json) {
    final bookJson = json['book'] as Map<String, dynamic>;
    return Book(
      id: bookJson['id'] as String,
      title: bookJson['title'] as String,
      author: bookJson['author'] as String,
      sourceUrl: bookJson['source_url'] as String,
      levelTag: bookJson['level_tag'] as String,
      language: bookJson['language'] as String,
      status: bookJson['status'] as String,
      chunks: (json['chunks'] as List)
          .map((c) => Chunk.fromJson(c as Map<String, dynamic>))
          .toList(),
    );
  }
}
