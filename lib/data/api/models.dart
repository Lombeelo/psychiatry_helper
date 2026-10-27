typedef Json = Map<String, dynamic>;

Json asJson(dynamic value) {
  if (value is Map<String, dynamic>) {
    return value;
  }
  if (value is Map) {
    return value.map((key, item) => MapEntry(key.toString(), item));
  }
  return <String, dynamic>{};
}

List<Json> asJsonList(dynamic value) {
  if (value is List) {
    return value.map(asJson).toList(growable: false);
  }
  return const <Json>[];
}

String asString(dynamic value, [String fallback = '']) =>
    value is String ? value : fallback;

bool asBool(dynamic value) => value == true;

int asInt(dynamic value) {
  if (value is num) {
    return value.toInt();
  }
  return int.tryParse('$value') ?? 0;
}

DateTime? asDate(dynamic value) {
  if (value is String) {
    return DateTime.tryParse(value)?.toLocal();
  }
  return null;
}

class AuthSession {
  const AuthSession({
    required this.token,
    required this.accountId,
    required this.role,
    required this.code,
    required this.displayName,
    required this.consentAccepted,
    required this.anamnesis,
  });

  final String token;
  final String accountId;
  final String role;
  final String code;
  final String displayName;
  final bool consentAccepted;
  final String anamnesis;

  bool get isDoctor => role == 'DOCTOR';
  bool get isPatient => role == 'PATIENT';

  AuthSession copyWith({bool? consentAccepted, String? anamnesis}) {
    return AuthSession(
      token: token,
      accountId: accountId,
      role: role,
      code: code,
      displayName: displayName,
      consentAccepted: consentAccepted ?? this.consentAccepted,
      anamnesis: anamnesis ?? this.anamnesis,
    );
  }

  factory AuthSession.fromLogin(Json json) {
    final account = asJson(json['account']);
    return AuthSession(
      token: asString(json['token']),
      accountId: asString(account['account_id']),
      role: asString(account['role']),
      code: asString(account['code']),
      displayName: asString(account['display_name']),
      consentAccepted: asBool(json['consent_accepted']),
      anamnesis: asString(json['anamnesis']),
    );
  }
}

class StatusInfo {
  const StatusInfo({
    required this.color,
    required this.reasons,
    required this.calculatedAt,
    required this.dataSufficient,
  });

  final String color;
  final List<String> reasons;
  final DateTime? calculatedAt;
  final bool dataSufficient;

  bool get hasData => dataSufficient;

  String get label {
    if (!dataSufficient) {
      return 'Недостаточно данных';
    }
    return switch (color) {
      'RED' => 'Красный',
      'YELLOW' => 'Жёлтый',
      'GREEN' => 'Зелёный',
      _ => 'Нет данных',
    };
  }

  factory StatusInfo.fromJson(Json json) {
    return StatusInfo(
      color: asString(json['color'], 'GREEN'),
      reasons: asJsonList(json['reasons'])
          .map((reason) => asString(reason['description']))
          .where((reason) => reason.isNotEmpty)
          .toList(growable: false),
      calculatedAt: asDate(json['calculated_at']),
      dataSufficient: asBool(json['data_sufficient']),
    );
  }
}

class PatientSummary {
  const PatientSummary({
    required this.id,
    required this.code,
    required this.displayName,
    required this.notes,
    required this.status,
  });

  final String id;
  final String code;
  final String displayName;
  final String notes;
  final StatusInfo status;

  factory PatientSummary.fromJson(Json json) {
    return PatientSummary(
      id: asString(json['patient_id']),
      code: asString(json['code']),
      displayName: asString(json['display_name']),
      notes: asString(json['notes']),
      status: StatusInfo.fromJson(asJson(json['status'])),
    );
  }
}

class DoctorInfo {
  const DoctorInfo({
    required this.id,
    required this.code,
    required this.displayName,
    required this.specialty,
  });

  final String id;
  final String code;
  final String displayName;
  final String specialty;

  factory DoctorInfo.fromJson(Json json) {
    return DoctorInfo(
      id: asString(json['doctor_id']),
      code: asString(json['code']),
      displayName: asString(json['display_name']),
      specialty: asString(json['specialty']),
    );
  }
}

class JournalEntryData {
  const JournalEntryData({
    required this.id,
    required this.text,
    required this.reflectionPrompt,
    required this.createdAt,
  });

  final String id;
  final String text;
  final String reflectionPrompt;
  final DateTime? createdAt;

  factory JournalEntryData.fromJson(Json json) {
    return JournalEntryData(
      id: asString(json['entry_id']),
      text: asString(json['text']),
      reflectionPrompt: asString(json['reflection_prompt']),
      createdAt: asDate(json['created_at']),
    );
  }
}

class MoodCheckInData {
  const MoodCheckInData({
    required this.id,
    required this.answers,
    required this.comment,
    required this.createdAt,
  });

  final String id;
  final List<MapEntry<String, String>> answers;
  final String comment;
  final DateTime? createdAt;

  factory MoodCheckInData.fromJson(Json json) {
    final answers = asJsonList(json['answers'])
        .map(
          (answer) =>
              MapEntry(asString(answer['question']), asString(answer['value'])),
        )
        .where((answer) => answer.key.isNotEmpty)
        .toList(growable: false);
    return MoodCheckInData(
      id: asString(json['check_in_id']),
      answers: answers,
      comment: asString(json['comment']),
      createdAt: asDate(json['created_at']),
    );
  }
}

class AssignmentData {
  const AssignmentData({
    required this.id,
    required this.patientId,
    required this.type,
    required this.title,
    required this.status,
    required this.startsAt,
    required this.endsAt,
    required this.frequency,
    required this.timesOfDay,
    required this.details,
    required this.createdAt,
  });

  final String id;
  final String patientId;
  final String type;
  final String title;
  final String status;
  final DateTime? startsAt;
  final DateTime? endsAt;
  final String frequency;
  final List<String> timesOfDay;
  final Json details;
  final DateTime? createdAt;

  bool get isActive => status == 'ACTIVE';

  String get typeLabel => switch (type) {
    'MEDICATION' => 'Лекарство',
    'QUESTIONNAIRE' => 'Опросник',
    'APP_CHECK_IN' => 'Вход в приложение',
    'JOURNAL_TASK' => 'Задание дневника',
    _ => 'Назначение',
  };

  String get statusLabel => switch (status) {
    'ACTIVE' => 'Активно',
    'COMPLETED' => 'Завершено',
    'PAUSED' => 'На паузе',
    'EXPIRED' => 'Просрочено',
    'CANCELLED' => 'Отменено',
    _ => status,
  };

  String get instructions {
    final value = asString(details['instructions']);
    if (value.isNotEmpty) {
      return value;
    }
    return asString(details['prompt']);
  }

  String get dosageLabel {
    final dosage = asJson(details['dosage']);
    if (dosage.isEmpty) {
      return '';
    }
    final amount = dosage['amount'];
    final unit = asString(dosage['unit']);
    final text = amount is num
        ? '${amount.toString().replaceFirst(RegExp(r'\.0$'), '')} $unit'
        : unit;
    return text.trim();
  }

  String get scheduleLabel {
    if (timesOfDay.isEmpty) {
      return frequency == 'DAILY' ? 'Ежедневно' : frequency;
    }
    final frequencyText = frequency == 'DAILY' ? 'Ежедневно' : 'По расписанию';
    return '$frequencyText, ${timesOfDay.join(', ')}';
  }

  factory AssignmentData.fromJson(Json json) {
    final period = asJson(json['period']);
    final schedule = asJson(json['schedule']);
    final rawTimes = schedule['times_of_day'];
    final timesOfDay = rawTimes is List
        ? rawTimes.map((value) => '$value').toList(growable: false)
        : const <String>[];

    return AssignmentData(
      id: asString(json['assignment_id']),
      patientId: asString(json['patient_id']),
      type: asString(json['type']),
      title: asString(json['title'], 'Назначение'),
      status: asString(json['status']),
      startsAt: asDate(period['starts_at']),
      endsAt: asDate(period['ends_at']),
      frequency: asString(schedule['frequency']),
      timesOfDay: timesOfDay,
      details: asJson(json['details']),
      createdAt: asDate(json['created_at']),
    );
  }
}

class PatientDetailsData {
  const PatientDetailsData({
    required this.patient,
    required this.assignments,
    required this.journalEntries,
    required this.moodCheckIns,
  });

  final PatientSummary patient;
  final List<AssignmentData> assignments;
  final List<JournalEntryData> journalEntries;
  final List<MoodCheckInData> moodCheckIns;

  factory PatientDetailsData.fromJson(Json json) {
    return PatientDetailsData(
      patient: PatientSummary.fromJson(asJson(json['patient'])),
      assignments: asJsonList(json['assignments'])
          .map(AssignmentData.fromJson)
          .toList(growable: false),
      journalEntries: asJsonList(json['journal_entries'])
          .map(JournalEntryData.fromJson)
          .toList(growable: false),
      moodCheckIns: asJsonList(json['mood_check_ins'])
          .map(MoodCheckInData.fromJson)
          .toList(growable: false),
    );
  }
}

class MedicationData {
  const MedicationData({
    required this.id,
    required this.name,
    required this.instruction,
  });

  final String id;
  final String name;
  final String instruction;

  factory MedicationData.fromJson(Json json) {
    return MedicationData(
      id: asString(json['medication_id']),
      name: asString(json['name']),
      instruction: asString(json['instruction_text']),
    );
  }
}

class MedicationGroupData {
  const MedicationGroupData({
    required this.id,
    required this.name,
    required this.description,
    required this.medications,
  });

  final String id;
  final String name;
  final String description;
  final List<MedicationData> medications;

  factory MedicationGroupData.fromJson(Json json) {
    return MedicationGroupData(
      id: asString(json['group_id']),
      name: asString(json['name']),
      description: asString(json['description']),
      medications: asJsonList(json['medications'])
          .map(MedicationData.fromJson)
          .toList(growable: false),
    );
  }
}

class NotificationData {
  const NotificationData({
    required this.id,
    required this.patientId,
    required this.message,
    required this.type,
    required this.createdAt,
    required this.readAt,
  });

  final String id;
  final String patientId;
  final String message;
  final String type;
  final DateTime? createdAt;
  final DateTime? readAt;

  bool get isUnread => readAt == null;

  factory NotificationData.fromJson(Json json) {
    return NotificationData(
      id: asString(json['notification_id']),
      patientId: asString(json['patient_id']),
      message: asString(json['message']),
      type: asString(json['type']),
      createdAt: asDate(json['created_at']),
      readAt: asDate(json['read_at']),
    );
  }
}

class ForestActivityData {
  const ForestActivityData({
    required this.type,
    required this.occurredAt,
    required this.points,
  });

  final String type;
  final DateTime? occurredAt;
  final int points;

  String get label => switch (type) {
    'JOURNAL_ENTRY' => 'Запись в дневнике',
    'ASSIGNMENT_COMPLETED' => 'Выполненное назначение',
    _ => 'Активность',
  };

  factory ForestActivityData.fromJson(Json json) {
    return ForestActivityData(
      type: asString(json['activity_type']),
      occurredAt: asDate(json['occurred_at']),
      points: asInt(json['points']),
    );
  }
}

class ForestData {
  const ForestData({required this.growthLevel, required this.activities});

  final int growthLevel;
  final List<ForestActivityData> activities;

  factory ForestData.fromJson(Json json) {
    final forest = asJson(json['forest']);
    return ForestData(
      growthLevel: asInt(forest['growth_level']),
      activities: asJsonList(forest['activities'])
          .map(ForestActivityData.fromJson)
          .toList(growable: false),
    );
  }
}

class SupportMessageData {
  const SupportMessageData({
    required this.id,
    required this.trigger,
    required this.message,
    required this.type,
  });

  final String id;
  final String trigger;
  final String message;
  final String type;

  factory SupportMessageData.fromJson(Json json) {
    return SupportMessageData(
      id: asString(json['content_id']),
      trigger: asString(json['trigger']),
      message: asString(json['message']),
      type: asString(json['content_type']),
    );
  }
}
