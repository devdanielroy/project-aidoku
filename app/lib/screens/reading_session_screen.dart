import 'package:flutter/material.dart';

import '../data/book_content_repository.dart';
import '../data/progress_store.dart';
import '../models/book.dart';
import '../models/loaded_chunk.dart';
import 'session/chunk_panel.dart';
import 'session/complete_view.dart';

/// Owns the whole reading session for one book: which chunk we're on,
/// fetching each chunk's full content (text + questions + breakdown) as
/// the reader reaches it, resuming/saving progress via [progressStore],
/// and whether the book is complete. The per-chunk loop itself —
/// reading, questions, breakdown — is delegated to ChunkPanel (see its
/// doc comment); this screen only needs to know the current chunk's id,
/// fetch its content, and know when to advance.
///
/// Chunk ids are fetched once up front (GET .../chunks — cheap, ids
/// only); each chunk's actual content is fetched lazily, one at a time,
/// as the reader reaches it — not all up front, matching how the
/// reading flow is actually used (see BookContentRepository.loadChunk).
class ReadingSessionScreen extends StatefulWidget {
  final Book book;
  final BookContentRepository repository;
  final ProgressStore progressStore;

  const ReadingSessionScreen({
    super.key,
    required this.book,
    required this.repository,
    required this.progressStore,
  });

  @override
  State<ReadingSessionScreen> createState() => _ReadingSessionScreenState();
}

class _ReadingSessionScreenState extends State<ReadingSessionScreen> {
  late final Future<List<int>> _chunkIdsFuture;

  List<int> _chunkIds = const [];
  int _chunkIndex = 0;
  bool _complete = false;
  Future<LoadedChunk>? _currentChunkFuture;

  @override
  void initState() {
    super.initState();
    _chunkIdsFuture = widget.repository.getChunkIds(widget.book.id);
    // try/catch here just stops this specific listener from surfacing an
    // "unhandled exception" warning; the FutureBuilder below awaits the
    // same _chunkIdsFuture independently and is what actually shows the
    // error.
    _loadInitialChunk();
  }

  Future<void> _loadInitialChunk() async {
    try {
      final ids = await _chunkIdsFuture;
      if (!mounted || ids.isEmpty) return;
      // Resume where the reader left off, if anywhere — a saved index
      // past the end (the book got shorter somehow, or it's stale) just
      // falls back to the start rather than crashing on a bad list
      // index.
      final saved = await widget.progressStore.getChunkIndex(widget.book.id);
      if (!mounted) return;
      final startIndex = (saved != null && saved >= 0 && saved < ids.length)
          ? saved
          : 0;
      setState(() {
        _chunkIds = ids;
        _chunkIndex = startIndex;
        _currentChunkFuture = widget.repository.loadChunk(ids[startIndex]);
      });
    } catch (_) {
      // Swallowed deliberately — see this method's call site.
    }
  }

  double get _progress {
    if (_complete) return 1.0;
    if (_chunkIds.isEmpty) return 0.0;
    return _chunkIndex / _chunkIds.length;
  }

  void _onChunkComplete() {
    if (_chunkIndex + 1 < _chunkIds.length) {
      final nextIndex = _chunkIndex + 1;
      widget.progressStore.saveChunkIndex(widget.book.id, nextIndex);
      setState(() {
        _chunkIndex = nextIndex;
        _currentChunkFuture = widget.repository.loadChunk(_chunkIds[nextIndex]);
      });
    } else {
      // Nothing left to resume — see ProgressStore.clearProgress.
      widget.progressStore.clearProgress(widget.book.id);
      setState(() => _complete = true);
    }
  }

  void _onRestart() {
    widget.progressStore.saveChunkIndex(widget.book.id, 0);
    setState(() {
      _chunkIndex = 0;
      _complete = false;
      _currentChunkFuture = widget.repository.loadChunk(_chunkIds[0]);
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.book.title),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(4),
          child: LinearProgressIndicator(value: _progress),
        ),
      ),
      body: FutureBuilder<List<int>>(
        future: _chunkIdsFuture,
        builder: (context, chunkIdsSnapshot) {
          if (chunkIdsSnapshot.connectionState != ConnectionState.done) {
            return const Center(child: CircularProgressIndicator());
          }
          if (chunkIdsSnapshot.hasError) {
            return Center(
              child: Text('Failed to load chunks: ${chunkIdsSnapshot.error}'),
            );
          }
          final chunkIds = chunkIdsSnapshot.data!;
          if (chunkIds.isEmpty) {
            return const Center(child: Text('This book has no chunks yet.'));
          }
          if (_complete) {
            return CompleteView(
              book: widget.book,
              totalChunks: chunkIds.length,
              onRestart: _onRestart,
            );
          }
          return FutureBuilder<LoadedChunk>(
            future: _currentChunkFuture,
            builder: (context, chunkSnapshot) {
              if (chunkSnapshot.connectionState != ConnectionState.done) {
                return const Center(child: CircularProgressIndicator());
              }
              if (chunkSnapshot.hasError) {
                return Center(
                  child: Text('Failed to load chunk: ${chunkSnapshot.error}'),
                );
              }
              final loaded = chunkSnapshot.data!;
              return ChunkPanel(
                key: ValueKey('chunk-${loaded.chunk.id}'),
                loadedChunk: loaded,
                chunkNumber: _chunkIndex + 1,
                totalChunks: chunkIds.length,
                isLastChunk: _chunkIndex + 1 >= chunkIds.length,
                onChunkComplete: _onChunkComplete,
              );
            },
          );
        },
      ),
    );
  }
}
