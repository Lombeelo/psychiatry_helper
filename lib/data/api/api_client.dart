import 'dart:convert';

import 'package:http/http.dart' as http;

import 'api_requests.dart';

typedef JsonMap = Map<String, dynamic>;

class APIException implements Exception {
  const APIException({required this.statusCode, required this.body});

  final int statusCode;
  final String body;

  @override
  String toString() => 'APIException($statusCode): $body';
}

class ApiClient {
  ApiClient({required String baseUrl, http.Client? client})
    : _baseUri = Uri.parse(baseUrl),
      _client = client ?? http.Client();

  final Uri _baseUri;
  final http.Client _client;

  /// Пока сервер использует DevAuthenticator, каждый запрос должен
  /// сообщать, от чьего имени он выполняется.
  String? _devUserId;
  String? _devRole;

  void setDevIdentity({String? userId, String? role}) {
    _devUserId = userId?.trim();
    _devRole = role?.trim();
  }

  void close() => _client.close();

  Future<JsonMap> healthz() => _request('GET', '/healthz');

  Future<JsonMap> createPatient(CreatePatientRequest request) =>
      _request('POST', '/v1/patients', body: request.toJson());

  Future<JsonMap> createDoctor(CreateDoctorRequest request) =>
      _request('POST', '/v1/doctors', body: request.toJson());

  Future<JsonMap> createRelationship(CreateRelationshipRequest request) =>
      _request('POST', '/v1/relationships', body: request.toJson());

  Future<JsonMap> endRelationship(String relationshipId) =>
      _request('POST', '/v1/relationships/${_segment(relationshipId)}/end');

  Future<List<JsonMap>> listAssignments({String? patientId}) async {
    final query = <String, dynamic>{};
    if (patientId != null) {
      query['patient_id'] = patientId;
    }

    final response = await _request('GET', '/v1/assignments', query: query);

    // Предполагается, что сервер возвращает либо JSON-массив,
    // либо объект вида {"assignments": [...]}.
    final dynamic items = response['assignments'] ?? response['data'];

    if (items is List) {
      return items.map(_asJsonMap).toList(growable: false);
    }

    throw const FormatException(
      'Ожидался массив назначений в поле "assignments" или "data".',
    );
  }

  Future<JsonMap> createAssignment(CreateAssignmentRequest request) =>
      _request('POST', '/v1/assignments', body: request.toJson());

  Future<JsonMap> pauseAssignment(String assignmentId) =>
      _request('POST', '/v1/assignments/${_segment(assignmentId)}/pause');

  Future<JsonMap> resumeAssignment(String assignmentId) =>
      _request('POST', '/v1/assignments/${_segment(assignmentId)}/resume');

  Future<JsonMap> closeAssignment(String assignmentId) =>
      _request('POST', '/v1/assignments/${_segment(assignmentId)}/close');

  Future<JsonMap> cancelAssignment(String assignmentId) =>
      _request('POST', '/v1/assignments/${_segment(assignmentId)}/cancel');

  Future<JsonMap> completeAssignment(
    String assignmentId,
    CompleteAssignmentRequest request,
  ) => _request(
    'POST',
    '/v1/assignments/${_segment(assignmentId)}/complete',
    body: request.toJson(),
  );

  Future<JsonMap> createJournalEntry(CreateJournalEntryRequest request) =>
      _request('POST', '/v1/journal-entries', body: request.toJson());

  Future<JsonMap> createMoodCheckIn(CreateMoodCheckInRequest request) =>
      _request('POST', '/v1/mood-check-ins', body: request.toJson());

  Future<JsonMap> _request(
    String method,
    String path, {
    JsonMap? query,
    Object? body,
  }) async {
    final uri = _baseUri.resolve(path).replace(queryParameters: query);

    final headers = <String, String>{
      'Accept': 'application/json',
      if (body != null) 'Content-Type': 'application/json',
      if ((_devUserId ?? '').isNotEmpty && (_devRole ?? '').isNotEmpty) ...{
        'X-Dev-User-ID': _devUserId!,
        'X-Dev-Role': _devRole!,
      },
    };

    final encodedBody = body == null ? null : jsonEncode(body);

    final http.Response response;
    switch (method) {
      case 'GET':
        response = await _client.get(uri, headers: headers);
      case 'POST':
        response = await _client.post(uri, headers: headers, body: encodedBody);
      default:
        throw ArgumentError.value(method, 'method', 'Не поддерживается');
    }

    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw APIException(statusCode: response.statusCode, body: response.body);
    }

    if (response.body.isEmpty) {
      return <String, dynamic>{};
    }

    final dynamic decoded = jsonDecode(response.body);

    if (decoded is Map<String, dynamic>) {
      return decoded;
    }

    // Позволяет обработать обёртку-массив в listAssignments.
    if (decoded is List) {
      return <String, dynamic>{'data': decoded};
    }

    throw FormatException(
      'Ожидался JSON-объект или массив, получено: ${decoded.runtimeType}',
    );
  }

  static String _segment(String value) => Uri.encodeComponent(value);

  static JsonMap _asJsonMap(dynamic value) {
    if (value is Map<String, dynamic>) {
      return value;
    }
    if (value is Map) {
      return value.map((key, value) => MapEntry(key.toString(), value));
    }
    throw FormatException('Ожидался JSON-объект, получено: $value');
  }
}
