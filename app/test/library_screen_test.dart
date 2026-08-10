import 'package:flutter_test/flutter_test.dart';

import 'package:aidoku/main.dart';

void main() {
  testWidgets('library screen shows the mock book', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(const AidokuApp());
    await tester.pumpAndSettle(); // wait for the async asset load

    expect(find.text('Pride and Prejudice'), findsOneWidget);
  });
}
