import 'common.dart';

enum AnswerType { singleChoice, multipleChoice, scale, freeText }

enum ResultCategory { low, moderate, high, other }

class Question {
  const Question({
    required this.questionId,
    required this.text,
    required this.answerType,
    required this.options,
    required this.scoreMapping,
  });

  final Id questionId;
  final String text;
  final AnswerType answerType;
  final List<String> options;
  final Map<String, double> scoreMapping;
}

class QuestionnaireVersion {
  const QuestionnaireVersion({
    required this.questionnaireVersionId,
    required this.version,
    required this.questions,
    required this.lowScoreThreshold,
    required this.createdAt,
    required this.isActive,
  });

  final Id questionnaireVersionId;
  final String version;
  final List<Question> questions;
  final double lowScoreThreshold;
  final DateTime createdAt;
  final bool isActive;
}

class Answer {
  const Answer({
    required this.questionId,
    required this.value,
    this.score,
  });

  final Id questionId;
  final String value;
  final double? score;
}

class QuestionnaireAttempt {
  const QuestionnaireAttempt({
    required this.attemptId,
    required this.questionnaireId,
    required this.questionnaireVersionId,
    required this.patientId,
    required this.answers,
    required this.score,
    required this.resultCategory,
    required this.completedAt,
    this.assignmentId,
  });

  final Id attemptId;
  final Id questionnaireId;
  final Id questionnaireVersionId;
  final Id patientId;
  final Id? assignmentId;
  final List<Answer> answers;
  final double score;
  final ResultCategory resultCategory;
  final DateTime completedAt;
}
