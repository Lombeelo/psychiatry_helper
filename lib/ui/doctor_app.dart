import 'package:flutter/material.dart';

import '../data/api/api_client.dart';
import '../data/api/api_requests.dart';
import '../data/api/models.dart';
import 'widgets.dart';

class DoctorApp extends StatefulWidget {
  const DoctorApp({
    super.key,
    required this.api,
    required this.session,
    required this.onLogout,
  });

  final ApiClient api;
  final AuthSession session;
  final VoidCallback onLogout;

  @override
  State<DoctorApp> createState() => _DoctorAppState();
}

class _DoctorAppState extends State<DoctorApp> {
  List<PatientSummary> _patients = const [];
  List<NotificationData> _notifications = const [];
  List<MedicationGroupData> _groups = const [];
  PatientDetailsData? _details;
  String? _selectedId;
  bool _loading = true;
  bool _loadingDetails = false;
  bool _showPrescribe = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _loadAll();
  }

  Future<void> _loadAll() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final patients = await widget.api.listPatients();
      final notifications = await widget.api.listNotifications();
      final groups = await widget.api.listMedicationGroups();
      if (!mounted) return;
      setState(() {
        _patients = patients;
        _notifications = notifications;
        _groups = groups;
      });
      if (patients.isNotEmpty) {
        await _selectPatient(_selectedId ?? patients.first.id);
      }
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

  Future<void> _selectPatient(String patientId) async {
    setState(() {
      _selectedId = patientId;
      _details = null;
      _loadingDetails = true;
      _showPrescribe = false;
      _error = null;
    });
    try {
      final details = await widget.api.getPatientDetails(patientId);
      if (mounted) setState(() => _details = details);
    } on APIException catch (error) {
      if (mounted) setState(() => _error = error.message);
    } catch (error) {
      if (mounted) {
        setState(() => _error = 'Не удалось загрузить карточку: $error');
      }
    } finally {
      if (mounted) setState(() => _loadingDetails = false);
    }
  }

  Future<void> _refreshDetails() async {
    final id = _selectedId;
    if (id == null) return;
    await _selectPatient(id);
    await _refreshNotifications();
  }

  Future<void> _refreshNotifications() async {
    try {
      final notifications = await widget.api.listNotifications();
      if (mounted) setState(() => _notifications = notifications);
    } catch (_) {
      // Уведомления не должны блокировать карточку пациента.
    }
  }

  Future<void> _openNotifications() async {
    await _refreshNotifications();
    if (!mounted) return;
    await showDialog<void>(
      context: context,
      builder: (dialogContext) => AlertDialog(
        title: Row(
          children: [
            const Icon(Icons.notifications_outlined),
            const SizedBox(width: 8),
            const Expanded(child: Text('Уведомления')),
            if (_unreadCount > 0) Chip(label: Text('$_unreadCount новых')),
          ],
        ),
        content: SizedBox(
          width: 540,
          height: 420,
          child: _notifications.isEmpty
              ? const EmptyState(
                  icon: Icons.notifications_none,
                  title: 'Уведомлений нет',
                  message: 'Здесь появятся события пациентов.',
                )
              : ListView.separated(
                  itemCount: _notifications.length,
                  separatorBuilder: (_, _) => const Divider(height: 1),
                  itemBuilder: (context, index) {
                    final notification = _notifications[index];
                    return ListTile(
                      leading: Icon(
                        notification.isUnread
                            ? Icons.mark_email_unread_outlined
                            : Icons.mark_email_read_outlined,
                      ),
                      title: Text(notification.message),
                      subtitle: Text(formatDateTime(notification.createdAt)),
                      onTap: () async {
                        Navigator.pop(dialogContext);
                        if (notification.isUnread) {
                          await _markRead(notification);
                        }
                        if (notification.patientId.isNotEmpty) {
                          await _selectPatient(notification.patientId);
                        }
                      },
                    );
                  },
                ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(dialogContext),
            child: const Text('Закрыть'),
          ),
        ],
      ),
    );
  }

  Future<void> _markRead(NotificationData notification) async {
    try {
      await widget.api.markNotificationRead(notification.id);
      await _refreshNotifications();
    } catch (_) {
      // Не критично для работы карточки.
    }
  }

  int get _unreadCount =>
      _notifications.where((notification) => notification.isUnread).length;

  void _logout() {
    widget.api.setToken(null);
    widget.onLogout();
  }

  Future<void> _addPatient() async {
    final code = await showDialog<String>(
      context: context,
      builder: (_) => const _AddPatientDialog(),
    );
    if (code == null || code.trim().isEmpty) {
      return;
    }
    try {
      await widget.api.createRelationship(code);
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(const SnackBar(content: Text('Пациент привязан')));
      await _loadAll();
    } on APIException catch (error) {
      if (!mounted) return;
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(error.message)));
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('Кабинет врача'),
            Text(widget.session.displayName, style: theme.textTheme.bodySmall),
          ],
        ),
        actions: [
          IconButton(
            tooltip: 'Добавить пациента по коду',
            onPressed: _loading ? null : _addPatient,
            icon: const Icon(Icons.person_add_alt_1),
          ),
          IconButton(
            tooltip: 'Уведомления',
            onPressed: _loading ? null : _openNotifications,
            icon: Stack(
              clipBehavior: Clip.none,
              children: [
                const Icon(Icons.notifications_none),
                if (_unreadCount > 0)
                  Positioned(
                    right: -5,
                    top: -5,
                    child: Container(
                      padding: const EdgeInsets.symmetric(
                        horizontal: 5,
                        vertical: 1,
                      ),
                      decoration: BoxDecoration(
                        color: theme.colorScheme.error,
                        borderRadius: BorderRadius.circular(10),
                      ),
                      child: Text(
                        '$_unreadCount',
                        style: TextStyle(
                          color: theme.colorScheme.onError,
                          fontSize: 11,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ),
                  ),
              ],
            ),
          ),
          PopupMenuButton<String>(
            onSelected: (value) {
              if (value == 'logout') _logout();
              if (value == 'refresh') _loadAll();
            },
            itemBuilder: (_) => const [
              PopupMenuItem(value: 'refresh', child: Text('Обновить данные')),
              PopupMenuItem(value: 'logout', child: Text('Выйти')),
            ],
          ),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _showPrescribe && _details != null
          ? _PrescribeMedicationPage(
              api: widget.api,
              patient: _details!.patient,
              details: _details!,
              groups: _groups,
              onBack: () => setState(() => _showPrescribe = false),
              onCreated: () async {
                final messenger = ScaffoldMessenger.of(context);
                setState(() => _showPrescribe = false);
                await _refreshDetails();
                if (!mounted) return;
                messenger.showSnackBar(
                  const SnackBar(content: Text('Лекарство назначено')),
                );
              },
            )
          : Row(
              crossAxisAlignment: CrossAxisAlignment.stretch,
              children: [
                _PatientListPane(
                  patients: _patients,
                  selectedId: _selectedId,
                  onSelected: _selectPatient,
                ),
                const VerticalDivider(width: 1),
                Expanded(child: _buildPatientArea(theme)),
              ],
            ),
    );
  }

  Widget _buildPatientArea(ThemeData theme) {
    if (_error != null) {
      return Center(
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxWidth: 520),
          child: Card(
            child: Padding(
              padding: const EdgeInsets.all(20),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(Icons.error_outline, color: theme.colorScheme.error),
                  const SizedBox(height: 8),
                  Text(_error!, textAlign: TextAlign.center),
                  const SizedBox(height: 12),
                  FilledButton(
                    onPressed: _loadAll,
                    child: const Text('Повторить'),
                  ),
                ],
              ),
            ),
          ),
        ),
      );
    }
    if (_patients.isEmpty) {
      return const Center(
        child: EmptyState(
          icon: Icons.people_outline,
          title: 'Пациентов пока нет',
          message: 'Привязка пациента выполняется по его коду.',
        ),
      );
    }
    if (_loadingDetails) {
      return const Center(child: CircularProgressIndicator());
    }
    final details = _details;
    if (details == null) {
      return const Center(child: Text('Выберите пациента'));
    }
    return _PatientDetailsView(
      details: details,
      onPrescribe: () => setState(() => _showPrescribe = true),
      onRefresh: _refreshDetails,
    );
  }
}

class _PatientListPane extends StatelessWidget {
  const _PatientListPane({
    required this.patients,
    required this.selectedId,
    required this.onSelected,
  });

  final List<PatientSummary> patients;
  final String? selectedId;
  final ValueChanged<String> onSelected;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      width: 320,
      color: theme.colorScheme.surface,
      child: Column(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(18, 18, 18, 10),
            child: Row(
              children: [
                const Icon(Icons.people_outline),
                const SizedBox(width: 8),
                Text(
                  'Пациенты',
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.w700,
                  ),
                ),
                const Spacer(),
                Text('${patients.length}'),
              ],
            ),
          ),
          const Divider(height: 1),
          Expanded(
            child: patients.isEmpty
                ? const Center(
                    child: Padding(
                      padding: EdgeInsets.all(20),
                      child: Text('Нет привязанных пациентов'),
                    ),
                  )
                : ListView.builder(
                    itemCount: patients.length,
                    itemBuilder: (context, index) {
                      final patient = patients[index];
                      final selected = patient.id == selectedId;
                      return Material(
                        color: selected
                            ? theme.colorScheme.primaryContainer
                            : Colors.transparent,
                        child: InkWell(
                          onTap: () => onSelected(patient.id),
                          child: Padding(
                            padding: const EdgeInsets.symmetric(
                              horizontal: 18,
                              vertical: 14,
                            ),
                            child: Row(
                              children: [
                                Tooltip(
                                  message: patient.status.label,
                                  child: StatusBadge(
                                    status: patient.status,
                                    compact: true,
                                  ),
                                ),
                                const SizedBox(width: 12),
                                Expanded(
                                  child: Column(
                                    crossAxisAlignment:
                                        CrossAxisAlignment.start,
                                    children: [
                                      Text(
                                        patient.displayName,
                                        maxLines: 1,
                                        overflow: TextOverflow.ellipsis,
                                        style: const TextStyle(
                                          fontWeight: FontWeight.w600,
                                        ),
                                      ),
                                      const SizedBox(height: 2),
                                      Text(
                                        patient.code,
                                        style: theme.textTheme.bodySmall
                                            ?.copyWith(
                                              color: theme
                                                  .colorScheme
                                                  .onSurfaceVariant,
                                            ),
                                      ),
                                    ],
                                  ),
                                ),
                              ],
                            ),
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

class _PatientDetailsView extends StatelessWidget {
  const _PatientDetailsView({
    required this.details,
    required this.onPrescribe,
    required this.onRefresh,
  });

  final PatientDetailsData details;
  final VoidCallback onPrescribe;
  final VoidCallback onRefresh;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final patient = details.patient;
    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: 980),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Card(
              child: Padding(
                padding: const EdgeInsets.all(20),
                child: Row(
                  children: [
                    CircleAvatar(
                      radius: 28,
                      child: Text(
                        patient.displayName.isEmpty
                            ? '?'
                            : patient.displayName[0].toUpperCase(),
                        style: const TextStyle(
                          fontSize: 22,
                          fontWeight: FontWeight.w700,
                        ),
                      ),
                    ),
                    const SizedBox(width: 16),
                    Expanded(
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Text(
                            patient.displayName,
                            style: theme.textTheme.headlineSmall?.copyWith(
                              fontWeight: FontWeight.w700,
                            ),
                          ),
                          const SizedBox(height: 4),
                          Text('Код: ${patient.code}'),
                        ],
                      ),
                    ),
                    IconButton(
                      tooltip: 'Обновить',
                      onPressed: onRefresh,
                      icon: const Icon(Icons.refresh),
                    ),
                    const SizedBox(width: 8),
                    FilledButton.icon(
                      onPressed: onPrescribe,
                      icon: const Icon(Icons.medication_outlined),
                      label: const Text('Назначить лекарство'),
                    ),
                  ],
                ),
              ),
            ),
            const SizedBox(height: 16),
            SectionCard(
              title: 'Индикатор состояния',
              trailing: Text(
                'расчёт: ${formatDateTime(patient.status.calculatedAt)}',
                style: theme.textTheme.bodySmall,
              ),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  StatusBadge(status: patient.status),
                  const SizedBox(height: 10),
                  if (!patient.status.dataSufficient)
                    Text(
                      'Недостаточно данных для расчёта: нет активных '
                      'лекарственных назначений за период.',
                      style: theme.textTheme.bodyMedium,
                    )
                  else if (patient.status.reasons.isEmpty)
                    const Text('Отклонений не выявлено.')
                  else
                    ...patient.status.reasons.map(
                      (reason) => Padding(
                        padding: const EdgeInsets.only(bottom: 4),
                        child: Row(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            const Text('• '),
                            Expanded(child: Text(reason)),
                          ],
                        ),
                      ),
                    ),
                  const SizedBox(height: 8),
                  Text(
                    'Светофор — вспомогательный индикатор, а не диагноз '
                    'и не оценка экстренности.',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ],
              ),
            ),
            SectionCard(
              title: 'Анамнез',
              child: patient.notes.isEmpty
                  ? const Text('Пациент ещё не заполнил анамнез.')
                  : Text(patient.notes),
            ),
            SectionCard(
              title: 'Назначения',
              child: details.assignments.isEmpty
                  ? const EmptyState(
                      icon: Icons.assignment_outlined,
                      title: 'Назначений нет',
                      message: 'Выпишите лекарство или другое назначение.',
                    )
                  : Column(
                      children: details.assignments
                          .map(
                            (assignment) =>
                                _assignmentTile(context, assignment),
                          )
                          .toList(growable: false),
                    ),
            ),
            SectionCard(
              title: 'Информация о пройденных опросниках',
              child: const EmptyState(
                icon: Icons.quiz_outlined,
                title: 'Пока нет пройденных опросников',
                message: 'Раздел опросников подключается отдельным модулем.',
              ),
            ),
            SectionCard(
              title: 'Дневник',
              child: details.journalEntries.isEmpty
                  ? const EmptyState(
                      icon: Icons.menu_book_outlined,
                      title: 'Записей пока нет',
                      message: 'Пациент ещё не вёл дневник.',
                    )
                  : Column(
                      children: details.journalEntries
                          .map((entry) => _journalTile(context, entry))
                          .toList(growable: false),
                    ),
            ),
            SectionCard(
              title: 'Отметки состояния',
              child: details.moodCheckIns.isEmpty
                  ? const EmptyState(
                      icon: Icons.mood_outlined,
                      title: 'Отметок пока нет',
                      message: 'Пациент ещё не отмечал состояние.',
                    )
                  : Column(
                      children: details.moodCheckIns
                          .map((checkIn) => _moodTile(context, checkIn))
                          .toList(growable: false),
                    ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _assignmentTile(BuildContext context, AssignmentData assignment) {
    final theme = Theme.of(context);
    return Card(
      elevation: 0,
      color: theme.colorScheme.surfaceContainerHighest,
      margin: const EdgeInsets.only(bottom: 8),
      child: ListTile(
        leading: Icon(_assignmentIcon(assignment.type)),
        title: Text(assignment.title),
        subtitle: Text(
          '${assignment.typeLabel} · ${assignment.scheduleLabel}\n'
          '${formatDate(assignment.startsAt)} — ${formatDate(assignment.endsAt)}',
        ),
        isThreeLine: true,
        trailing: Chip(label: Text(assignment.statusLabel)),
      ),
    );
  }

  Widget _journalTile(BuildContext context, JournalEntryData entry) {
    final theme = Theme.of(context);
    return Card(
      elevation: 0,
      color: theme.colorScheme.surfaceContainerHighest,
      margin: const EdgeInsets.only(bottom: 8),
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              formatDateTime(entry.createdAt),
              style: theme.textTheme.bodySmall,
            ),
            const SizedBox(height: 4),
            Text(entry.text),
            if (entry.reflectionPrompt.isNotEmpty) ...[
              const SizedBox(height: 6),
              Text(
                'Вопрос: ${entry.reflectionPrompt}',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.onSurfaceVariant,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _moodTile(BuildContext context, MoodCheckInData checkIn) {
    final theme = Theme.of(context);
    return Card(
      elevation: 0,
      color: theme.colorScheme.surfaceContainerHighest,
      margin: const EdgeInsets.only(bottom: 8),
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              formatDateTime(checkIn.createdAt),
              style: theme.textTheme.bodySmall,
            ),
            const SizedBox(height: 4),
            Text(
              checkIn.answers
                  .map((answer) => '${answer.key}: ${answer.value}')
                  .join(' · '),
            ),
            if (checkIn.comment.isNotEmpty) ...[
              const SizedBox(height: 4),
              Text(checkIn.comment, style: theme.textTheme.bodySmall),
            ],
          ],
        ),
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

class _PrescribeMedicationPage extends StatefulWidget {
  const _PrescribeMedicationPage({
    required this.api,
    required this.patient,
    required this.details,
    required this.groups,
    required this.onBack,
    required this.onCreated,
  });

  final ApiClient api;
  final PatientSummary patient;
  final PatientDetailsData details;
  final List<MedicationGroupData> groups;
  final VoidCallback onBack;
  final VoidCallback onCreated;

  @override
  State<_PrescribeMedicationPage> createState() =>
      _PrescribeMedicationPageState();
}

class _PrescribeMedicationPageState extends State<_PrescribeMedicationPage> {
  final _amount = TextEditingController();
  final _unit = TextEditingController(text: 'мг');
  final _times = TextEditingController(text: '09:00');
  final _start = TextEditingController(text: formatDateInput(DateTime.now()));
  final _end = TextEditingController(
    text: formatDateInput(DateTime.now().add(const Duration(days: 30))),
  );
  final _instructions = TextEditingController();

  String? _groupId;
  String? _medicationId;
  bool _busy = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    if (widget.groups.isNotEmpty) {
      _groupId = widget.groups.first.id;
      final medications = widget.groups.first.medications;
      if (medications.isNotEmpty) {
        _medicationId = medications.first.id;
        _instructions.text = medications.first.instruction;
      }
    }
  }

  @override
  void dispose() {
    _amount.dispose();
    _unit.dispose();
    _times.dispose();
    _start.dispose();
    _end.dispose();
    _instructions.dispose();
    super.dispose();
  }

  MedicationGroupData? get _selectedGroup {
    for (final group in widget.groups) {
      if (group.id == _groupId) return group;
    }
    return null;
  }

  MedicationData? get _selectedMedication {
    final group = _selectedGroup;
    if (group == null) return null;
    for (final medication in group.medications) {
      if (medication.id == _medicationId) return medication;
    }
    return null;
  }

  void _selectGroup(MedicationGroupData group) {
    setState(() {
      _groupId = group.id;
      _medicationId = group.medications.isEmpty
          ? null
          : group.medications.first.id;
      _instructions.text = group.medications.isEmpty
          ? ''
          : group.medications.first.instruction;
    });
  }

  void _selectMedication(MedicationData medication) {
    setState(() {
      _medicationId = medication.id;
      _instructions.text = medication.instruction;
    });
  }

  Future<void> _submit() async {
    final medication = _selectedMedication;
    if (medication == null) {
      setState(() => _error = 'Выберите лекарство');
      return;
    }
    final start = DateTime.tryParse(_start.text.trim());
    final end = DateTime.tryParse(_end.text.trim());
    final amount = double.tryParse(_amount.text.trim().replaceAll(',', '.'));
    if (start == null || end == null || !end.isAfter(start)) {
      setState(() => _error = 'Проверьте период приёма');
      return;
    }
    if (amount == null || amount <= 0) {
      setState(() => _error = 'Укажите положительную дозировку');
      return;
    }
    final times = _times.text
        .split(',')
        .map((value) => value.trim())
        .where((value) => value.isNotEmpty)
        .toList(growable: false);
    if (times.isEmpty) {
      setState(() => _error = 'Укажите время приёма');
      return;
    }

    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await widget.api.createAssignment(
        CreateAssignmentRequest(
          patientId: widget.patient.id,
          title:
              '${medication.name} ${_amount.text.trim()} ${_unit.text.trim()}',
          startsAt: start,
          endsAt: end,
          frequency: 'DAILY',
          daysOfWeek: const [1, 2, 3, 4, 5, 6, 7],
          timesOfDay: times,
          type: 'MEDICATION',
          details: {
            'medication_id': medication.id,
            'dosage': {
              'amount': amount,
              'unit': _unit.text.trim(),
              'instructions': _instructions.text.trim(),
            },
            'instructions': _instructions.text.trim(),
          },
        ),
      );
      if (mounted) widget.onCreated();
    } on APIException catch (error) {
      if (mounted) setState(() => _error = error.message);
    } catch (error) {
      if (mounted) setState(() => _error = 'Не удалось назначить: $error');
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final group = _selectedGroup;
    final medication = _selectedMedication;
    return Column(
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(18, 14, 18, 10),
          child: Row(
            children: [
              IconButton(
                tooltip: 'Назад',
                onPressed: widget.onBack,
                icon: const Icon(Icons.arrow_back),
              ),
              const SizedBox(width: 4),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Меню выписываемых лекарств',
                      style: theme.textTheme.titleLarge?.copyWith(
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    Text('Пациент: ${widget.patient.displayName}'),
                  ],
                ),
              ),
            ],
          ),
        ),
        const Divider(height: 1),
        Expanded(
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              SizedBox(
                width: 230,
                child: ListView(
                  children: [
                    const Padding(
                      padding: EdgeInsets.fromLTRB(16, 16, 16, 8),
                      child: Text(
                        'Группы',
                        style: TextStyle(fontWeight: FontWeight.w700),
                      ),
                    ),
                    ...widget.groups.map(
                      (item) => ListTile(
                        selected: item.id == _groupId,
                        title: Text(item.name),
                        subtitle: Text(
                          item.description,
                          maxLines: 2,
                          overflow: TextOverflow.ellipsis,
                        ),
                        onTap: () => _selectGroup(item),
                      ),
                    ),
                  ],
                ),
              ),
              const VerticalDivider(width: 1),
              SizedBox(
                width: 260,
                child: ListView(
                  children: [
                    const Padding(
                      padding: EdgeInsets.fromLTRB(16, 16, 16, 8),
                      child: Text(
                        'Лекарства',
                        style: TextStyle(fontWeight: FontWeight.w700),
                      ),
                    ),
                    if (group != null)
                      ...group.medications.map(
                        (item) => ListTile(
                          selected: item.id == _medicationId,
                          title: Text(item.name),
                          onTap: () => _selectMedication(item),
                        ),
                      ),
                  ],
                ),
              ),
              const VerticalDivider(width: 1),
              Expanded(
                child: SingleChildScrollView(
                  padding: const EdgeInsets.all(22),
                  child: ConstrainedBox(
                    constraints: const BoxConstraints(maxWidth: 620),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        if (medication == null)
                          const EmptyState(
                            icon: Icons.medication_outlined,
                            title: 'Выберите лекарство',
                            message: 'Сначала выберите группу и препарат.',
                          )
                        else ...[
                          SectionCard(
                            title: medication.name,
                            child: Text(
                              medication.instruction.isEmpty
                                  ? 'Инструкция не заполнена.'
                                  : medication.instruction,
                            ),
                          ),
                          Row(
                            children: [
                              Expanded(
                                child: TextField(
                                  controller: _amount,
                                  decoration: const InputDecoration(
                                    labelText: 'Дозировка',
                                  ),
                                ),
                              ),
                              const SizedBox(width: 12),
                              Expanded(
                                child: TextField(
                                  controller: _unit,
                                  decoration: const InputDecoration(
                                    labelText: 'Ед. изм.',
                                  ),
                                ),
                              ),
                            ],
                          ),
                          const SizedBox(height: 12),
                          Row(
                            children: [
                              Expanded(
                                child: TextField(
                                  controller: _start,
                                  decoration: const InputDecoration(
                                    labelText: 'Период с',
                                    hintText: 'ГГГГ-ММ-ДД',
                                  ),
                                ),
                              ),
                              const SizedBox(width: 12),
                              Expanded(
                                child: TextField(
                                  controller: _end,
                                  decoration: const InputDecoration(
                                    labelText: 'Период по',
                                    hintText: 'ГГГГ-ММ-ДД',
                                  ),
                                ),
                              ),
                            ],
                          ),
                          const SizedBox(height: 12),
                          TextField(
                            controller: _times,
                            decoration: const InputDecoration(
                              labelText: 'Время приёма',
                              hintText: '09:00, 21:00',
                            ),
                          ),
                          const SizedBox(height: 12),
                          TextField(
                            controller: _instructions,
                            minLines: 2,
                            maxLines: 4,
                            decoration: const InputDecoration(
                              labelText: 'Инструкции пациенту',
                              alignLabelWithHint: true,
                            ),
                          ),
                          const SizedBox(height: 16),
                          _SelectionHelp(details: widget.details),
                          if (_error != null) ...[
                            const SizedBox(height: 12),
                            Text(
                              _error!,
                              style: TextStyle(color: theme.colorScheme.error),
                            ),
                          ],
                          const SizedBox(height: 16),
                          SizedBox(
                            width: double.infinity,
                            child: FilledButton.icon(
                              onPressed: _busy ? null : _submit,
                              icon: const Icon(Icons.check),
                              label: Padding(
                                padding: const EdgeInsets.symmetric(
                                  vertical: 10,
                                ),
                                child: _busy
                                    ? const SizedBox(
                                        width: 20,
                                        height: 20,
                                        child: CircularProgressIndicator(
                                          strokeWidth: 2,
                                        ),
                                      )
                                    : const Text('Назначить'),
                              ),
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
        ),
      ],
    );
  }
}

class _SelectionHelp extends StatelessWidget {
  const _SelectionHelp({required this.details});

  final PatientDetailsData details;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    MoodCheckInData? latestMood;
    if (details.moodCheckIns.isNotEmpty) {
      latestMood = details.moodCheckIns.first;
    }

    return Card(
      elevation: 0,
      color: theme.colorScheme.secondaryContainer,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.lightbulb_outline),
                const SizedBox(width: 8),
                Text(
                  'Помощь подбора',
                  style: theme.textTheme.titleSmall?.copyWith(
                    fontWeight: FontWeight.w700,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              latestMood == null
                  ? 'У пациента пока нет отметок состояния. '
                        'Обсудите выбор препарата на приёме.'
                  : 'Последняя отметка: '
                        '${latestMood.answers.map((a) => '${a.key} — ${a.value}').join(', ')}.'
                        '${latestMood.comment.isEmpty ? '' : ' Комментарий: ${latestMood.comment}.'}',
            ),
            const SizedBox(height: 6),
            Text(
              'Это подсказка для разговора с пациентом, а не автоматическая '
              'рекомендация лечения.',
              style: theme.textTheme.bodySmall,
            ),
          ],
        ),
      ),
    );
  }
}

class _AddPatientDialog extends StatefulWidget {
  const _AddPatientDialog();

  @override
  State<_AddPatientDialog> createState() => _AddPatientDialogState();
}

class _AddPatientDialogState extends State<_AddPatientDialog> {
  final _code = TextEditingController();
  String? _error;

  @override
  void dispose() {
    _code.dispose();
    super.dispose();
  }

  void _submit() {
    final code = _code.text.trim().toUpperCase();
    if (!RegExp(r'^P[A-Z0-9]{6}$').hasMatch(code)) {
      setState(() => _error = 'Код пациента должен иметь вид PXXXXXX');
      return;
    }
    Navigator.pop(context, code);
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Добавить пациента'),
      content: SizedBox(
        width: 420,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Text(
              'Введите код пациента из его личного кабинета. '
              'Связь создаётся сразу, подтверждение второй стороны не нужно.',
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _code,
              textCapitalization: TextCapitalization.characters,
              decoration: const InputDecoration(
                labelText: 'Код пациента',
                hintText: 'PXXXXXX',
              ),
              onSubmitted: (_) => _submit(),
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
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: const Text('Отмена'),
        ),
        FilledButton(onPressed: _submit, child: const Text('Привязать')),
      ],
    );
  }
}
