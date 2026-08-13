import 'package:flutter/material.dart';

import '../data/book_content_repository.dart';
import '../models/book.dart';
import '../models/loaded_chunk.dart';
import 'session/chunk_panel.dart';
import 'session/complete_view.dart';

/// Owns the whole reading session for one book: which chunk we're on,
/// fetching each chunk's full content (text + questions + breakdown) as
/// the reader reaches it, and whether the book is complete. The
/// per-chunk loop itself — reading, questions, breakdown — is delegated
/// to ChunkPanel (see its doc comment); this screen only needs to know
/// the current chunk's id, fetch its content, and know when to advance.
///
/// Chunk ids are fetched once up front (GET .../chunks — cheap, ids
/// only); each chunk's actual content is fetched lazily, one at a time,
/// as the reader reaches it — not all up front, matching how the
/// reading flow is actually used (see BookContentRepository.loadChunk).
class ReadingSessionScreen extends StatefulWidget {
  final Book book;
  final BookContentRepository repository;

  const ReadingSessionScreen({
    super.key,
    required this.book,
    required this.repository,
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
    // Kick off the first chunk's content fetch the moment the id list
    // arrives, rather than waiting for a rebuild to notice — .catchError
    // here just stops this specific listener from surfacing an "unhandled
    // exception" warning; the FutureBuilder below watches the same
    // _chunkIdsFuture independently and is what actually shows the error.
    _chunkIdsFuture
        .then((ids) {
          if (!mounted) return;
          setState(() {
            _chunkIds = ids;
            if (ids.isNotEmpty) {
              _currentChunkFuture = widget.repository.loadChunk(ids[0]);
            }
          });
        })
        .catchError((_) {});
  }

  double get _progress {
    if (_complete) return 1.0;
    if (_chunkIds.isEmpty) return 0.0;
    return _chunkIndex / _chunkIds.length;
  }

  void _onChunkComplete() {
    if (_chunkIndex + 1 < _chunkIds.length) {
      setState(() {
        _chunkIndex++;
        _currentChunkFuture = widget.repository.loadChunk(
          _chunkIds[_chunkIndex],
        );
      });
    } else {
      setState(() => _complete = true);
    }
  }

  void _onRestart() {
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
