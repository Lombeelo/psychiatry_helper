import 'common.dart';

sealed class AssignmentDetails {
  const AssignmentDetails();
}

class MedicationDetails extends AssignmentDetails {
  const MedicationDetails({
    required this.medicationId,
    required this.dosage,
    required this.instructions,
  });

  final Id medicationId;
  final Dosage dosage;
  final String instructions;
}

class QuestionnaireDetails extends AssignmentDetails {
  const QuestionnaireDetails({
    required this.questionnaireId,
    required this.questionnaireVersionId,
  });

  final Id questionnaireId;
  final Id questionnaireVersionId;
}

class AppCheckInDetails extends AssignmentDetails {
  const AppCheckInDetails(this.instructions);

  final String instructions;
}

class JournalTaskDetails extends AssignmentDetails {
  const JournalTaskDetails(this.prompt);

  final String prompt;
}

class Assignment {
  const Assignment({
    required this.assignmentId,
    required this.patientId,
    required this.doctorId,
    required this.type,
    required this.title,
    required this.period,
    required this.schedule,
    required this.status,
    required this.details,
    required this.createdAt,
  });

  final Id assignmentId;
  final Id patientId;
  final Id doctorId;
  final AssignmentType type;
  final String title;
  final Period period;
  final Schedule schedule;
  final AssignmentStatus status;
  final AssignmentDetails details;
  final DateTime createdAt;

  Assignment pause() => _changeStatus(AssignmentStatus.paused);
  Assignment resume() => _changeStatus(AssignmentStatus.active);
  Assignment cancel() => _changeStatus(AssignmentStatus.cancelled);
  Assignment close() => _changeStatus(AssignmentStatus.completed);

  Assignment _changeStatus(AssignmentStatus next) {
    final allowed = switch (status) {
      AssignmentStatus.active => {
          AssignmentStatus.paused,
          AssignmentStatus.completed,
          AssignmentStatus.cancelled,
          AssignmentStatus.expired,
        },
      AssignmentStatus.paused => {
          AssignmentStatus.active,
          AssignmentStatus.cancelled,
          AssignmentStatus.expired,
        },
      AssignmentStatus.completed ||
      AssignmentStatus.expired ||
      AssignmentStatus.cancelled =>
        <AssignmentStatus>{},
    };

    if (!allowed.contains(next)) {
      throw StateError('Недопустимый переход: $status → $next');
    }

    return Assignment(
      assignmentId: assignmentId,
      patientId: patientId,
      doctorId: doctorId,
      type: type,
      title: title,
      period: period,
      schedule: schedule,
      status: next,
      details: details,
      createdAt: createdAt,
    );
  }
}

class MedicationIntake {
  const MedicationIntake({
    required this.takenAt,
    required this.reportedAsTaken,
    this.dosageTaken,
  });

  final DateTime takenAt;
  final bool reportedAsTaken;
  final String? dosageTaken;
}

class SideEffectReport {
  const SideEffectReport({
    required this.effects,
    required this.reportedAt,
    this.comment,
  });

  final List<String> effects;
  final DateTime reportedAt;
  final String? comment;
}

class Completion {
  const Completion({
    required this.completionId,
    required this.assignmentId,
    required this.patientId,
    required this.completedAt,
    this.comment,
    this.result,
    this.medicationIntake,
    this.sideEffectReport,
  });

  final Id completionId;
  final Id assignmentId;
  final Id patientId;
  final DateTime completedAt;
  final String? comment;
  final String? result;
  final MedicationIntake? medicationIntake;
  final SideEffectReport? sideEffectReport;
}
