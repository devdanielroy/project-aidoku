// A fake book-content HTTP transport for widget tests — mirrors the real
// service's routes and response shapes exactly (see
// book-content/internal/api's route list), so BookContentRepository
// itself runs unmodified and gets exercised for real; only the HTTP
// transport is faked, via package:http's own MockClient (the
// package-sanctioned way to test http.Client-based code, not a bespoke
// hack). Deliberately not a call to a real running server — widget tests
// should be hermetic, not depend on `docker compose up -d`.

import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

const testBook = {
  'id': 1,
  'gutenberg_id': 1,
  'title': 'Test Book',
  'author': 'Test Author',
  'source_url': 'https://example.com/test.txt',
  'level': 5,
  'target_language': 'en',
  'native_language': 'ja',
  'status': 'published',
};

/// A second book in the opposite language pair from testBook — only
/// used by tests that need more than one available pair (see
/// language_pair_test.dart / settings_screen_test.dart /
/// library_screen_test.dart's filtering cases). Deliberately not
/// wired into testChunks/_questionsByChunk/_breakdownByChunk below —
/// nothing exercises reading this book, only its presence in the
/// /aidoku/books listing.
const testBookOtherPair = {
  'id': 2,
  'gutenberg_id': 2,
  'title': 'Other Pair Book',
  'author': 'Other Author',
  'source_url': 'https://example.com/other.txt',
  'level': 3,
  'target_language': 'ja',
  'native_language': 'en',
  'status': 'published',
};

const testChunks = [
  {
    'id': 101,
    'book_id': 1,
    'index': 0,
    'text':
        'It is a truth universally acknowledged, that a single man in '
        'possession of a good fortune, must be in want of a wife.',
    'char_count': 119,
  },
  {
    'id': 102,
    'book_id': 1,
    'index': 1,
    'text':
        'She was a woman of mean understanding, little information, '
        'and uncertain temper.',
    'char_count': 82,
  },
  {
    'id': 103,
    'book_id': 1,
    'index': 2,
    'text': 'When she was discontented she fancied herself nervous.',
    'char_count': 56,
  },
];

/// The correct option's text for each of chunk 101's 3 questions, in
/// question order (vocab, grammar, comprehension) — tests look these up
/// by text rather than by position, since QuestionsView now shuffles
/// options for display (see its own doc comment on why).
const chunk101CorrectAnswers = [
  'accepted as true',
  'logical certainty',
  'a common assumption about wealthy single men',
];

/// Same as [chunk101CorrectAnswers], for chunk 103 (the last of the 3
/// test chunks) — used to drive a full chunk completion in tests that
/// need to reach the "book finished" state (see resume_test.dart).
const chunk103CorrectAnswers = [
  'unhappy/dissatisfied',
  'past simple (passive-like adjective use)',
  'imagines herself nervous',
];

const _questionsByChunk = {
  101: [
    {
      'id': 1001,
      'chunk_id': 101,
      'type': 'vocab',
      // Deliberately doesn't quote the target word itself (it's already
      // underlined in the passage above) — a prompt that does would
      // create a second RichText containing "acknowledged", which is
      // exactly what highlight_test.dart's find.byWidgetPredicate
      // disambiguates against.
      'prompt': 'What does the underlined word mean here?',
      'options': ['accepted as true', 'denied', 'forgotten', 'questioned'],
      'answer_index': 0,
      'explanation': '"acknowledged" means accepted as true.',
      'highlight': 'acknowledged',
    },
    {
      'id': 1002,
      'chunk_id': 101,
      'type': 'grammar',
      'prompt': 'What does "must" express here?',
      'options': [
        'logical certainty',
        'obligation',
        'permission',
        'past habit',
      ],
      'answer_index': 0,
      'explanation': '"must" expresses logical certainty here.',
      'highlight': 'must',
    },
    {
      'id': 1003,
      'chunk_id': 101,
      'type': 'comprehension',
      'prompt': 'What is this sentence about?',
      'options': [
        'a common assumption about wealthy single men',
        'a legal requirement',
        'a marriage announcement',
        'a warning',
      ],
      'answer_index': 0,
      'explanation': 'It states a common social assumption.',
      'highlight': null,
    },
  ],
  102: [
    {
      'id': 1004,
      'chunk_id': 102,
      'type': 'vocab',
      'prompt': 'What does "mean" mean here?',
      'options': ['average/poor', 'unkind', 'intended', 'numeric average'],
      'answer_index': 0,
      'explanation': '"mean" here means average or poor quality.',
      'highlight': 'mean',
    },
    {
      'id': 1005,
      'chunk_id': 102,
      'type': 'grammar',
      'prompt': 'What kind of phrase is "of mean understanding"?',
      'options': [
        'a descriptive of-phrase',
        'a question',
        'a command',
        'a comparison',
      ],
      'answer_index': 0,
      'explanation': 'It describes a quality using "of + noun phrase".',
      'highlight': 'of mean understanding',
    },
    {
      'id': 1006,
      'chunk_id': 102,
      'type': 'comprehension',
      'prompt': 'What is this sentence describing?',
      'options': [
        "a woman's limited understanding and uncertain temper",
        'a wealthy man',
        'a wedding',
        'the weather',
      ],
      'answer_index': 0,
      'explanation': "It describes the woman's character.",
      'highlight': null,
    },
  ],
  103: [
    {
      'id': 1007,
      'chunk_id': 103,
      'type': 'vocab',
      'prompt': 'What does "discontented" mean?',
      'options': ['unhappy/dissatisfied', 'excited', 'confused', 'tired'],
      'answer_index': 0,
      'explanation': '"discontented" means unhappy or dissatisfied.',
      'highlight': 'discontented',
    },
    {
      'id': 1008,
      'chunk_id': 103,
      'type': 'grammar',
      'prompt': 'What tense is "was discontented"?',
      'options': [
        'past simple (passive-like adjective use)',
        'present continuous',
        'future',
        'past perfect',
      ],
      'answer_index': 0,
      'explanation': 'It uses "was" + adjective, past simple.',
      'highlight': 'was discontented',
    },
    {
      'id': 1009,
      'chunk_id': 103,
      'type': 'comprehension',
      'prompt': 'What does she do when discontented?',
      'options': [
        'imagines herself nervous',
        'goes for a walk',
        'writes a letter',
        'falls asleep',
      ],
      'answer_index': 0,
      'explanation': 'She fancied (imagined) herself nervous.',
      'highlight': null,
    },
  ],
};

const _breakdownByChunk = {
  101: '【文構造】Test breakdown for chunk 101.',
  102: '【文構造】Test breakdown for chunk 102.',
  103: '【文構造】Test breakdown for chunk 103.',
};

/// An http.Client whose responses are these fixtures — pass to
/// BookContentRepository(client: fakeBookContentClient()) in tests.
///
/// [books] overrides what GET /aidoku/books returns — defaults to just
/// testBook (every existing test's assumption), but a test exercising
/// language-pair filtering/selection (SettingsScreen, LibraryScreen's
/// empty states) can pass [testBook, testBookOtherPair] to make more
/// than one pair available. Every other endpoint stays tied to
/// testBook/testChunks regardless — nothing needs to actually read
/// testBookOtherPair's content.
http.Client fakeBookContentClient({
  List<Map<String, dynamic>> books = const [testBook],
}) {
  return MockClient((request) async {
    final path = request.url.path;

    if (path == '/aidoku/books') {
      return _ok({'books': books});
    }
    if (RegExp(r'^/aidoku/book/\d+$').hasMatch(path)) {
      return _ok(testBook);
    }
    if (RegExp(r'^/aidoku/book/\d+/chunks$').hasMatch(path)) {
      return _ok({'chunk_ids': testChunks.map((c) => c['id']).toList()});
    }
    if (RegExp(r'^/aidoku/book/\d+/chunks/summary$').hasMatch(path)) {
      return _ok({
        'chunks': testChunks
            .map(
              (c) => {
                'id': c['id'],
                'index': c['index'],
                // Real previews are server-truncated to the first
                // sentence (see book-content's firstSentencePreview) -
                // the fixture doesn't bother replicating that logic,
                // just returns the whole (short) test chunk text as-is.
                'preview': c['text'],
              },
            )
            .toList(),
      });
    }
    final chunkMatch = RegExp(r'^/aidoku/chunk/(\d+)$').firstMatch(path);
    if (chunkMatch != null) {
      final id = int.parse(chunkMatch.group(1)!);
      return _ok(testChunks.firstWhere((c) => c['id'] == id));
    }
    final questionsMatch = RegExp(
      r'^/aidoku/chunk/(\d+)/questions$',
    ).firstMatch(path);
    if (questionsMatch != null) {
      final chunkId = int.parse(questionsMatch.group(1)!);
      final questions = _questionsByChunk[chunkId]!;
      return _ok({'question_ids': questions.map((q) => q['id']).toList()});
    }
    final questionMatch = RegExp(r'^/aidoku/question/(\d+)$').firstMatch(path);
    if (questionMatch != null) {
      final id = int.parse(questionMatch.group(1)!);
      final question = _questionsByChunk.values
          .expand((qs) => qs)
          .firstWhere((q) => q['id'] == id);
      return _ok(question);
    }
    final breakdownMatch = RegExp(
      r'^/aidoku/chunk/(\d+)/breakdown$',
    ).firstMatch(path);
    if (breakdownMatch != null) {
      final chunkId = int.parse(breakdownMatch.group(1)!);
      return _ok({
        'id': chunkId,
        'chunk_id': chunkId,
        'content': _breakdownByChunk[chunkId],
      });
    }
    return http.Response(jsonEncode({'error': 'not found'}), 404);
  });
}

http.Response _ok(Object body) => http.Response(
  jsonEncode(body),
  200,
  headers: {'content-type': 'application/json'},
);
