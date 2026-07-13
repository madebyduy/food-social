import 'package:flutter_test/flutter_test.dart';
import 'package:mobile/main.dart';

void main() {
  testWidgets('AnNgon renders the auth gate first', (tester) async {
    await tester.pumpWidget(const AnNgonApp());

    expect(find.text('Chào mừng quay lại'), findsOneWidget);
    expect(find.text('Đăng nhập'), findsAtLeastNWidgets(1));
    expect(find.text('Đăng ký'), findsOneWidget);
  });
}
