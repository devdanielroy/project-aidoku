/// The three question types every chunk is tested with — see
/// AIDOKU_DESIGN.md §2 step 4.
enum QuestionType { vocab, grammar, comprehension }

QuestionType questionTypeFromString(String value) {
  switch (value) {
    case 'vocab':
      return QuestionType.vocab;
    case 'grammar':
      return QuestionType.grammar;
    case 'comprehension':
      return QuestionType.comprehension;
  }
  throw ArgumentError('Unknown question type: $value');
}

/// One multiple-choice question tied to a chunk. Mirrors book-content's
/// question response exactly (GET /aidoku/question/{question_id}) — see
/// book-content/internal/db's Question struct and AIDOKU_DESIGN.md §4.
///
/// [answerIndex] is always 0 in the stored data — the generation prompt
/// (pipeline/internal/question) deliberately always puts the correct
/// answer first and leaves shuffling for display to the client. See
/// QuestionsView, which shuffles before rendering rather than trusting
/// [options] to already be in a random order.
class Question {
  final int id;
  final int chunkId;
  final QuestionType type;
  final String prompt;
  final List<String> options;
  final int answerIndex;
  final String explanation;
  final String? highlight;

  const Question({
    required this.id,
    required this.chunkId,
    required this.type,
    required this.prompt,
    required this.options,
    required this.answerIndex,
    required this.explanation,
    this.highlight,
  });

  factory Question.fromJson(Map<String, dynamic> json) {
    return Question(
      id: json['id'] as int,
      chunkId: json['chunk_id'] as int,
      type: questionTypeFromString(json['type'] as String),
      prompt: json['prompt'] as String,
      options: (json['options'] as List).cast<String>(),
      answerIndex: json['answer_index'] as int,
      explanation: json['explanation'] as String,
      highlight: json['highlight'] as String?,
    );
  }
}
