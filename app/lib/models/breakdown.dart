/// The full post-answer explanation of a chunk (vocab, grammar, meaning),
/// written in the learner's L1. See AIDOKU_DESIGN.md §2 step 5 / §4.
class Breakdown {
  final String id;
  final String content;

  const Breakdown({required this.id, required this.content});

  factory Breakdown.fromJson(Map<String, dynamic> json) {
    return Breakdown(
      id: json['id'] as String,
      content: json['content'] as String,
    );
  }
}
