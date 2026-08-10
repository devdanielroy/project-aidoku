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

/// One multiple-choice question tied to a chunk. Mirrors the Question
/// entity in AIDOKU_DESIGN.md §4 — options/answer here take a fixed
/// multiple-choice shape (options + answerIndex), the simplest concrete
/// form of that field for this mock; the real pipeline's exact question
/// shape is still open.
class Question {
  final String id;
  final QuestionType type;
  final String prompt;
  final List<String> options;
  final int answerIndex;
  final String explanation;

  /// The exact substring of the chunk's text this question is about —
  /// underlined in the passage while this question is on screen instead
  /// of being re-quoted in the prompt. Null for comprehension questions,
  /// which are about the whole chunk rather than one word or phrase. Must
  /// appear verbatim in the chunk text; see ReadingView, which falls back
  /// to no underline if it doesn't.
  final String? highlight;

  const Question({
    required this.id,
    required this.type,
    required this.prompt,
    required this.options,
    required this.answerIndex,
    required this.explanation,
    this.highlight,
  });

  factory Question.fromJson(Map<String, dynamic> json) {
    return Question(
      id: json['id'] as String,
      type: questionTypeFromString(json['type'] as String),
      prompt: json['prompt'] as String,
      options: (json['options'] as List).cast<String>(),
      answerIndex: json['answer_index'] as int,
      explanation: json['explanation'] as String,
      highlight: json['highlight'] as String?,
    );
  }
}
