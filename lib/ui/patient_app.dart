import 'package:flutter/material.dart';

import '../data/api/api_client.dart';
import '../data/api/api_requests.dart';
import '../data/api/models.dart';
import 'widgets.dart';

class PatientApp extends StatefulWidget {
  const PatientApp({
    super.key,
    required this.api,
    required this.session,
    required this.onLogout,
  });

  final ApiClient api;
  final AuthSession session;
  final VoidCallback onLogout;

  @override
  State<PatientApp> createState() => _PatientAppState();
}

class _PatientAppState extends State<PatientApp> {
  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('Мой кабинет'),
            Text(
              '${widget.session.displayName} · ${widget.session.code}',
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ],
        ),
        actions: [
          PopupMenuButton<String>(
            onSelected: (value) {
              if (value == 'logout') {
                widget.api.setToken(null);
                widget.onLogout();
              }
            },
            itemBuilder: (_) => const [
              PopupMenuItem(value: 'logout', child: Text('Выйти')),
            ],
          ),
        ],
      ),
      body: PatientHomeView(api: widget.api, session: widget.session),
    );
  }
}

class PatientHomeView extends StatefulWidget {
  const PatientHomeView({super.key, required this.api, required this.session});

  final ApiClient api;
  final AuthSession session;

  @override
  State<PatientHomeView> createState() => _PatientHomeViewState();
}

class _PatientHomeViewState extends State<PatientHomeView> {
  List<AssignmentData> _assignments = const [];
  DoctorInfo? _doctor;
  ForestData? _forest;
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final assignments = await widget.api.listAssignments();
      final doctor = await _loadDoctor();
      final forest = await _loadForest();
      if (!mounted) return;
      setState(() {
        _assignments = assignments;
        _doctor = doctor;
        _forest = forest;
      });
    } on APIException catch (error) {
      if (mounted) setState(() => _error = error.message);
    } catch (error) {
      if (mounted) {
        setState(() => _error = 'Не удалось загрузить данные: $error');
      }
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<DoctorInfo?> _loadDoctor() async {
    try {
      return await widget.api.myDoctor();
    } on APIException {
      return null;
    }
  }

  Future<ForestData?> _loadForest() async {
    try {
      return await widget.api.forest();
    } catch (_) {
      return null;
    }
  }

  Future<void> _openJournal() async {
    await Navigator.of(context).push<void>(
      MaterialPageRoute(builder: (_) => JournalScreen(api: widget.api)),
    );
    await _load();
  }

  Future<void> _openSupport() async {
    await Navigator.of(context).push<void>(
      MaterialPageRoute(builder: (_) => SupportScreen(api: widget.api)),
    );
  }

  Future<void> _openForest() async {
    await Navigator.of(context).push<void>(
      MaterialPageRoute(builder: (_) => ForestScreen(api: widget.api)),
    );
    await _load();
  }

  Future<void> _openAssignment(AssignmentData assignment) async {
    final changed = await Navigator.of(context).push<bool>(
      MaterialPageRoute(
        builder: (_) =>
            AssignmentScreen(api: widget.api, assignment: assignment),
      ),
    );
    if (changed == true) {
      await _load();
    }
  }

  Future<void> _openMoodCheckIn() async {
    final request = await showDialog<CreateMoodCheckInRequest>(
      context: context,
      builder: (_) => const MoodCheckInDialog(),
    );
    if (request == null) return;
    try {
      await widget.api.createMoodCheckIn(request);
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('Отметка сохранена')));
      await _load();
    } on APIException catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(error.message)));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final activeAssignments = _assignments
        .where((assignment) => assignment.isActive)
        .toList();
    final otherAssignments = _assignments
        .where((assignment) => !assignment.isActive)
        .toList();

    return _loading
        ? const Center(child: CircularProgressIndicator())
        : RefreshIndicator(
            onRefresh: _load,
            child: ListView(
              padding: const EdgeInsets.all(24),
              children: [
                if (_error != null)
                  Card(
                    color: theme.colorScheme.errorContainer,
                    child: ListTile(
                      leading: Icon(
                        Icons.error_outline,
                        color: theme.colorScheme.onErrorContainer,
                      ),
                      title: Text(_error!),
                      trailing: TextButton(
                        onPressed: _load,
                        child: const Text('Повторить'),
                      ),
                    ),
                  ),
                Text(
                  'Здравствуйте, ${widget.session.displayName.split(' ').first}!',
                  style: theme.textTheme.headlineSmall?.copyWith(
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  'Ваш код: ${widget.session.code}. Этот код сообщается врачу '
                  'для привязки.',
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
                const SizedBox(height: 20),
                Wrap(
                  spacing: 14,
                  runSpacing: 14,
                  children: [
                    MenuCard(
                      icon: Icons.menu_book_outlined,
                      title: 'Дневник',
                      subtitle: 'Записи и вопросы для рефлексии',
                      onTap: _openJournal,
                    ),
                    MenuCard(
                      icon: Icons.self_improvement,
                      title: 'Поддержка «Ёжик»',
                      subtitle: 'Добрые слова и подсказки',
                      onTap: _openSupport,
                      color: const Color(0xFF6D8B74),
                    ),
                    MenuCard(
                      icon: Icons.forest_outlined,
                      title: 'Мой лес',
                      subtitle: 'Игровой прогресс за активность',
                      onTap: _openForest,
                      color: const Color(0xFF2E7D32),
                    ),
                    MenuCard(
                      icon: Icons.mood_outlined,
                      title: 'Отметить состояние',
                      subtitle: 'Короткая отметка настроения',
                      onTap: _openMoodCheckIn,
                      color: const Color(0xFF8D6E63),
                    ),
                  ],
                ),
                const SizedBox(height: 20),
                SectionCard(
                  title: 'Мой врач',
                  child: _doctor == null
                      ? const EmptyState(
                          icon: Icons.medical_services_outlined,
                          title: 'Врач пока не привязан',
                          message:
                              'Сообщите врачу свой код, чтобы он добавил вас.',
                        )
                      : ListTile(
                          contentPadding: EdgeInsets.zero,
                          leading: const CircleAvatar(
                            child: Icon(Icons.medical_services_outlined),
                          ),
                          title: Text(_doctor!.displayName),
                          subtitle: Text(
                            '${_doctor!.specialty} · код ${_doctor!.code}',
                          ),
                        ),
                ),
                SectionCard(
                  title: 'Мои назначения',
                  child: _assignments.isEmpty
                      ? const EmptyState(
                          icon: Icons.assignment_outlined,
                          title: 'Назначений пока нет',
                          message: 'Врач добавит назначения после приёма.',
                        )
                      : Column(
                          children: [
                            ...activeAssignments.map(
                              (assignment) => _assignmentTile(assignment),
                            ),
                            if (otherAssignments.isNotEmpty) ...[
                              const Divider(height: 24),
                              Padding(
                                padding: const EdgeInsets.only(bottom: 8),
                                child: Text(
                                  'Завершённые и отменённые',
                                  style: theme.textTheme.bodySmall,
                                ),
                              ),
                              ...otherAssignments.map(
                                (assignment) => _assignmentTile(assignment),
                              ),
                            ],
                          ],
                        ),
                ),
                if (_forest != null)
                  Card(
                    child: ListTile(
                      leading: const Icon(Icons.park_outlined),
                      title: const Text('Лес растёт'),
                      subtitle: Text(
                        'Уровень роста: ${_forest!.growthLevel}. '
                        'Продолжайте вести дневник и выполнять назначения.',
                      ),
                      trailing: Text(
                        '${_forest!.growthLevel}',
                        style: theme.textTheme.titleLarge?.copyWith(
                          color: const Color(0xFF2E7D32),
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ),
                  ),
                if (widget.session.anamnesis.trim().isNotEmpty) ...[
                  const SizedBox(height: 16),
                  SectionCard(
                    title: 'Мой анамнез',
                    child: Text(widget.session.anamnesis),
                  ),
                ],
                const SizedBox(height: 40),
              ],
            ),
          );
  }

  Widget _assignmentTile(AssignmentData assignment) {
    final theme = Theme.of(context);
    return Card(
      elevation: 0,
      color: theme.colorScheme.surfaceContainerHighest,
      margin: const EdgeInsets.only(bottom: 8),
      child: ListTile(
        leading: Icon(_assignmentIcon(assignment.type)),
        title: Text(assignment.title),
        subtitle: Text(
          '${assignment.typeLabel} · ${assignment.statusLabel}\n'
          '${assignment.scheduleLabel}, '
          '${formatDate(assignment.startsAt)} — ${formatDate(assignment.endsAt)}',
        ),
        isThreeLine: true,
        trailing: const Icon(Icons.chevron_right),
        onTap: () => _openAssignment(assignment),
      ),
    );
  }
}

IconData _assignmentIcon(String type) => switch (type) {
  'MEDICATION' => Icons.medication_outlined,
  'QUESTIONNAIRE' => Icons.quiz_outlined,
  'APP_CHECK_IN' => Icons.login,
  'JOURNAL_TASK' => Icons.edit_note,
  _ => Icons.assignment_outlined,
};

class JournalScreen extends StatefulWidget {
  const JournalScreen({super.key, required this.api, this.initialPrompt = ''});

  final ApiClient api;
  final String initialPrompt;

  @override
  State<JournalScreen> createState() => _JournalScreenState();
}

class _JournalScreenState extends State<JournalScreen> {
  List<JournalEntryData> _entries = const [];
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final entries = await widget.api.listJournalEntries();
      if (mounted) setState(() => _entries = entries);
    } on APIException catch (error) {
      if (mounted) setState(() => _error = error.message);
    } catch (error) {
      if (mounted) {
        setState(() => _error = 'Не удалось загрузить дневник: $error');
      }
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _addEntry() async {
    final request = await showDialog<CreateJournalEntryRequest>(
      context: context,
      builder: (_) => JournalEntryDialog(initialPrompt: widget.initialPrompt),
    );
    if (request == null) return;
    try {
      await widget.api.createJournalEntry(request);
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('Запись сохранена')));
      await _load();
    } on APIException catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(error.message)));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Дневник'),
        actions: [
          IconButton(
            tooltip: 'Новая запись',
            onPressed: _addEntry,
            icon: const Icon(Icons.add),
          ),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(20),
              children: [
                Text(
                  'Записная книжка. Предлагаемые вопросы помогают мягкой '
                  'рефлексии — отвечать на них необязательно.',
                  style: Theme.of(context).textTheme.bodyMedium,
                ),
                const SizedBox(height: 16),
                if (_error != null)
                  Text(
                    _error!,
                    style: TextStyle(
                      color: Theme.of(context).colorScheme.error,
                    ),
                  ),
                if (_entries.isEmpty)
                  const EmptyState(
                    icon: Icons.menu_book_outlined,
                    title: 'Записей пока нет',
                    message: 'Начните с одной короткой записи.',
                  )
                else
                  ..._entries.map(
                    (entry) => Card(
                      child: Padding(
                        padding: const EdgeInsets.all(16),
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(
                              formatDateTime(entry.createdAt),
                              style: Theme.of(context).textTheme.bodySmall,
                            ),
                            const SizedBox(height: 6),
                            Text(entry.text),
                            if (entry.reflectionPrompt.isNotEmpty) ...[
                              const SizedBox(height: 8),
                              Text(
                                'Вопрос: ${entry.reflectionPrompt}',
                                style: Theme.of(context).textTheme.bodySmall
                                    ?.copyWith(
                                      color: Theme.of(context)
                                          .colorScheme
                                          .onSurfaceVariant,
                                    ),
                              ),
                            ],
                          ],
                        ),
                      ),
                    ),
                  ),
              ],
            ),
    );
  }
}

class JournalEntryDialog extends StatefulWidget {
  const JournalEntryDialog({super.key, this.initialPrompt = ''});

  final String initialPrompt;

  @override
  State<JournalEntryDialog> createState() => _JournalEntryDialogState();
}

class _JournalEntryDialogState extends State<JournalEntryDialog> {
  final _text = TextEditingController();
  late final TextEditingController _prompt;
  String? _error;

  static const _prompts = [
    'Что сегодня помогло вам почувствовать себя лучше?',
    'Что тревожило вас сегодня?',
    'За что вы можете себя поблагодарить?',
    'Что хотелось бы обсудить с врачом?',
  ];

  @override
  void initState() {
    super.initState();
    _prompt = TextEditingController(text: widget.initialPrompt);
  }

  @override
  void dispose() {
    _text.dispose();
    _prompt.dispose();
    super.dispose();
  }

  void _submit() {
    if (_text.text.trim().isEmpty) {
      setState(() => _error = 'Запись не может быть пустой');
      return;
    }
    Navigator.pop(
      context,
      CreateJournalEntryRequest(
        text: _text.text.trim(),
        reflectionPrompt: _prompt.text.trim(),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Запись в дневник'),
      content: SizedBox(
        width: 560,
        child: SingleChildScrollView(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: _prompts
                    .map(
                      (prompt) => ActionChip(
                        label: Text(prompt),
                        onPressed: () => setState(() => _prompt.text = prompt),
                      ),
                    )
                    .toList(growable: false),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _prompt,
                decoration: const InputDecoration(
                  labelText: 'Вопрос для рефлексии (необязательно)',
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: _text,
                minLines: 6,
                maxLines: 12,
                decoration: const InputDecoration(
                  labelText: 'Текст записи',
                  alignLabelWithHint: true,
                ),
              ),
              if (_error != null) ...[
                const SizedBox(height: 8),
                Text(
                  _error!,
                  style: TextStyle(color: Theme.of(context).colorScheme.error),
                ),
              ],
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: const Text('Отмена'),
        ),
        FilledButton(onPressed: _submit, child: const Text('Сохранить')),
      ],
    );
  }
}

class AssignmentScreen extends StatefulWidget {
  const AssignmentScreen({
    super.key,
    required this.api,
    required this.assignment,
  });

  final ApiClient api;
  final AssignmentData assignment;

  @override
  State<AssignmentScreen> createState() => _AssignmentScreenState();
}

class _AssignmentScreenState extends State<AssignmentScreen> {
  final _comment = TextEditingController();
  final _dosageTaken = TextEditingController();
  final _effects = TextEditingController();
  bool _taken = true;
  bool _busy = false;
  String? _error;

  @override
  void dispose() {
    _comment.dispose();
    _dosageTaken.dispose();
    _effects.dispose();
    super.dispose();
  }

  Future<void> _complete() async {
    final effects = _effects.text
        .split(RegExp(r'[,;\n]'))
        .map((value) => value.trim())
        .where((value) => value.isNotEmpty)
        .toList(growable: false);
    final now = DateTime.now();
    final request = CompleteAssignmentRequest(
      comment: _comment.text.trim(),
      medicationIntake: widget.assignment.type == 'MEDICATION'
          ? MedicationIntakeRequest(
              takenAt: now,
              reportedAsTaken: _taken,
              dosageTaken: _dosageTaken.text.trim(),
            )
          : null,
      sideEffects: effects.isEmpty
          ? null
          : SideEffectReportRequest(
              effects: effects,
              comment: '',
              reportedAt: now,
            ),
    );

    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await widget.api.completeAssignment(widget.assignment.id, request);
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('Выполнение отмечено')));
      Navigator.pop(context, true);
    } on APIException catch (error) {
      if (mounted) setState(() => _error = error.message);
    } catch (error) {
      if (mounted) setState(() => _error = 'Не удалось сохранить: $error');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _openJournalTask() async {
    await Navigator.of(context).push<void>(
      MaterialPageRoute(
        builder: (_) => JournalScreen(
          api: widget.api,
          initialPrompt: widget.assignment.instructions,
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final assignment = widget.assignment;
    final theme = Theme.of(context);
    final canComplete =
        assignment.isActive && assignment.type != 'QUESTIONNAIRE';
    return Scaffold(
      appBar: AppBar(title: Text(assignment.title)),
      body: ListView(
        padding: const EdgeInsets.all(20),
        children: [
          Card(
            child: Padding(
              padding: const EdgeInsets.all(18),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Icon(_assignmentIcon(assignment.type), size: 30),
                      const SizedBox(width: 12),
                      Expanded(
                        child: Text(
                          assignment.typeLabel,
                          style: theme.textTheme.titleLarge?.copyWith(
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                      ),
                      Chip(label: Text(assignment.statusLabel)),
                    ],
                  ),
                  const SizedBox(height: 12),
                  Text('Расписание: ${assignment.scheduleLabel}'),
                  Text(
                    'Период: ${formatDate(assignment.startsAt)} — '
                    '${formatDate(assignment.endsAt)}',
                  ),
                  if (assignment.instructions.isNotEmpty) ...[
                    const SizedBox(height: 12),
                    Text(assignment.instructions),
                  ],
                  if (assignment.dosageLabel.isNotEmpty) ...[
                    const SizedBox(height: 8),
                    Text('Дозировка: ${assignment.dosageLabel}'),
                  ],
                ],
              ),
            ),
          ),
          const SizedBox(height: 16),
          if (assignment.type == 'MEDICATION')
            SectionCard(
              title: 'Отметка о приёме',
              child: Column(
                children: [
                  CheckboxListTile(
                    contentPadding: EdgeInsets.zero,
                    value: _taken,
                    onChanged: (value) =>
                        setState(() => _taken = value ?? false),
                    title: const Text('Лекарство принято'),
                  ),
                  TextField(
                    controller: _dosageTaken,
                    decoration: const InputDecoration(
                      labelText: 'Сколько принято (необязательно)',
                      hintText: 'Например, 1 таблетка',
                    ),
                  ),
                  const SizedBox(height: 12),
                  TextField(
                    controller: _effects,
                    decoration: const InputDecoration(
                      labelText: 'Побочные эффекты (необязательно)',
                      hintText: 'Тошнота, головокружение',
                    ),
                  ),
                  const SizedBox(height: 12),
                  const Text(
                    'Приложение не советует менять дозировку или прекращать '
                    'приём — обсудите это с врачом.',
                  ),
                ],
              ),
            ),
          if (assignment.type == 'JOURNAL_TASK')
            SectionCard(
              title: 'Задание дневника',
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    assignment.instructions.isEmpty
                        ? 'Запишите свои мысли в дневник.'
                        : assignment.instructions,
                  ),
                  const SizedBox(height: 12),
                  OutlinedButton.icon(
                    onPressed: _openJournalTask,
                    icon: const Icon(Icons.edit_note),
                    label: const Text('Открыть дневник'),
                  ),
                ],
              ),
            ),
          if (assignment.type == 'QUESTIONNAIRE')
            const SectionCard(
              title: 'Опросник',
              child: EmptyState(
                icon: Icons.quiz_outlined,
                title: 'Прохождение опросника подключается отдельно',
                message: 'Завершить назначение может врач после консультации.',
              ),
            ),
          SectionCard(
            title: 'Комментарий',
            child: TextField(
              controller: _comment,
              minLines: 2,
              maxLines: 4,
              decoration: const InputDecoration(
                labelText: 'Комментарий к выполнению (необязательно)',
              ),
            ),
          ),
          if (_error != null)
            Padding(
              padding: const EdgeInsets.only(bottom: 12),
              child: Text(
                _error!,
                style: TextStyle(color: theme.colorScheme.error),
              ),
            ),
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              onPressed: canComplete && !_busy ? _complete : null,
              child: Padding(
                padding: const EdgeInsets.symmetric(vertical: 12),
                child: _busy
                    ? const SizedBox(
                        width: 20,
                        height: 20,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : Text(
                        canComplete
                            ? 'Отметить выполнение'
                            : 'Назначение недоступно для выполнения',
                      ),
              ),
            ),
          ),
        ],
      ),
    );
  }
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
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Отметить состояние'),
      content: SizedBox(
        width: 430,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            DropdownButtonFormField<String>(
              initialValue: _mood,
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
              initialValue: _anxiety,
              decoration: const InputDecoration(labelText: 'Тревожность'),
              items: const [
                DropdownMenuItem(value: 'Низкая', child: Text('Низкая')),
                DropdownMenuItem(value: 'Умеренная', child: Text('Умеренная')),
                DropdownMenuItem(value: 'Высокая', child: Text('Высокая')),
              ],
              onChanged: (value) {
                if (value != null) setState(() => _anxiety = value);
              },
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _comment,
              minLines: 2,
              maxLines: 4,
              decoration: const InputDecoration(
                labelText: 'Комментарий (необязательно)',
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
}

class SupportScreen extends StatefulWidget {
  const SupportScreen({super.key, required this.api});

  final ApiClient api;

  @override
  State<SupportScreen> createState() => _SupportScreenState();
}

class _SupportScreenState extends State<SupportScreen> {
  final _input = TextEditingController();
  final _scroll = ScrollController();
  final List<_ChatMessage> _messages = [];
  List<SupportMessageData> _contents = const [];
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _input.dispose();
    _scroll.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final contents = await widget.api.supportMessages();
      if (!mounted) return;
      setState(() {
        _contents = contents;
        _messages.add(
          const _ChatMessage(
            text:
                'Привет! Я Ёжик. Я рядом и помогу добрым словом. '
                'Если станет тяжело — напишите, и я подскажу обратиться к врачу.',
            fromUser: false,
          ),
        );
      });
    } catch (_) {
      if (mounted) {
        setState(() {
          _messages.add(
            const _ChatMessage(
              text:
                  'Привет! Я Ёжик. Если станет тяжело, пожалуйста, '
                  'свяжитесь со своим врачом.',
              fromUser: false,
            ),
          );
        });
      }
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  void _send([String? preset]) {
    final text = (preset ?? _input.text).trim();
    if (text.isEmpty) return;
    _input.clear();
    setState(() {
      _messages.add(_ChatMessage(text: text, fromUser: true));
      _messages.add(_ChatMessage(text: _replyFor(text), fromUser: false));
    });
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scroll.hasClients) {
        _scroll.animateTo(
          _scroll.position.maxScrollExtent,
          duration: const Duration(milliseconds: 220),
          curve: Curves.easeOut,
        );
      }
    });
  }

  String _replyFor(String text) {
    final lower = text.toLowerCase();
    if (lower.contains('врач') || lower.contains('помощ')) {
      return 'Я вижу, тебе нужна помощь. Пожалуйста, обратись к своему врачу — '
          'он знает твою ситуацию и может помочь.';
    }
    if (lower.contains('тревог') ||
        lower.contains('плохо') ||
        lower.contains('не могу') ||
        lower.contains('страш')) {
      return 'Спасибо, что делишься. Попробуй сделать медленный вдох и выдох. '
          'Если тревога не отпускает, обсуди это с врачом — вместе спокойнее.';
    }
    if (lower.contains('пропуск') || lower.contains('лекарств')) {
      return 'Пропуск бывает у всех. Не меняй дозировку самостоятельно: '
          'напиши врачу и уточни, как поступить дальше.';
    }
    final encouragements = _contents
        .where((content) => content.type == 'ENCOURAGEMENT')
        .map((content) => content.message)
        .toList(growable: false);
    if (encouragements.isNotEmpty) {
      return encouragements[_messages.length % encouragements.length];
    }
    return 'Спасибо, что заботишься о себе. Я рядом. Если нужна помощь — '
        'напиши своему врачу.';
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Поддержка «Ёжик»')),
      body: Column(
        children: [
          Expanded(
            child: _loading
                ? const Center(child: CircularProgressIndicator())
                : ListView.builder(
                    controller: _scroll,
                    padding: const EdgeInsets.all(16),
                    itemCount: _messages.length,
                    itemBuilder: (context, index) {
                      final message = _messages[index];
                      return Align(
                        alignment: message.fromUser
                            ? Alignment.centerRight
                            : Alignment.centerLeft,
                        child: Container(
                          margin: const EdgeInsets.only(bottom: 10),
                          padding: const EdgeInsets.symmetric(
                            horizontal: 14,
                            vertical: 10,
                          ),
                          constraints: const BoxConstraints(maxWidth: 520),
                          decoration: BoxDecoration(
                            color: message.fromUser
                                ? Theme.of(context).colorScheme.primaryContainer
                                : Theme.of(context)
                                      .colorScheme
                                      .surfaceContainerHighest,
                            borderRadius: BorderRadius.circular(14),
                          ),
                          child: Text(message.text),
                        ),
                      );
                    },
                  ),
          ),
          Padding(
            padding: const EdgeInsets.all(12),
            child: Column(
              children: [
                Wrap(
                  spacing: 8,
                  runSpacing: 4,
                  children: [
                    ActionChip(
                      label: const Text('Мне тревожно'),
                      onPressed: () => _send('Мне тревожно'),
                    ),
                    ActionChip(
                      label: const Text('Пропустил лекарство'),
                      onPressed: () => _send('Я пропустил лекарство'),
                    ),
                    ActionChip(
                      label: const Text('Связаться с врачом'),
                      onPressed: () => _send('Мне нужна помощь врача'),
                    ),
                  ],
                ),
                const SizedBox(height: 8),
                Row(
                  children: [
                    Expanded(
                      child: TextField(
                        controller: _input,
                        onSubmitted: (_) => _send(),
                        decoration: const InputDecoration(
                          hintText: 'Напишите Ёжику...',
                        ),
                      ),
                    ),
                    const SizedBox(width: 8),
                    IconButton.filled(
                      onPressed: () => _send(),
                      icon: const Icon(Icons.send),
                    ),
                  ],
                ),
                const SizedBox(height: 4),
                Text(
                  'Ёжик отвечает подготовленными фразами. Это не медицинская '
                  'консультация и не замена врачу.',
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _ChatMessage {
  const _ChatMessage({required this.text, required this.fromUser});

  final String text;
  final bool fromUser;
}

class ForestScreen extends StatefulWidget {
  const ForestScreen({super.key, required this.api});

  final ApiClient api;

  @override
  State<ForestScreen> createState() => _ForestScreenState();
}

class _ForestScreenState extends State<ForestScreen> {
  ForestData? _forest;
  bool _loading = true;
  String? _error;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final forest = await widget.api.forest();
      if (mounted) setState(() => _forest = forest);
    } on APIException catch (error) {
      if (mounted) setState(() => _error = error.message);
    } catch (error) {
      if (mounted) setState(() => _error = 'Не удалось загрузить лес: $error');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final forest = _forest;
    final growth = forest?.growthLevel ?? 0;
    final trees = (growth ~/ 10).clamp(1, 12).toInt();
    return Scaffold(
      appBar: AppBar(title: const Text('Мой лес')),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : ListView(
              padding: const EdgeInsets.all(20),
              children: [
                Card(
                  child: Padding(
                    padding: const EdgeInsets.all(20),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          'Уровень роста: $growth',
                          style: theme.textTheme.headlineSmall?.copyWith(
                            fontWeight: FontWeight.w700,
                          ),
                        ),
                        const SizedBox(height: 12),
                        Wrap(
                          spacing: 6,
                          runSpacing: 6,
                          children: List.generate(
                            trees,
                            (index) => Text(
                              index < 3 ? '🌱' : (index < 7 ? '🌿' : '🌳'),
                              style: const TextStyle(fontSize: 34),
                            ),
                          ),
                        ),
                        const SizedBox(height: 12),
                        LinearProgressIndicator(
                          value: (growth % 100) / 100,
                          minHeight: 8,
                          borderRadius: BorderRadius.circular(4),
                        ),
                        const SizedBox(height: 8),
                        const Text(
                          'Лес растёт за записи в дневнике и выполненные '
                          'назначения. Это игровая обратная связь, а не оценка '
                          'вашего состояния.',
                        ),
                      ],
                    ),
                  ),
                ),
                if (_error != null)
                  Text(
                    _error!,
                    style: TextStyle(color: theme.colorScheme.error),
                  ),
                const SizedBox(height: 12),
                SectionCard(
                  title: 'Что помогло лесу вырасти',
                  child: forest == null || forest.activities.isEmpty
                      ? const EmptyState(
                          icon: Icons.park_outlined,
                          title: 'Пока тихо',
                          message: 'Запишите мысль в дневник или выполните назначение.',
                        )
                      : Column(
                          children: forest.activities
                              .take(12)
                              .map(
                                (activity) => ListTile(
                                  contentPadding: EdgeInsets.zero,
                                  leading: const Icon(Icons.add_circle_outline),
                                  title: Text(activity.label),
                                  subtitle: Text(
                                    formatDateTime(activity.occurredAt),
                                  ),
                                  trailing: Text('+${activity.points}'),
                                ),
                              )
                              .toList(growable: false),
                        ),
                ),
              ],
            ),
    );
  }
}
