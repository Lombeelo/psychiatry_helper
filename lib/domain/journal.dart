import 'common.dart';

class JournalEntry {
  const JournalEntry({
    required this.entryId,
    required this.patientId,
    required this.text,
    required this.createdAt,
    required this.updatedAt,
    this.reflectionPrompt,
  });

  final Id entryId;
  final Id patientId;
  final String text;
  final String? reflectionPrompt;
  final DateTime createdAt;
  final DateTime updatedAt;
}

class MoodAnswer {
  const MoodAnswer({
    required this.question,
    required this.value,
  });

  final String question;
  final String value;
}

class MoodCheckIn {
  const MoodCheckIn({
    required this.checkInId,
    required this.patientId,
    required this.answers,
    required this.createdAt,
    this.comment,
  });

  final Id checkInId;
  final Id patientId;
  final List<MoodAnswer> answers;
  final String? comment;
  final DateTime createdAt;
}
