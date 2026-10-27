import 'common.dart';

class StatusReason {
  const StatusReason({
    required this.kind,
    required this.description,
    required this.observedValue,
    required this.period,
  });

  final StatusReasonKind kind;
  final String description;
  final String observedValue;
  final Period period;
}

class PatientStatus {
  const PatientStatus({
    required this.color,
    required this.reasons,
    required this.calculatedAt,
  });

  final StatusColor color;
  final List<StatusReason> reasons;
  final DateTime calculatedAt;
}

/// Результат анализа пропусков приёма лекарств.
class MedicationAdherence {
  const MedicationAdherence({required this.missedCount, required this.period});

  final int missedCount;
  final Period period;
}

/// Результат анализа последовательных низких результатов
/// в рамках одного опросника и его версии.
class QuestionnaireTrend {
  const QuestionnaireTrend({
    required this.questionnaireId,
    required this.questionnaireVersionId,
    required this.consecutiveLowResults,
    required this.threshold,
  });

  final Id questionnaireId;
  final Id questionnaireVersionId;
  final int consecutiveLowResults;
  final double threshold;
}

/// Доменный сервис расчёта индикатора состояния.
/// Пороги и период оценки задаются явными полями, чтобы индикатор
/// не выглядел диагнозом или автоматической оценкой экстренности.
abstract class StatusPolicy {
  const StatusPolicy({
    required this.missedMedicationThreshold,
    required this.missedMedicationPeriod,
    required this.lowScoreCountThreshold,
  });

  final int missedMedicationThreshold;
  final Duration missedMedicationPeriod;
  final int lowScoreCountThreshold;

  Future<PatientStatus> evaluate(Id patientId);
}

class Notification {
  const Notification({
    required this.notificationId,
    required this.recipientId,
    required this.patientId,
    required this.type,
    required this.message,
    required this.createdAt,
    this.readAt,
  });

  final Id notificationId;
  final Id recipientId;
  final Id patientId;
  final NotificationType type;
  final String message;
  final DateTime createdAt;
  final DateTime? readAt;
}
