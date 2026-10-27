import 'dart:convert';

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
        'display_name': _dateOnly(dateOfBirth),
        'date_of_birth': _dateOnly(dateOfBirth),
        'policy_version': policyVersion,
        'accepted': accepted,
      };

  static String _dateOnly(DateTime value) =>
      '${value.year.toString().padLeft(4, '0')}-'
      '${value.month.toString().padLeft(2, '0')}-'
      '${value.day.toString().padLeft(2, '0')}';
}

class CreateRelationshipRequest {
  const CreateRelationshipRequest({required this.patientId});

  final String patientId;

  JsonMap toJson() => {'patient_id': patientId};
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

  /// Например: {"instructions":"..."}.
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
        'details': jsonEncode(details),
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
  });

  final String comment;
  final String result;

  JsonMap toJson() => {
        'comment': comment,
        'result': result,
      };
}

class CreateJournalEntryRequest {
  const CreateJournalEntryRequest({
    required this.text,
    this.reflectionPrompt = '',
  });

  final String text;
  final String reflectionPrompt;

  JsonMap toJson() => {
        'text': text,
        'reflection_prompt': reflectionPrompt,
      };
}

class CreateMoodCheckInRequest {
  const CreateMoodCheckInRequest({
    required this.answers,
    this.comment = '',
  });

  /// Каждый элемент должен соответствовать domain.MoodAnswer в Go.
  final List<JsonMap> answers;
  final String comment;

  JsonMap toJson() => {
        'answers': answers,
        'comment': comment,
      };
}
