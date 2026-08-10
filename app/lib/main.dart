import 'package:flutter/material.dart';

import 'screens/library_screen.dart';

void main() {
  runApp(const AidokuApp());
}

class AidokuApp extends StatelessWidget {
  const AidokuApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Aidoku',
      // Placeholder seed color, not a locked-in brand choice.
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.indigo),
        useMaterial3: true,
      ),
      home: const LibraryScreen(),
    );
  }
}
