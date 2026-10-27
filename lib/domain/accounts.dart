import 'common.dart';

/// Общая часть учётной записи. Patient, Doctor и AdminStub различаются
/// ролью и профилем.
abstract class Account {
  const Account({
    required this.accountId,
    required this.code,
    required this.personalData,
    required this.createdAt,
  });

  final Id accountId;
  final UserCode code;
  final PersonalData personalData;
  final DateTime createdAt;
}

class PatientProfile {
  const PatientProfile({this.notes = ''});

  final String notes;
}

class DoctorProfile {
  const DoctorProfile({this.specialty = ''});

  final String specialty;
}

class Patient extends Account {
  const Patient({
    required super.accountId,
    required super.code,
    required super.personalData,
    required super.createdAt,
    required this.profile,
    required this.consent,
  });

  final PatientProfile profile;
  final Consent consent;
}

class Doctor extends Account {
  const Doctor({
    required super.accountId,
    required super.code,
    required super.personalData,
    required super.createdAt,
    required this.profile,
  });

  final DoctorProfile profile;
}

/// Заглушка роли администратора. Административные функции в MVP
/// не реализуются, поэтому доступен только сам факт наличия роли.
class AdminStub extends Account {
  const AdminStub({
    required super.accountId,
    required super.code,
    required super.personalData,
    required super.createdAt,
    this.available = false,
  });

  final bool available;
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

  bool get isActive => status == RelationshipStatus.active;

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
