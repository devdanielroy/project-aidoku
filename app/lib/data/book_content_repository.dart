import 'dart:convert';

import 'package:http/http.dart' as http;

import '../config.dart';
import '../models/book.dart';
import '../models/breakdown.dart';
import '../models/chunk.dart';
import '../models/chunk_summary.dart';
import '../models/loaded_chunk.dart';
import '../models/question.dart';

/// Thrown when a book-content request fails — a network error, an
/// unexpected status code, or a response body that doesn't parse the
/// way the caller expected. Callers (see LibraryScreen's FutureBuilder)
/// show this via `.toString()`, so the message is written to be
/// reasonable to show a user, not just a developer.
class BookContentException implements Exception {
  final String message;

  const BookContentException(this.message);

  @override
  String toString() => message;
}

/// Talks to the book-content service (see /book-content at the repo
/// root) over HTTP. Read-only, matching the service itself — nothing
/// here writes anything. Base URL comes from AppConfig, not a
/// constructor default, so which backend this points at is a build-time
/// decision (--dart-define), not scattered across call sites.
class BookContentRepository {
  final http.Client _client;
  final String _baseUrl;

  BookContentRepository({http.Client? client, String? baseUrl})
    : _client = client ?? http.Client(),
      _baseUrl = baseUrl ?? AppConfig.bookContentBaseUrl;

  Future<List<Book>> getBooks() async {
    final json = await _getJson('/aidoku/books');
    final books = json['books'] as List;
    return books.map((b) => Book.fromJson(b as Map<String, dynamic>)).toList();
  }

  Future<Book> getBook(int bookId) async {
    return Book.fromJson(await _getJson('/aidoku/book/$bookId'));
  }

  /// Ordered by reading position — bare ids, not full chunk objects; see
  /// Chunk's own doc comment for why book-content's list endpoint is
  /// shaped that way.
  Future<List<int>> getChunkIds(int bookId) async {
    final json = await _getJson('/aidoku/book/$bookId/chunks');
    return (json['chunk_ids'] as List).cast<int>();
  }

  Future<Chunk> getChunk(int chunkId) async {
    return Chunk.fromJson(await _getJson('/aidoku/chunk/$chunkId'));
  }

  /// Teasers for every chunk in bookID, in reading order — for the chunk
  /// review list (see ChunkListScreen), which needs something to show
  /// per row without fetching every chunk's full text just for that.
  Future<List<ChunkSummary>> getChunkSummaries(int bookId) async {
    final json = await _getJson('/aidoku/book/$bookId/chunks/summary');
    final chunks = json['chunks'] as List;
    return chunks
        .map((c) => ChunkSummary.fromJson(c as Map<String, dynamic>))
        .toList();
  }

  Future<List<int>> getQuestionIds(int chunkId) async {
    final json = await _getJson('/aidoku/chunk/$chunkId/questions');
    return (json['question_ids'] as List).cast<int>();
  }

  Future<Question> getQuestion(int questionId) async {
    return Question.fromJson(await _getJson('/aidoku/question/$questionId'));
  }

  Future<Breakdown> getBreakdown(int chunkId) async {
    return Breakdown.fromJson(
      await _getJson('/aidoku/chunk/$chunkId/breakdown'),
    );
  }

  /// Fetches everything ChunkPanel needs for chunkId — the chunk text,
  /// all 3 questions, and the breakdown — as one call. The question-id
  /// list is fetched first (each question needs its own id), then every
  /// question and the breakdown are fetched concurrently. Fetching the
  /// breakdown now rather than only once all 3 questions are answered
  /// is deliberate: it means BreakdownView never has to show its own
  /// loading spinner right when the reader expects an instant
  /// transition — see LoadedChunk's own doc comment. The cost (an
  /// occasionally-wasted fetch, if the reader abandons the chunk
  /// mid-questions) is cheap against a local/nearby service.
  Future<LoadedChunk> loadChunk(int chunkId) async {
    final chunk = await getChunk(chunkId);
    final questionIds = await getQuestionIds(chunkId);
    final results = await Future.wait([
      Future.wait(questionIds.map(getQuestion)),
      getBreakdown(chunkId),
    ]);
    return LoadedChunk(
      chunk: chunk,
      questions: results[0] as List<Question>,
      breakdown: results[1] as Breakdown,
    );
  }

  Future<Map<String, dynamic>> _getJson(String path) async {
    final uri = Uri.parse('$_baseUrl$path');
    final http.Response response;
    try {
      response = await _client.get(uri);
    } catch (e) {
      throw BookContentException('Couldn\'t reach the book service: $e');
    }
    if (response.statusCode != 200) {
      throw BookContentException(
        '$path returned ${response.statusCode}: ${response.body}',
      );
    }
    try {
      return jsonDecode(response.body) as Map<String, dynamic>;
    } on FormatException catch (e) {
      throw BookContentException('$path returned invalid JSON: $e');
    }
  }
}
