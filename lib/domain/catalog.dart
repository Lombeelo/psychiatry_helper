import 'common.dart';

/// Группа препаратов из справочника (побочные эффекты, болезни,
/// скорость наступления эффекта описываются в карточке группы).
class MedicationGroup {
  const MedicationGroup({
    required this.groupId,
    required this.name,
    required this.description,
  });

  final Id groupId;
  final String name;
  final String description;
}

/// Препарат из справочника. Приложение не назначает препарат само:
/// врач выбирает его из справочника.
class Medication {
  const Medication({
    required this.medicationId,
    required this.name,
    required this.groupId,
    required this.instructionText,
  });

  final Id medicationId;
  final String name;
  final Id groupId;
  final String instructionText;
}
