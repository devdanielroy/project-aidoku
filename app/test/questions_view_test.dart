// QuestionsView's startAnswered mode in isolation — the default
// interactive path is already covered end to end via core_loop_test.dart
// and friends; this is just the "already passed, show it pre-answered"
// branch (see ChunkReviewSessionScreen), cheap to test directly since
// QuestionsView has no network/repository dependency at all.

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:aidoku/models/question.dart';
import 'package:aidoku/screens/session/questions_view.dart';

const _questions = [
  Question(
    id: 1,
    chunkId: 1,
    type: QuestionType.vocab,
    prompt: 'prompt one',
    options: ['right', 'wrong a', 'wrong b', 'wrong c'],
    answerIndex: 0,
    explanation: 'explanation one',
  ),
  Question(
    id: 2,
    chunkId: 1,
    type: QuestionType.grammar,
    prompt: 'prompt two',
    options: ['wrong a', 'right', 'wrong b', 'wrong c'],
    answerIndex: 1,
    explanation: 'explanation two',
  ),
];

void main() {
  testWidgets('startAnswered: false leaves questions neutral until tapped', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: QuestionsView(questions: _questions, onComplete: () {}),
        ),
      ),
    );

    // Nothing selected yet - no advance button, no explanation.
    expect(find.text('Next question'), findsNothing);
    expect(find.text('explanation one'), findsNothing);
  });

  testWidgets(
    'startAnswered: true pre-selects the correct option on every question, no tap needed',
    (WidgetTester tester) async {
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: QuestionsView(
              questions: _questions,
              onComplete: () {},
              startAnswered: true,
            ),
          ),
        ),
      );

      // Answered immediately, without tapping anything - the advance
      // button and explanation are already there.
      expect(find.text('Next question'), findsOneWidget);
      expect(find.text('explanation one'), findsOneWidget);

      await tester.tap(find.text('Next question'));
      await tester.pump();

      // The second (last) question is pre-filled too, not just the first.
      expect(find.text('See the full breakdown'), findsOneWidget);
      expect(find.text('explanation two'), findsOneWidget);
    },
  );

  testWidgets(
    'startAnswered: true never calls onAnswered - nothing was actually re-answered',
    (WidgetTester tester) async {
      var calls = 0;
      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: QuestionsView(
              questions: _questions,
              onComplete: () {},
              startAnswered: true,
              onAnswered: (_, _) => calls++,
            ),
          ),
        ),
      );
      await tester.tap(find.text('Next question'));
      await tester.pump();

      expect(calls, 0);
    },
  );
}
