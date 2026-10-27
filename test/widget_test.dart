import 'package:flutter_test/flutter_test.dart';

import 'package:psychiatry_helper/data/api/api_client.dart';
import 'package:psychiatry_helper/main.dart';

void main() {
  testWidgets('login flow starts with role selection', (tester) async {
    await tester.pumpWidget(
      CareApp(api: ApiClient(baseUrl: 'http://127.0.0.1:8080')),
    );

    expect(find.text('Сопровождение пациента'), findsOneWidget);
    expect(find.text('Я врач'), findsOneWidget);
    expect(find.text('Я пациент'), findsOneWidget);

    await tester.tap(find.text('Я врач'));
    await tester.pumpAndSettle();

    expect(find.text('Вход в систему'), findsOneWidget);
    expect(find.text('Логин'), findsOneWidget);
    expect(find.text('Пароль'), findsOneWidget);
  });
}
