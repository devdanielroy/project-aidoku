import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import 'score_store.dart';

/// ScoreStore backed by shared_preferences — this device's local
/// storage, not synced anywhere. See ProgressStore's own doc comment
/// (LocalScoreStore's sibling) for the intended migration path once
/// there's a real user account.
class LocalScoreStore implements ScoreStore {
  static const _keyPrefix = 'score.answers.book_';

  @override
  Future<void> recordAnswer({
    required int bookId,
    required int questionId,
    required bool correct,
  }) async {
    final prefs = await SharedPreferences.getInstance();
    final answers = await getAnswers(bookId);
    answers[questionId] = correct;
    await prefs.setString('$_keyPrefix$bookId', _encode(answers));
  }

  @override
  Future<Map<int, bool>> getAnswers(int bookId) async {
    final prefs = await SharedPreferences.getInstance();
    final raw = prefs.getString('$_keyPrefix$bookId');
    if (raw == null) return {};
    return _decode(raw);
  }

  @override
  Future<void> clearScore(int bookId) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove('$_keyPrefix$bookId');
  }

  // Stored as a JSON object of string question-id -> bool (JSON object
  // keys are always strings; question ids are parsed back to int on the
  // way out).
  String _encode(Map<int, bool> answers) =>
      jsonEncode(answers.map((id, correct) => MapEntry('$id', correct)));

  Map<int, bool> _decode(String raw) {
    final json = jsonDecode(raw) as Map<String, dynamic>;
    return json.map((id, correct) => MapEntry(int.parse(id), correct as bool));
  }
}
