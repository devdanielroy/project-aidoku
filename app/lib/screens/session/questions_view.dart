import 'package:flutter/material.dart';

import '../../models/chunk.dart';
import '../../models/question.dart';

/// Step 2 of the core loop (AIDOKU_DESIGN.md §2 step 4): three questions
/// per chunk (vocab, grammar, comprehension), answered one at a time in
/// that order. Each question gives immediate feedback via its own
/// `explanation`; the chunk's full `breakdown` is reserved for after all
/// three are answered — see BreakdownView.
class QuestionsView extends StatefulWidget {
  final Chunk chunk;
  final VoidCallback onComplete;

  /// Called whenever the displayed question changes (including once for
  /// the first question, right after the initial frame) so a parent that
  /// also shows the passage — see ReadingWithQuestionsPanel — can
  /// underline that question's `highlight` span in the text instead of
  /// this view re-quoting it.
  final ValueChanged<Question>? onQuestionChanged;

  const QuestionsView({
    super.key,
    required this.chunk,
    required this.onComplete,
    this.onQuestionChanged,
  });

  @override
  State<QuestionsView> createState() => _QuestionsViewState();
}

class _QuestionsViewState extends State<QuestionsView> {
  int _questionIndex = 0;
  int? _selectedOption;

  Question get _question => widget.chunk.questions[_questionIndex];
  bool get _isLastQuestion =>
      _questionIndex == widget.chunk.questions.length - 1;

  @override
  void initState() {
    super.initState();
    // Deferred to after the first frame: calling widget.onQuestionChanged
    // synchronously here could trigger setState on an ancestor that's
    // still mid-build (this widget is being built as part of it).
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (mounted) widget.onQuestionChanged?.call(_question);
    });
  }

  void _selectOption(int index) {
    if (_selectedOption != null) return; // already answered — locked in
    setState(() => _selectedOption = index);
  }

  void _advance() {
    if (_isLastQuestion) {
      widget.onComplete();
      return;
    }
    setState(() {
      _questionIndex++;
      _selectedOption = null;
    });
    widget.onQuestionChanged?.call(_question);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final question = _question;
    final answered = _selectedOption != null;

    // top: false — this view now always lives inside the bottom sheet
    // panel (see ReadingWithQuestionsPanel), not flush against the
    // physical top of the screen, so there's no top inset to protect.
    //
    // The question/options/explanation scroll (the sheet's height is a
    // fixed fraction of the screen, not a fit-content size, so on a short
    // window or a long explanation this can be taller than the visible
    // area) but the advance button is pinned outside that scroll area —
    // otherwise the one control the reader actually needs to tap next
    // could scroll out of view below the fold.
    return SafeArea(
      top: false,
      child: Column(
        children: [
          Expanded(
            child: SingleChildScrollView(
              padding: const EdgeInsets.fromLTRB(24, 24, 24, 12),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    _questionTypeLabel(question.type),
                    style: theme.textTheme.labelLarge?.copyWith(
                      color: theme.colorScheme.secondary,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    'Question ${_questionIndex + 1} of ${widget.chunk.questions.length}',
                    style: theme.textTheme.bodySmall,
                  ),
                  const SizedBox(height: 16),
                  Text(question.prompt, style: theme.textTheme.titleLarge),
                  const SizedBox(height: 24),
                  for (var i = 0; i < question.options.length; i++) ...[
                    if (i > 0) const SizedBox(height: 12),
                    _OptionTile(
                      text: question.options[i],
                      state: !answered
                          ? _OptionState.neutral
                          : i == question.answerIndex
                          ? _OptionState.correct
                          : i == _selectedOption
                          ? _OptionState.incorrect
                          : _OptionState.disabled,
                      onTap: () => _selectOption(i),
                    ),
                  ],
                  if (answered) ...[
                    const SizedBox(height: 16),
                    Container(
                      padding: const EdgeInsets.all(16),
                      decoration: BoxDecoration(
                        color: theme.colorScheme.surfaceContainerHighest,
                        borderRadius: BorderRadius.circular(12),
                      ),
                      child: Text(
                        question.explanation,
                        style: theme.textTheme.bodyMedium?.copyWith(
                          height: 1.5,
                        ),
                      ),
                    ),
                  ],
                ],
              ),
            ),
          ),
          if (answered)
            Padding(
              padding: const EdgeInsets.fromLTRB(24, 0, 24, 24),
              child: FilledButton(
                onPressed: _advance,
                child: Text(
                  _isLastQuestion ? 'See the full breakdown' : 'Next question',
                ),
              ),
            ),
        ],
      ),
    );
  }
}

String _questionTypeLabel(QuestionType type) {
  switch (type) {
    case QuestionType.vocab:
      return 'VOCABULARY';
    case QuestionType.grammar:
      return 'GRAMMAR';
    case QuestionType.comprehension:
      return 'COMPREHENSION';
  }
}

enum _OptionState { neutral, correct, incorrect, disabled }

class _OptionTile extends StatelessWidget {
  final String text;
  final _OptionState state;
  final VoidCallback onTap;

  const _OptionTile({
    required this.text,
    required this.state,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    Color? backgroundColor;
    Color borderColor;
    IconData? icon;

    switch (state) {
      case _OptionState.neutral:
        borderColor = colorScheme.outline;
      case _OptionState.correct:
        backgroundColor = colorScheme.primaryContainer;
        borderColor = colorScheme.primary;
        icon = Icons.check_circle;
      case _OptionState.incorrect:
        backgroundColor = colorScheme.errorContainer;
        borderColor = colorScheme.error;
        icon = Icons.cancel;
      case _OptionState.disabled:
        borderColor = colorScheme.outlineVariant;
    }

    return InkWell(
      onTap: state == _OptionState.neutral ? onTap : null,
      borderRadius: BorderRadius.circular(12),
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 14),
        decoration: BoxDecoration(
          color: backgroundColor,
          border: Border.all(color: borderColor),
          borderRadius: BorderRadius.circular(12),
        ),
        child: Row(
          children: [
            Expanded(child: Text(text)),
            if (icon != null) Icon(icon, color: borderColor),
          ],
        ),
      ),
    );
  }
}
