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

/// Правило подсчёта конкретной версии опросника.
/// Порог низкого результата хранится в версии, потому что результаты
/// разных опросников не сравниваются по общей шкале.
class ScoringRule {
  const ScoringRule({required this.description});

  final String description;

  double calculate(Iterable<Answer> answers) =>
      answers.fold<double>(0, (total, answer) => total + answer.score);

  ResultCategory classify(double score, {required double lowScoreThreshold}) {
    if (score <= lowScoreThreshold) {
      return ResultCategory.low;
    }
    return ResultCategory.other;
  }
}

class QuestionnaireVersion {
  const QuestionnaireVersion({
    required this.questionnaireVersionId,
    required this.version,
    required this.scoringRule,
    required this.questions,
    required this.lowScoreThreshold,
    required this.createdAt,
    required this.isActive,
  });

  final Id questionnaireVersionId;
  final String version;
  final ScoringRule scoringRule;
  final List<Question> questions;
  final double lowScoreThreshold;
  final DateTime createdAt;
  final bool isActive;
}

/// Опросник и его версии. Завершённая попытка всегда ссылается
/// на конкретную версию.
class Questionnaire {
  const Questionnaire({
    required this.questionnaireId,
    required this.name,
    required this.isActive,
    required this.versions,
  });

  final Id questionnaireId;
  final String name;
  final bool isActive;
  final List<QuestionnaireVersion> versions;
}

class Answer {
  const Answer({
    required this.questionId,
    required this.value,
    required this.score,
  });

  final Id questionId;
  final String value;
  final double score;
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

  /// Проверяет, что попытка заполнена по указанной версии опросника.
  void validateAgainst(QuestionnaireVersion version) {
    if (questionnaireVersionId != version.questionnaireVersionId) {
      throw StateError('Версия попытки не совпадает с версией опросника');
    }
    if (answers.length != version.questions.length) {
      throw StateError('Опросник заполнен не полностью');
    }

    final expected = version.questions
        .map((question) => question.questionId)
        .toSet();
    for (final answer in answers) {
      if (!expected.remove(answer.questionId)) {
        throw StateError('Ответ на неизвестный вопрос: ${answer.questionId}');
      }
    }
    if (expected.isNotEmpty) {
      throw StateError('Не на все вопросы дан ответ');
    }
  }
}
