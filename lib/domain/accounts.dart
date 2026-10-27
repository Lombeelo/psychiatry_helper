import 'common.dart';

class Patient {
  const Patient({
    required this.accountId,
    required this.code,
    required this.personalData,
    required this.createdAt,
    required this.notes,
    required this.consent,
  });

  final Id accountId;
  final UserCode code;
  final PersonalData personalData;
  final DateTime createdAt;
  final String notes;
  final Consent consent;
}

class Doctor {
  const Doctor({
    required this.accountId,
    required this.code,
    required this.personalData,
    required this.createdAt,
    required this.specialty,
  });

  final Id accountId;
  final UserCode code;
  final PersonalData personalData;
  final DateTime createdAt;
  final String specialty;
}

class CareRelationship {
  const CareRelationship({
    required this.relationshipId,
    required this.patientId,
    required this.doctorId,
    required this.createdAt,
    required this.status,
    this.endedAt,
  });

  final Id relationshipId;
  final Id patientId;
  final Id doctorId;
  final DateTime createdAt;
  final RelationshipStatus status;
  final DateTime? endedAt;

  CareRelationship end(DateTime at) {
    if (status == RelationshipStatus.ended) {
      return this;
    }

    return CareRelationship(
      relationshipId: relationshipId,
      patientId: patientId,
      doctorId: doctorId,
      createdAt: createdAt,
      status: RelationshipStatus.ended,
      endedAt: at,
    );
  }
}
