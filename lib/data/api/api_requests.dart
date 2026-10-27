typedef JsonMap = Map<String, dynamic>;

class CreatePatientRequest {
  const CreatePatientRequest({
    required this.displayName,
    required this.dateOfBirth,
    required this.policyVersion,
    required this.accepted,
  });

  final String displayName;
  final DateTime dateOfBirth;
  final String policyVersion;
  final bool accepted;

  JsonMap toJson() => {
    'display_name': displayName,
    'date_of_birth': _dateOnly(dateOfBirth),
    'policy_version': policyVersion,
    'accepted': accepted,
  };
}

class CreateDoctorRequest {
  const CreateDoctorRequest({
    required this.displayName,
    required this.dateOfBirth,
    required this.specialty,
  });

  final String displayName;
  final DateTime dateOfBirth;
  final String specialty;

  JsonMap toJson() => {
    'display_name': displayName,
    'date_of_birth': _dateOnly(dateOfBirth),
    'specialty': specialty,
  };
}

class CreateRelationshipRequest {
  const CreateRelationshipRequest({required this.patientCode});

  /// Код пациента вида PXXXXXX — связь создаётся сразу после его ввода.
  final String patientCode;

  JsonMap toJson() => {'patient_code': patientCode.trim()};
}

class CreateAssignmentRequest {
  const CreateAssignmentRequest({
    required this.patientId,
    required this.title,
    required this.startsAt,
    required this.endsAt,
    required this.frequency,
    required this.daysOfWeek,
    required this.timesOfDay,
    required this.type,
    required this.details,
  });

  final String patientId;
  final String title;
  final DateTime startsAt;
  final DateTime endsAt;

  /// Например, "DAILY" или "WEEKLY" — должно соответствовать серверу.
  final String frequency;

  /// Dart DateTime: понедельник = 1, воскресенье = 7.
  /// Конвертируем в Go time.Weekday: воскресенье = 0.
  final List<int> daysOfWeek;

  /// Локальное время, например "09:30".
  final List<String> timesOfDay;

  /// Должно совпадать с допустимыми значениями domain.AssignmentType в Go.
  final String type;

  /// Типизированные детали, например
  /// {"medication_id": "...", "dosage": {...}, "instructions": "..."}.
  final JsonMap details;

  JsonMap toJson() => {
    'patient_id': patientId,
    'title': title,
    'starts_at': startsAt.toUtc().toIso8601String(),
    'ends_at': endsAt.toUtc().toIso8601String(),
    'frequency': frequency,
    'days_of_week': daysOfWeek.map(_dartWeekdayToGo).toList(),
    'times_of_day': timesOfDay,
    'type': type,
    'details': details,
  };

  static int _dartWeekdayToGo(int weekday) {
    if (weekday < DateTime.monday || weekday > DateTime.sunday) {
      throw RangeError.range(weekday, 1, 7, 'weekday');
    }

    // Dart: Mon=1 ... Sun=7; Go: Sun=0, Mon=1 ... Sat=6.
    return weekday == DateTime.sunday ? 0 : weekday;
  }
}

class CompleteAssignmentRequest {
  const CompleteAssignmentRequest({
    this.comment = '',
    this.result = '',
    this.medicationIntake,
    this.sideEffects,
  });

  final String comment;
  final String result;
  final MedicationIntakeRequest? medicationIntake;
  final SideEffectReportRequest? sideEffects;

  JsonMap toJson() {
    final json = <String, dynamic>{'comment': comment, 'result': result};
    if (medicationIntake != null) {
      json['medication_intake'] = medicationIntake!.toJson();
    }
    if (sideEffects != null) {
      json['side_effects'] = sideEffects!.toJson();
    }
    return json;
  }
}

class MedicationIntakeRequest {
  const MedicationIntakeRequest({
    required this.takenAt,
    required this.reportedAsTaken,
    required this.dosageTaken,
  });

  final DateTime takenAt;
  final bool reportedAsTaken;
  final String dosageTaken;

  JsonMap toJson() => {
    'taken_at': takenAt.toUtc().toIso8601String(),
    'reported_as_taken': reportedAsTaken,
    'dosage_taken': dosageTaken,
  };
}

class SideEffectReportRequest {
  const SideEffectReportRequest({
    required this.effects,
    required this.comment,
    required this.reportedAt,
  });

  final List<String> effects;
  final String comment;
  final DateTime reportedAt;

  JsonMap toJson() => {
    'effects': effects,
    'comment': comment,
    'reported_at': reportedAt.toUtc().toIso8601String(),
  };
}

class CreateJournalEntryRequest {
  const CreateJournalEntryRequest({
    required this.text,
    this.reflectionPrompt = '',
  });

  final String text;
  final String reflectionPrompt;

  JsonMap toJson() => {'text': text, 'reflection_prompt': reflectionPrompt};
}

class CreateMoodCheckInRequest {
  const CreateMoodCheckInRequest({required this.answers, this.comment = ''});

  /// Каждый элемент должен соответствовать domain.MoodAnswer в Go.
  final List<JsonMap> answers;
  final String comment;

  JsonMap toJson() => {'answers': answers, 'comment': comment};
}

String _dateOnly(DateTime value) =>
    '${value.year.toString().padLeft(4, '0')}-'
    '${value.month.toString().padLeft(2, '0')}-'
    '${value.day.toString().padLeft(2, '0')}';
