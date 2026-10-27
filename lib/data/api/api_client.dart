import 'dart:convert';

import 'package:http/http.dart' as http;

import 'api_requests.dart';
import 'models.dart';

class APIException implements Exception {
  const APIException({required this.statusCode, required this.body});

  final int statusCode;
  final String body;

  String get message {
    try {
      final decoded = jsonDecode(body);
      if (decoded is Map && decoded['message'] is String) {
        return decoded['message'] as String;
      }
    } catch (_) {
      // Ниже вернём исходное тело ответа.
    }
    return body.isEmpty ? 'Ошибка сервера ($statusCode)' : body;
  }

  @override
  String toString() => message;
}

class ApiClient {
  ApiClient({required String baseUrl, http.Client? client})
    : _baseUri = Uri.parse(baseUrl),
      _client = client ?? http.Client();

  final Uri _baseUri;
  final http.Client _client;
  String? _token;

  void setToken(String? token) {
    _token = token;
  }

  void close() => _client.close();

  Future<AuthSession> login({
    required String login,
    required String password,
    required String role,
  }) async {
    final response = await _request(
      'POST',
      '/v1/auth/login',
      body: {'login': login, 'password': password, 'role': role},
      authenticated: false,
    );
    return AuthSession.fromLogin(_asMap(response));
  }

  Future<void> acceptConsent({String policyVersion = 'v1'}) async {
    await _request(
      'POST',
      '/v1/consent',
      body: {'policy_version': policyVersion},
    );
  }

  Future<String> saveAnamnesis(String notes) async {
    final response = await _request(
      'PUT',
      '/v1/anamnesis',
      body: {'notes': notes},
    );
    return asString(_asMap(response)['notes']);
  }

  Future<List<PatientSummary>> listPatients() async {
    final response = await _request('GET', '/v1/patients');
    return asJsonList(_asMap(response)['patients'])
        .map(PatientSummary.fromJson)
        .toList(growable: false);
  }

  Future<void> createRelationship(String patientCode) async {
    await _request(
      'POST',
      '/v1/relationships',
      body: {'patient_code': patientCode.trim()},
    );
  }

  Future<PatientDetailsData> getPatientDetails(String patientId) async {
    final response = await _request(
      'GET',
      '/v1/patients/${_segment(patientId)}',
    );
    return PatientDetailsData.fromJson(_asMap(response));
  }

  Future<DoctorInfo> myDoctor() async {
    final response = await _request('GET', '/v1/my-doctor');
    return DoctorInfo.fromJson(_asMap(response));
  }

  Future<List<NotificationData>> listNotifications() async {
    final response = await _request('GET', '/v1/notifications');
    return asJsonList(_asMap(response)['notifications'])
        .map(NotificationData.fromJson)
        .toList(growable: false);
  }

  Future<void> markNotificationRead(String notificationId) async {
    await _request(
      'POST',
      '/v1/notifications/${_segment(notificationId)}/read',
    );
  }

  Future<List<MedicationGroupData>> listMedicationGroups() async {
    final response = await _request('GET', '/v1/medications');
    return asJsonList(_asMap(response)['groups'])
        .map(MedicationGroupData.fromJson)
        .toList(growable: false);
  }

  Future<List<AssignmentData>> listAssignments({String? patientId}) async {
    final query = <String, String>{};
    if (patientId != null && patientId.isNotEmpty) {
      query['patient_id'] = patientId;
    }
    final response = await _request('GET', '/v1/assignments', query: query);
    final items = response is List
        ? asJsonList(response)
        : asJsonList(
            _asMap(response)['assignments'] ?? _asMap(response)['data'],
          );
    return items.map(AssignmentData.fromJson).toList(growable: false);
  }

  Future<AssignmentData> createAssignment(
    CreateAssignmentRequest request,
  ) async {
    final response = await _request(
      'POST',
      '/v1/assignments',
      body: request.toJson(),
    );
    return AssignmentData.fromJson(_asMap(response));
  }

  Future<void> pauseAssignment(String assignmentId) async {
    await _request('POST', '/v1/assignments/${_segment(assignmentId)}/pause');
  }

  Future<void> resumeAssignment(String assignmentId) async {
    await _request('POST', '/v1/assignments/${_segment(assignmentId)}/resume');
  }

  Future<void> closeAssignment(String assignmentId) async {
    await _request('POST', '/v1/assignments/${_segment(assignmentId)}/close');
  }

  Future<void> cancelAssignment(String assignmentId) async {
    await _request('POST', '/v1/assignments/${_segment(assignmentId)}/cancel');
  }

  Future<void> completeAssignment(
    String assignmentId,
    CompleteAssignmentRequest request,
  ) async {
    await _request(
      'POST',
      '/v1/assignments/${_segment(assignmentId)}/complete',
      body: request.toJson(),
    );
  }

  Future<List<JournalEntryData>> listJournalEntries() async {
    final response = await _request('GET', '/v1/journal-entries');
    return asJsonList(_asMap(response)['journal_entries'])
        .map(JournalEntryData.fromJson)
        .toList(growable: false);
  }

  Future<void> createJournalEntry(CreateJournalEntryRequest request) async {
    await _request('POST', '/v1/journal-entries', body: request.toJson());
  }

  Future<void> createMoodCheckIn(CreateMoodCheckInRequest request) async {
    await _request('POST', '/v1/mood-check-ins', body: request.toJson());
  }

  Future<ForestData> forest() async {
    final response = await _request('GET', '/v1/forest');
    return ForestData.fromJson(_asMap(response));
  }

  Future<List<SupportMessageData>> supportMessages() async {
    final response = await _request('GET', '/v1/support/messages');
    return asJsonList(_asMap(response)['messages'])
        .map(SupportMessageData.fromJson)
        .toList(growable: false);
  }

  Future<dynamic> _request(
    String method,
    String path, {
    Map<String, String>? query,
    Object? body,
    bool authenticated = true,
  }) async {
    final uri = _baseUri
        .resolve(path)
        .replace(
          queryParameters: query == null || query.isEmpty ? null : query,
        );
    final headers = <String, String>{
      'Accept': 'application/json',
      if (body != null) 'Content-Type': 'application/json',
      if (authenticated && (_token ?? '').isNotEmpty)
        'Authorization': 'Bearer $_token',
    };

    final encodedBody = body == null ? null : jsonEncode(body);
    final http.Response response;
    switch (method) {
      case 'GET':
        response = await _client.get(uri, headers: headers);
      case 'POST':
        response = await _client.post(uri, headers: headers, body: encodedBody);
      case 'PUT':
        response = await _client.put(uri, headers: headers, body: encodedBody);
      default:
        throw ArgumentError.value(method, 'method', 'Не поддерживается');
    }

    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw APIException(statusCode: response.statusCode, body: response.body);
    }
    if (response.body.trim().isEmpty) {
      return <String, dynamic>{};
    }
    return jsonDecode(response.body);
  }

  static Json _asMap(dynamic value) => asJson(value);

  static String _segment(String value) => Uri.encodeComponent(value);
}
