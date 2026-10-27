typedef Id = String; // UUID в строковом представлении

enum AccountRole { patient, doctor, admin }

enum AssignmentType { medication, questionnaire, appCheckIn, journalTask }

enum AssignmentStatus { active, completed, paused, expired, cancelled }

enum RelationshipStatus { active, ended }

enum StatusColor { green, yellow, red }

enum StatusReasonKind { missedMedications, stablyLowQuestionnaireResults }

enum NotificationType {
  patientStatusChanged,
  medicationSideEffectReported,
  assignmentReminder,
  questionnaireCompleted,
}

class UserCode {
  const UserCode({required this.value, required this.role});

  /// Длина кода вместе с префиксом: PXXXXXX или VXXXXXX.
  static const int length = 7;

  final String value;
  final AccountRole role;

  /// Проверяет формат кода, который отображается вместо персональных данных.
  /// Точный алфавит суффикса задаётся здесь и дублируется на сервере.
  bool isValid() {
    final normalized = value.trim();
    if (normalized.length != length) {
      return false;
    }

    final prefix = switch (role) {
      AccountRole.patient => 'P',
      AccountRole.doctor => 'V',
      // Администратор — заглушка и не имеет рабочего кода.
      AccountRole.admin => null,
    };

    if (prefix == null || !normalized.startsWith(prefix)) {
      return false;
    }

    return _suffixPattern.hasMatch(normalized.substring(1));
  }

  static final RegExp _suffixPattern = RegExp(r'^[A-Z0-9]{6}$');
}

class PersonalData {
  const PersonalData({required this.displayName, this.dateOfBirth});

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

  const Period({required this.startsAt, required this.endsAt});

  bool contains(DateTime value) =>
      !value.isBefore(startsAt) && !value.isAfter(endsAt);
}

class Schedule {
  const Schedule({
    required this.frequency,
    required this.daysOfWeek,
    required this.timesOfDay,
  });

  /// Например: "DAILY" или "WEEKLY". Формат следует закрепить в API.
  final String frequency;
  final List<int> daysOfWeek; // 1 = понедельник, 7 = воскресенье
  final List<String> timesOfDay; // Локальное время пациента, например "09:30"
}

class Dosage {
  const Dosage({
    required this.amount,
    required this.unit,
    required this.instructions,
  });

  final double amount;
  final String unit;
  final String instructions;
}
