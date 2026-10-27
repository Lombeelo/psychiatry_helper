import 'common.dart';

enum SupportContentType { encouragement, reflection, contactDoctorPrompt }

/// Заранее подготовленное поддерживающее сообщение.
/// Не содержит медицинских советов и рекомендаций по лечению.
class SupportContent {
  const SupportContent({
    required this.contentId,
    required this.trigger,
    required this.message,
    required this.contentType,
  });

  final Id contentId;
  final String trigger;
  final String message;
  final SupportContentType contentType;
}

class ForestActivity {
  const ForestActivity({
    required this.activityType,
    required this.occurredAt,
    required this.points,
  });

  final String activityType;
  final DateTime occurredAt;
  final int points;
}

/// Состояние личного леса. Лес не меняет медицинский статус пациента.
class ForestState {
  const ForestState({
    required this.forestId,
    required this.patientId,
    required this.growthLevel,
    required this.activities,
    required this.updatedAt,
  });

  final Id forestId;
  final Id patientId;
  final int growthLevel;
  final List<ForestActivity> activities;
  final DateTime updatedAt;

  ForestState applyActivity(ForestActivity activity) {
    if (activity.points < 0) {
      throw ArgumentError.value(activity.points, 'points', 'не может быть < 0');
    }

    // Повторное открытие той же записи не должно начислять прогресс снова.
    final alreadyApplied = activities.any(
      (existing) =>
          existing.activityType == activity.activityType &&
          existing.occurredAt == activity.occurredAt,
    );
    if (alreadyApplied) {
      return this;
    }

    return ForestState(
      forestId: forestId,
      patientId: patientId,
      growthLevel: growthLevel + activity.points,
      activities: [...activities, activity],
      updatedAt: activity.occurredAt,
    );
  }
}
