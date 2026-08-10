import 'breakdown.dart';
import 'question.dart';

/// One reading chunk of a book, with its three questions and breakdown
/// attached. See AIDOKU_DESIGN.md §3a/§4.
///
/// Note: the real data model (§4) keeps Chunk/Question/Breakdown as
/// separate entities joined by chunk_id; this mock nests them under Chunk
/// because that's the shape a "get chunk with everything" API response
/// would take, and it's the simplest shape for a bundled JSON asset with
/// no backend behind it yet.
class Chunk {
  final String id;
  final String bookId;
  final int index;
  final String text;
  final int charCount;
  final List<Question> questions;
  final Breakdown breakdown;

  const Chunk({
    required this.id,
    required this.bookId,
    required this.index,
    required this.text,
    required this.charCount,
    required this.questions,
    required this.breakdown,
  });

  factory Chunk.fromJson(Map<String, dynamic> json) {
    return Chunk(
      id: json['id'] as String,
      bookId: json['book_id'] as String,
      index: json['index'] as int,
      text: json['text'] as String,
      charCount: json['char_count'] as int,
      questions: (json['questions'] as List)
          .map((q) => Question.fromJson(q as Map<String, dynamic>))
          .toList(),
      breakdown: Breakdown.fromJson(json['breakdown'] as Map<String, dynamic>),
    );
  }
}
