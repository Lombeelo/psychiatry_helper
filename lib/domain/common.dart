typedef Id = String; // UUID в строковом представлении

enum AccountRole { patient, doctor, admin }

enum AssignmentType {
  medication,
  questionnaire,
  appCheckIn,
  journalTask,
}

enum AssignmentStatus {
  active,
  completed,
  paused,
  expired,
  cancelled,
}

enum RelationshipStatus { active, ended }

enum StatusColor { green, yellow, red }

enum StatusReasonKind {
  missedMedications,
  stablyLowQuestionnaireResults,
}

enum NotificationType {
  patientStatusChanged,
  medicationSideEffectReported,
  assignmentReminder,
  questionnaireCompleted,
}

class UserCode {
  const UserCode({
    required this.value,
    required this.role,
  });

  final String value;
  final AccountRole role;

  bool get hasExpectedPrefix {
    final prefix = switch (role) {
      AccountRole.patient => 'P',
      AccountRole.doctor => 'V',
      AccountRole.admin => '',
    };

    return prefix.isEmpty || value.startsWith(prefix);
  }

  // Проверку точного формата XXXXXX выполняйте по конфигурации сервера.
}

class PersonalData {
  const PersonalData({
    required this.displayName,
    this.dateOfBirth,
  });

  final String displayName;
  final DateTime? dateOfBirth;
}

class Consent {
  const Consent({
    required this.accepted,
    required this.policyVersion,
    this.acceptedAt,
  });

  final bool accepted;
  final String policyVersion;
  final DateTime? acceptedAt;
}

class Period {
  final DateTime startsAt;
  final DateTime endsAt;

  const Period({
    required this.startsAt,
    required this.endsAt,
  });


  bool contains(DateTime value) =>
      !value.isBefore(startsAt) && !value.isAfter(endsAt);
}

class Schedule {
  const Schedule({
    required this.frequency,
    required this.daysOfWeek,
    required this.timesOfDay,
    required this.timeZoneId,
  });

  /// Например: "DAILY" или "WEEKLY". Формат следует закрепить в API.
  final String frequency;
  final List<int> daysOfWeek; // 1 = понедельник, 7 = воскресенье
  final List<String> timesOfDay; // Локальное время, например "09:30"
  final String timeZoneId; // Например, "Europe/Moscow"
}

class Dosage {
  const Dosage({
    required this.amount,
    required this.unit,
    this.instructions,
  });

  final double amount;
  final String unit;
  final String? instructions;
}
