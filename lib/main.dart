import 'dart:convert';

import 'package:flutter/material.dart';

import 'data/api/api_client.dart';
import 'data/api/api_requests.dart';


typedef JsonMap = Map<String, dynamic>;

void main() {
  runApp(
    CareApp(
      api: ApiClient(baseUrl: const String.fromEnvironment(
        'API_BASE_URL',
        defaultValue: 'http://localhost:8080',
      )),
    ),
  );
}

enum AppRole { doctor, patient }

class CareApp extends StatelessWidget {
  const CareApp({super.key, required this.api});

  final ApiClient api;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Сопровождение пациента',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xFF39796B),
        ),
        useMaterial3: true,
        scaffoldBackgroundColor: const Color(0xFFF6F8F7),
        inputDecorationTheme: const InputDecorationTheme(
          border: OutlineInputBorder(),
        ),
      ),
      home: AppShell(api: api),
    );
  }
}

class AppShell extends StatefulWidget {
  const AppShell({super.key, required this.api});

  final ApiClient api;

  @override
  State<AppShell> createState() => _AppShellState();
}

class _AppShellState extends State<AppShell> {
  AppRole _role = AppRole.doctor;
  int _selectedIndex = 0;

  @override
  Widget build(BuildContext context) {
    final destinations = _role == AppRole.doctor
        ? const [
            NavigationRailDestination(
              icon: Icon(Icons.people_outline),
              selectedIcon: Icon(Icons.people),
              label: Text('Пациенты'),
            ),
            NavigationRailDestination(
              icon: Icon(Icons.assignment_outlined),
              selectedIcon: Icon(Icons.assignment),
              label: Text('Назначения'),
            ),
          ]
        : const [
            NavigationRailDestination(
              icon: Icon(Icons.today_outlined),
              selectedIcon: Icon(Icons.today),
              label: Text('Сегодня'),
            ),
            NavigationRailDestination(
              icon: Icon(Icons.menu_book_outlined),
              selectedIcon: Icon(Icons.menu_book),
              label: Text('Дневник'),
            ),
          ];

    final page = _role == AppRole.doctor
        ? (_selectedIndex == 0
            ? DoctorPatientsPage(api: widget.api)
            : DoctorAssignmentsPage(api: widget.api))
        : (_selectedIndex == 0
            ? PatientHomePage(api: widget.api)
            : PatientJournalPage(api: widget.api));

    return Scaffold(
      body: Row(
        children: [
          NavigationRail(
            extended: true,
            selectedIndex: _selectedIndex,
            onDestinationSelected: (index) {
              setState(() => _selectedIndex = index);
            },
            leading: Padding(
              padding: const EdgeInsets.fromLTRB(12, 20, 12, 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Icon(Icons.health_and_safety, size: 32),
                  const SizedBox(height: 12),
                  SegmentedButton<AppRole>(
                    segments: const [
                      ButtonSegment(
                        value: AppRole.doctor,
                        label: Text('Врач'),
                        icon: Icon(Icons.medical_services_outlined),
                      ),
                      ButtonSegment(
                        value: AppRole.patient,
                        label: Text('Пациент'),
                        icon: Icon(Icons.person_outline),
                      ),
                    ],
                    selected: {_role},
                    onSelectionChanged: (selection) {
                      setState(() {
                        _role = selection.first;
                        _selectedIndex = 0;
                      });
                    },
                  ),
                ],
              ),
            ),
            destinations: destinations,
          ),
          const VerticalDivider(width: 1),
          Expanded(child: page),
        ],
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Врач: пациенты
// -----------------------------------------------------------------------------

class DoctorPatientsPage extends StatefulWidget {
  const DoctorPatientsPage({super.key, required this.api});

  final ApiClient api;

  @override
  State<DoctorPatientsPage> createState() => _DoctorPatientsPageState();
}

class _DoctorPatientsPageState extends State<DoctorPatientsPage> {
  final _patientIdController = TextEditingController();

  @override
  void dispose() {
    _patientIdController.dispose();
    super.dispose();
  }

  Future<void> _createRelationship() async {
    final patientId = _patientIdController.text.trim();
    if (patientId.isEmpty) {
      _showMessage(context, 'Введите ID пациента');
      return;
    }

    try {
      final result = await widget.api.createRelationship(
        CreateRelationshipRequest(patientId: patientId),
      );
      if (!mounted) return;
      _showJsonDialog(context, 'Связь создана', result);
    } catch (error) {
      if (mounted) _showMessage(context, 'Не удалось создать связь: $error');
    }
  }

  Future<void> _createPatient() async {
    final request = await showDialog<CreatePatientRequest>(
      context: context,
      builder: (_) => const CreatePatientDialog(),
    );
    if (request == null) return;

    try {
      final result = await widget.api.createPatient(request);
      if (!mounted) return;
      _showJsonDialog(context, 'Пациент создан', result);
    } catch (error) {
      if (mounted) _showMessage(context, 'Не удалось создать пациента: $error');
    }
  }

  @override
  Widget build(BuildContext context) {
    return PageFrame(
      title: 'Пациенты',
      subtitle: 'Управление связями и создание учётных записей',
      actions: [
        FilledButton.icon(
          onPressed: _createPatient,
          icon: const Icon(Icons.person_add_alt_1),
          label: const Text('Создать пациента'),
        ),
      ],
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 760),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const InfoCard(
              icon: Icons.info_outline,
              text: 'В текущем API нет метода получения списка пациентов. '
                  'Для создания связи нужен ID пациента.',
            ),
            const SizedBox(height: 20),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(20),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Связать с пациентом',
                      style: Theme.of(context).textTheme.titleLarge,
                    ),
                    const SizedBox(height: 16),
                    TextField(
                      controller: _patientIdController,
                      decoration: const InputDecoration(
                        labelText: 'ID пациента',
                        hintText: 'UUID',
                      ),
                    ),
                    const SizedBox(height: 12),
                    FilledButton(
                      onPressed: _createRelationship,
                      child: const Text('Создать связь'),
                    ),
                  ],
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Врач: назначения
// -----------------------------------------------------------------------------

class DoctorAssignmentsPage extends StatefulWidget {
  const DoctorAssignmentsPage({super.key, required this.api});

  final ApiClient api;

  @override
  State<DoctorAssignmentsPage> createState() => _DoctorAssignmentsPageState();
}

class _DoctorAssignmentsPageState extends State<DoctorAssignmentsPage> {
  final _patientIdController = TextEditingController();
  bool _loading = false;
  List<JsonMap> _assignments = const [];

  @override
  void dispose() {
    _patientIdController.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() => _loading = true);
    try {
      final patientId = _patientIdController.text.trim();
      final items = await widget.api.listAssignments(
        patientId: patientId.isEmpty ? null : patientId,
      );
      if (!mounted) return;
      setState(() => _assignments = items);
    } catch (error) {
      if (mounted) _showMessage(context, 'Не удалось загрузить назначения: $error');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _create() async {
    final request = await showDialog<CreateAssignmentRequest>(
      context: context,
      builder: (_) => const CreateAssignmentDialog(),
    );
    if (request == null) return;

    try {
      final result = await widget.api.createAssignment(request);
      if (!mounted) return;
      _showJsonDialog(context, 'Назначение создано', result);
      await _load();
    } catch (error) {
      if (mounted) _showMessage(context, 'Не удалось создать назначение: $error');
    }
  }

  Future<void> _action(
    String id,
    String action,
    Future<JsonMap> Function(String) call,
  ) async {
    try {
      final result = await call(id);
      if (!mounted) return;
      _showJsonDialog(context, 'Назначение: $action', result);
      await _load();
    } catch (error) {
      if (mounted) _showMessage(context, 'Операция не выполнена: $error');
    }
  }

  @override
  Widget build(BuildContext context) {
    return PageFrame(
      title: 'Назначения',
      subtitle: 'Просмотр и управление назначениями пациентов',
      actions: [
        FilledButton.icon(
          onPressed: _create,
          icon: const Icon(Icons.add),
          label: const Text('Новое назначение'),
        ),
      ],
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              SizedBox(
                width: 420,
                child: TextField(
                  controller: _patientIdController,
                  decoration: const InputDecoration(
                    labelText: 'Фильтр по ID пациента',
                  ),
                  onSubmitted: (_) => _load(),
                ),
              ),
              const SizedBox(width: 12),
              OutlinedButton.icon(
                onPressed: _loading ? null : _load,
                icon: const Icon(Icons.refresh),
                label: const Text('Загрузить'),
              ),
            ],
          ),
          const SizedBox(height: 16),
          if (_loading) const LinearProgressIndicator(),
          Expanded(
            child: _assignments.isEmpty
                ? const Center(child: Text('Назначения не загружены'))
                : ListView.separated(
                    itemCount: _assignments.length,
                    separatorBuilder: (_, __) => const SizedBox(height: 8),
                    itemBuilder: (context, index) {
                      final item = _assignments[index];
                      final id = '${item['assignment_id'] ?? item['id'] ?? ''}';
                      final title = '${item['title'] ?? 'Назначение'}';

                      return Card(
                        child: ListTile(
                          title: Text(title),
                          subtitle: SelectableText(
                            const JsonEncoder.withIndent('  ').convert(item),
                          ),
                          trailing: Wrap(
                            spacing: 8,
                            children: [
                              IconButton(
                                tooltip: 'Пауза',
                                onPressed: id.isEmpty
                                    ? null
                                    : () => _action(
                                          id,
                                          'пауза',
                                          widget.api.pauseAssignment,
                                        ),
                                icon: const Icon(Icons.pause),
                              ),
                              IconButton(
                                tooltip: 'Возобновить',
                                onPressed: id.isEmpty
                                    ? null
                                    : () => _action(
                                          id,
                                          'возобновление',
                                          widget.api.resumeAssignment,
                                        ),
                                icon: const Icon(Icons.play_arrow),
                              ),
                            ],
                          ),
                        ),
                      );
                    },
                  ),
          ),
        ],
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Пациент: сегодня и отметка выполнения
// -----------------------------------------------------------------------------

class PatientHomePage extends StatefulWidget {
  const PatientHomePage({super.key, required this.api});

  final ApiClient api;

  @override
  State<PatientHomePage> createState() => _PatientHomePageState();
}

class _PatientHomePageState extends State<PatientHomePage> {
  final _patientIdController = TextEditingController();
  bool _loading = false;
  List<JsonMap> _assignments = const [];

  @override
  void dispose() {
    _patientIdController.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    final patientId = _patientIdController.text.trim();
    if (patientId.isEmpty) {
      _showMessage(context, 'Введите ID пациента');
      return;
    }

    setState(() => _loading = true);
    try {
      final items = await widget.api.listAssignments(patientId: patientId);
      if (mounted) setState(() => _assignments = items);
    } catch (error) {
      if (mounted) _showMessage(context, 'Не удалось загрузить назначения: $error');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _complete(String assignmentId) async {
    final request = await showDialog<CompleteAssignmentRequest>(
      context: context,
      builder: (_) => const CompleteAssignmentDialog(),
    );
    if (request == null) return;

    try {
      final result = await widget.api.completeAssignment(
        assignmentId,
        request,
      );
      if (!mounted) return;
      _showJsonDialog(context, 'Выполнение отправлено', result);
      await _load();
    } catch (error) {
      if (mounted) _showMessage(context, 'Не удалось отправить выполнение: $error');
    }
  }

  @override
  Widget build(BuildContext context) {
    return PageFrame(
      title: 'Сегодня',
      subtitle: 'Ваши назначения и отметки выполнения',
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const InfoCard(
            icon: Icons.info_outline,
            text: 'Для загрузки данных введите ID пациента. '
                'В дальнейшем его можно получать после входа в аккаунт.',
          ),
          const SizedBox(height: 16),
          Row(
            children: [
              SizedBox(
                width: 420,
                child: TextField(
                  controller: _patientIdController,
                  decoration: const InputDecoration(labelText: 'ID пациента'),
                  onSubmitted: (_) => _load(),
                ),
              ),
              const SizedBox(width: 12),
              FilledButton(
                onPressed: _loading ? null : _load,
                child: const Text('Загрузить назначения'),
              ),
            ],
          ),
          const SizedBox(height: 16),
          if (_loading) const LinearProgressIndicator(),
          Expanded(
            child: _assignments.isEmpty
                ? const Center(child: Text('Назначения не загружены'))
                : ListView.separated(
                    itemCount: _assignments.length,
                    separatorBuilder: (_, __) => const SizedBox(height: 8),
                    itemBuilder: (context, index) {
                      final item = _assignments[index];
                      final id = '${item['assignment_id'] ?? item['id'] ?? ''}';
                      return Card(
                        child: ListTile(
                          title: Text('${item['title'] ?? 'Назначение'}'),
                          subtitle: Text(
                            'Статус: ${item['status'] ?? '—'}',
                          ),
                          trailing: FilledButton.tonal(
                            onPressed: id.isEmpty ? null : () => _complete(id),
                            child: const Text('Отметить выполнение'),
                          ),
                        ),
                      );
                    },
                  ),
          ),
        ],
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Пациент: дневник и состояние
// -----------------------------------------------------------------------------

class PatientJournalPage extends StatelessWidget {
  const PatientJournalPage({super.key, required this.api});

  final ApiClient api;

  Future<void> _createJournalEntry(BuildContext context) async {
    final request = await showDialog<CreateJournalEntryRequest>(
      context: context,
      builder: (_) => const JournalEntryDialog(),
    );
    if (request == null) return;

    try {
      final result = await api.createJournalEntry(request);
      if (context.mounted) {
        _showJsonDialog(context, 'Запись сохранена', result);
      }
    } catch (error) {
      if (context.mounted) {
        _showMessage(context, 'Не удалось сохранить запись: $error');
      }
    }
  }

  Future<void> _createMoodCheckIn(BuildContext context) async {
    final request = await showDialog<CreateMoodCheckInRequest>(
      context: context,
      builder: (_) => const MoodCheckInDialog(),
    );
    if (request == null) return;

    try {
      final result = await api.createMoodCheckIn(request);
      if (context.mounted) {
        _showJsonDialog(context, 'Отметка сохранена', result);
      }
    } catch (error) {
      if (context.mounted) {
        _showMessage(context, 'Не удалось сохранить отметку: $error');
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return PageFrame(
      title: 'Дневник',
      subtitle: 'Записи и короткие отметки состояния',
      child: Wrap(
        spacing: 16,
        runSpacing: 16,
        children: [
          ActionCard(
            icon: Icons.edit_note,
            title: 'Запись в дневник',
            description: 'Добавить текстовую запись',
            onPressed: () => _createJournalEntry(context),
          ),
          ActionCard(
            icon: Icons.mood_outlined,
            title: 'Отметить состояние',
            description: 'Ответить на короткие вопросы',
            onPressed: () => _createMoodCheckIn(context),
          ),
        ],
      ),
    );
  }
}

// -----------------------------------------------------------------------------
// Диалоги
// -----------------------------------------------------------------------------

class CreatePatientDialog extends StatefulWidget {
  const CreatePatientDialog({super.key});

  @override
  State<CreatePatientDialog> createState() => _CreatePatientDialogState();
}

class _CreatePatientDialogState extends State<CreatePatientDialog> {
  final _name = TextEditingController();
  final _birthDate = TextEditingController();
  final _policy = TextEditingController(text: 'v1');
  bool _accepted = false;

  @override
  void dispose() {
    _name.dispose();
    _birthDate.dispose();
    _policy.dispose();
    super.dispose();
  }

  void _submit() {
    final date = DateTime.tryParse(_birthDate.text.trim());
    if (_name.text.trim().isEmpty || date == null || !_accepted) return;

    Navigator.pop(
      context,
      CreatePatientRequest(
        displayName: _name.text.trim(),
        dateOfBirth: date,
        policyVersion: _policy.text.trim(),
        accepted: _accepted,
      ),
    );
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
        title: const Text('Создать пациента'),
        content: SizedBox(
          width: 420,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: _name,
                decoration: const InputDecoration(labelText: 'Имя'),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _birthDate,
                decoration: const InputDecoration(
                  labelText: 'Дата рождения',
                  hintText: 'ГГГГ-ММ-ДД',
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _policy,
                decoration: const InputDecoration(
                  labelText: 'Версия политики согласия',
                ),
              ),
              CheckboxListTile(
                contentPadding: EdgeInsets.zero,
                value: _accepted,
                onChanged: (value) =>
                    setState(() => _accepted = value ?? false),
                title: const Text('Согласие получено'),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Отмена'),
          ),
          FilledButton(
            onPressed: _submit,
            child: const Text('Создать'),
          ),
        ],
      );
}

class CreateAssignmentDialog extends StatefulWidget {
  const CreateAssignmentDialog({super.key});

  @override
  State<CreateAssignmentDialog> createState() =>
      _CreateAssignmentDialogState();
}

class _CreateAssignmentDialogState extends State<CreateAssignmentDialog> {
  final _patientId = TextEditingController();
  final _title = TextEditingController();
  final _start = TextEditingController(text: '2026-10-27');
  final _end = TextEditingController(text: '2026-11-03');
  final _times = TextEditingController(text: '09:00');
  final _details = TextEditingController(text: '{}');

  String _type = 'APP_CHECK_IN';
  String _frequency = 'DAILY';

  @override
  void dispose() {
    _patientId.dispose();
    _title.dispose();
    _start.dispose();
    _end.dispose();
    _times.dispose();
    _details.dispose();
    super.dispose();
  }

  void _submit() {
    final startDate = DateTime.tryParse(_start.text.trim());
    final endDate = DateTime.tryParse(_end.text.trim());
    final patientId = _patientId.text.trim();
    final title = _title.text.trim();

    if (patientId.isEmpty || title.isEmpty || startDate == null ||
        endDate == null) {
      return;
    }

    JsonMap details;
    try {
      final decoded = jsonDecode(_details.text);
      if (decoded is! Map) return;
      details = decoded.map((key, value) => MapEntry(key.toString(), value));
    } catch (_) {
      return;
    }

    Navigator.pop(
      context,
      CreateAssignmentRequest(
        patientId: patientId,
        title: title,
        startsAt: startDate,
        endsAt: endDate,
        frequency: _frequency,
        daysOfWeek: const [1, 2, 3, 4, 5, 6, 7],
        timesOfDay: _times.text
            .split(',')
            .map((time) => time.trim())
            .where((time) => time.isNotEmpty)
            .toList(),
        type: _type,
        details: details,
      ),
    );
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
        title: const Text('Новое назначение'),
        content: SizedBox(
          width: 500,
          child: SingleChildScrollView(
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                TextField(
                  controller: _patientId,
                  decoration: const InputDecoration(labelText: 'ID пациента'),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _title,
                  decoration: const InputDecoration(labelText: 'Название'),
                ),
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  value: _type,
                  decoration: const InputDecoration(labelText: 'Тип'),
                  items: const [
                    DropdownMenuItem(
                      value: 'MEDICATION',
                      child: Text('Лекарство'),
                    ),
                    DropdownMenuItem(
                      value: 'QUESTIONNAIRE',
                      child: Text('Опросник'),
                    ),
                    DropdownMenuItem(
                      value: 'APP_CHECK_IN',
                      child: Text('Отметка в приложении'),
                    ),
                    DropdownMenuItem(
                      value: 'JOURNAL_TASK',
                      child: Text('Задание дневника'),
                    ),
                  ],
                  onChanged: (value) {
                    if (value != null) setState(() => _type = value);
                  },
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _start,
                  decoration: const InputDecoration(
                    labelText: 'Начало',
                    hintText: 'ГГГГ-ММ-ДД',
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _end,
                  decoration: const InputDecoration(
                    labelText: 'Окончание',
                    hintText: 'ГГГГ-ММ-ДД',
                  ),
                ),
                const SizedBox(height: 12),
                DropdownButtonFormField<String>(
                  value: _frequency,
                  decoration: const InputDecoration(labelText: 'Частота'),
                  items: const [
                    DropdownMenuItem(value: 'DAILY', child: Text('Ежедневно')),
                    DropdownMenuItem(value: 'WEEKLY', child: Text('Еженедельно')),
                  ],
                  onChanged: (value) {
                    if (value != null) setState(() => _frequency = value);
                  },
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _times,
                  decoration: const InputDecoration(
                    labelText: 'Время',
                    hintText: '09:00, 18:00',
                  ),
                ),
                const SizedBox(height: 12),
                TextField(
                  controller: _details,
                  minLines: 2,
                  maxLines: 5,
                  decoration: const InputDecoration(
                    labelText: 'Details (JSON)',
                    hintText: '{"instructions":"..."}',
                  ),
                ),
              ],
            ),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Отмена'),
          ),
          FilledButton(
            onPressed: _submit,
            child: const Text('Создать'),
          ),
        ],
      );
}

class CompleteAssignmentDialog extends StatefulWidget {
  const CompleteAssignmentDialog({super.key});

  @override
  State<CompleteAssignmentDialog> createState() =>
      _CompleteAssignmentDialogState();
}

class _CompleteAssignmentDialogState extends State<CompleteAssignmentDialog> {
  final _comment = TextEditingController();
  final _result = TextEditingController();

  @override
  void dispose() {
    _comment.dispose();
    _result.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
        title: const Text('Отметить выполнение'),
        content: SizedBox(
          width: 420,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: _comment,
                decoration: const InputDecoration(labelText: 'Комментарий'),
                minLines: 2,
                maxLines: 4,
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _result,
                decoration: const InputDecoration(labelText: 'Результат'),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Отмена'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(
              context,
              CompleteAssignmentRequest(
                comment: _comment.text.trim(),
                result: _result.text.trim(),
              ),
            ),
            child: const Text('Отправить'),
          ),
        ],
      );
}

class JournalEntryDialog extends StatefulWidget {
  const JournalEntryDialog({super.key});

  @override
  State<JournalEntryDialog> createState() => _JournalEntryDialogState();
}

class _JournalEntryDialogState extends State<JournalEntryDialog> {
  final _text = TextEditingController();
  final _prompt = TextEditingController();

  @override
  void dispose() {
    _text.dispose();
    _prompt.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
        title: const Text('Запись в дневник'),
        content: SizedBox(
          width: 500,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextField(
                controller: _text,
                minLines: 5,
                maxLines: 10,
                decoration: const InputDecoration(
                  labelText: 'Текст записи',
                  alignLabelWithHint: true,
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _prompt,
                decoration: const InputDecoration(
                  labelText: 'Вопрос для размышления (необязательно)',
                ),
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Отмена'),
          ),
          FilledButton(
            onPressed: _text.text.trim().isEmpty
                ? null
                : () => Navigator.pop(
                      context,
                      CreateJournalEntryRequest(
                        text: _text.text.trim(),
                        reflectionPrompt: _prompt.text.trim(),
                      ),
                    ),
            child: const Text('Сохранить'),
          ),
        ],
      );
}

class MoodCheckInDialog extends StatefulWidget {
  const MoodCheckInDialog({super.key});

  @override
  State<MoodCheckInDialog> createState() => _MoodCheckInDialogState();
}

class _MoodCheckInDialogState extends State<MoodCheckInDialog> {
  String _mood = 'Нейтральное';
  String _anxiety = 'Умеренная';
  final _comment = TextEditingController();

  @override
  void dispose() {
    _comment.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) => AlertDialog(
        title: const Text('Отметить состояние'),
        content: SizedBox(
          width: 420,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              DropdownButtonFormField<String>(
                value: _mood,
                decoration: const InputDecoration(labelText: 'Настроение'),
                items: const [
                  DropdownMenuItem(value: 'Хорошее', child: Text('Хорошее')),
                  DropdownMenuItem(
                    value: 'Нейтральное',
                    child: Text('Нейтральное'),
                  ),
                  DropdownMenuItem(value: 'Плохое', child: Text('Плохое')),
                ],
                onChanged: (value) {
                  if (value != null) setState(() => _mood = value);
                },
              ),
              const SizedBox(height: 12),
              DropdownButtonFormField<String>(
                value: _anxiety,
                decoration: const InputDecoration(labelText: 'Тревожность'),
                items: const [
                  DropdownMenuItem(value: 'Низкая', child: Text('Низкая')),
                  DropdownMenuItem(
                    value: 'Умеренная',
                    child: Text('Умеренная'),
                  ),
                  DropdownMenuItem(value: 'Высокая', child: Text('Высокая')),
                ],
                onChanged: (value) {
                  if (value != null) setState(() => _anxiety = value);
                },
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _comment,
                decoration: const InputDecoration(
                  labelText: 'Комментарий (необязательно)',
                ),
                minLines: 2,
                maxLines: 4,
              ),
            ],
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(context),
            child: const Text('Отмена'),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(
              context,
              CreateMoodCheckInRequest(
                answers: [
                  {'question': 'Настроение', 'value': _mood},
                  {'question': 'Тревожность', 'value': _anxiety},
                ],
                comment: _comment.text.trim(),
              ),
            ),
            child: const Text('Сохранить'),
          ),
        ],
      );
}

// -----------------------------------------------------------------------------
// Общие компоненты
// -----------------------------------------------------------------------------

class PageFrame extends StatelessWidget {
  const PageFrame({
    super.key,
    required this.title,
    required this.subtitle,
    required this.child,
    this.actions = const [],
  });

  final String title;
  final String subtitle;
  final Widget child;
  final List<Widget> actions;

  @override
  Widget build(BuildContext context) => Padding(
        padding: const EdgeInsets.all(28),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(title, style: Theme.of(context).textTheme.headlineMedium),
                      const SizedBox(height: 4),
                      Text(subtitle),
                    ],
                  ),
                ),
                ...actions,
              ],
            ),
            const SizedBox(height: 24),
            Expanded(child: child),
          ],
        ),
      );
}

class InfoCard extends StatelessWidget {
  const InfoCard({
    super.key,
    required this.icon,
    required this.text,
  });

  final IconData icon;
  final String text;

  @override
  Widget build(BuildContext context) => Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              Icon(icon),
              const SizedBox(width: 12),
              Expanded(child: Text(text)),
            ],
          ),
        ),
      );
}

class ActionCard extends StatelessWidget {
  const ActionCard({
    super.key,
    required this.icon,
    required this.title,
    required this.description,
    required this.onPressed,
  });

  final IconData icon;
  final String title;
  final String description;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) => SizedBox(
        width: 320,
        child: Card(
          child: Padding(
            padding: const EdgeInsets.all(20),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Icon(icon, size: 32),
                const SizedBox(height: 12),
                Text(title, style: Theme.of(context).textTheme.titleLarge),
                const SizedBox(height: 6),
                Text(description),
                const SizedBox(height: 16),
                FilledButton(
                  onPressed: onPressed,
                  child: const Text('Открыть'),
                ),
              ],
            ),
          ),
        ),
      );
}

void _showMessage(BuildContext context, String message) {
  ScaffoldMessenger.of(context)
    ..hideCurrentSnackBar()
    ..showSnackBar(SnackBar(content: Text(message)));
}

void _showJsonDialog(
  BuildContext context,
  String title,
  JsonMap value,
) {
  showDialog<void>(
    context: context,
    builder: (_) => AlertDialog(
      title: Text(title),
      content: SizedBox(
        width: 560,
        child: SingleChildScrollView(
          child: SelectableText(
            const JsonEncoder.withIndent('  ').convert(value),
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: const Text('Закрыть'),
        ),
      ],
    ),
  );
}
