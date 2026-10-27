import '../domain/accounts.dart';
import '../domain/assignments.dart';
import '../domain/common.dart';

abstract interface class AssignmentRepository {
  Future<List<Assignment>> forPatient(Id patientId);
  Future<Assignment> create(Assignment assignment);
  Future<Assignment> updateStatus(Id assignmentId, AssignmentStatus status);
}

abstract interface class CareRelationshipRepository {
  Future<CareRelationship> connectWithCode(UserCode code);
  Future<List<CareRelationship>> activeForDoctor(Id doctorId);
}

abstract interface class PatientRepository {
  Future<Patient?> getById(Id patientId);
}
