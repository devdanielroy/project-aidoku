import 'package:flutter/material.dart';

import '../models/book.dart';
import '../models/chunk.dart';
import 'session/chunk_panel.dart';
import 'session/complete_view.dart';

/// Owns the whole reading session for one book: which chunk we're on, and
/// whether the book (or, for now, mock excerpt) is complete. The entire
/// per-chunk loop — reading, questions, breakdown — is delegated to
/// ChunkPanel, which is fundamentally one continuous, persistent screen
/// per chunk rather than a stack of independent ones (see its doc
/// comment); this screen only needs to know when a chunk finishes and
/// whether another one follows.
class ReadingSessionScreen extends StatefulWidget {
  final Book book;

  const ReadingSessionScreen({super.key, required this.book});

  @override
  State<ReadingSessionScreen> createState() => _ReadingSessionScreenState();
}

class _ReadingSessionScreenState extends State<ReadingSessionScreen> {
  int _chunkIndex = 0;
  bool _complete = false;

  Chunk get _currentChunk => widget.book.chunks[_chunkIndex];
  int get _totalChunks => widget.book.chunks.length;

  double get _progress => _complete ? 1.0 : _chunkIndex / _totalChunks;

  void _onChunkComplete() {
    if (_chunkIndex + 1 < _totalChunks) {
      setState(() => _chunkIndex++);
    } else {
      setState(() => _complete = true);
    }
  }

  void _onRestart() {
    setState(() {
      _chunkIndex = 0;
      _complete = false;
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
      body: _complete
          ? CompleteView(book: widget.book, onRestart: _onRestart)
          : ChunkPanel(
              key: ValueKey('chunk-${_currentChunk.id}'),
              chunk: _currentChunk,
              chunkNumber: _chunkIndex + 1,
              totalChunks: _totalChunks,
              isLastChunk: _chunkIndex + 1 >= _totalChunks,
              onChunkComplete: _onChunkComplete,
            ),
    );
  }
}
