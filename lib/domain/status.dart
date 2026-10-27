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

class AppNotification {
  const AppNotification({
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
