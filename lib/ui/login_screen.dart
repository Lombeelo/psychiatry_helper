import 'package:flutter/material.dart';

import '../data/api/api_client.dart';
import '../data/api/models.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key, required this.api, required this.onLoggedIn});

  final ApiClient api;
  final ValueChanged<AuthSession> onLoggedIn;

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _login = TextEditingController();
  final _password = TextEditingController();
  String? _role;
  bool _busy = false;
  bool _obscure = true;
  String? _error;

  @override
  void dispose() {
    _login.dispose();
    _password.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final login = _login.text.trim();
    final password = _password.text;
    if (login.isEmpty || password.isEmpty) {
      setState(() => _error = 'Введите логин и пароль');
      return;
    }

    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final session = await widget.api.login(
        login: login,
        password: password,
        role: _role ?? 'PATIENT',
      );
      widget.api.setToken(session.token);
      if (!mounted) return;
      widget.onLoggedIn(session);
    } on APIException catch (error) {
      if (mounted) {
        setState(() => _error = error.message);
      }
    } catch (error) {
      if (mounted) {
        setState(() => _error = 'Не удалось войти: $error');
      }
    } finally {
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      body: DecoratedBox(
        decoration: const BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topLeft,
            end: Alignment.bottomRight,
            colors: [Color(0xFFE8F0ED), Color(0xFFF7F9F8)],
          ),
        ),
        child: SafeArea(
          child: Center(
            child: SingleChildScrollView(
              padding: const EdgeInsets.all(24),
              child: ConstrainedBox(
                constraints: const BoxConstraints(maxWidth: 520),
                child: Card(
                  elevation: 2,
                  child: Padding(
                    padding: const EdgeInsets.all(28),
                    child: AnimatedSize(
                      duration: const Duration(milliseconds: 180),
                      child: _role == null
                          ? _buildRoleChoice(theme)
                          : _buildLoginForm(theme),
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }

  Widget _buildRoleChoice(ThemeData theme) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        const Icon(Icons.health_and_safety, size: 44, color: Color(0xFF39796B)),
        const SizedBox(height: 12),
        Text(
          'Сопровождение пациента',
          textAlign: TextAlign.center,
          style: theme.textTheme.headlineSmall?.copyWith(
            fontWeight: FontWeight.w700,
          ),
        ),
        const SizedBox(height: 6),
        Text(
          'Выберите, как вы хотите войти',
          textAlign: TextAlign.center,
          style: theme.textTheme.bodyMedium?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: 26),
        _RoleCard(
          icon: Icons.medical_services_outlined,
          title: 'Я врач',
          subtitle: 'Список пациентов, светофор и назначения',
          onTap: () => setState(() => _role = 'DOCTOR'),
        ),
        const SizedBox(height: 12),
        _RoleCard(
          icon: Icons.person_outline,
          title: 'Я пациент',
          subtitle: 'Дневник, назначения и поддержка',
          onTap: () => setState(() => _role = 'PATIENT'),
        ),
      ],
    );
  }

  Widget _buildLoginForm(ThemeData theme) {
    final isDoctor = _role == 'DOCTOR';
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Row(
          children: [
            IconButton(
              tooltip: 'Назад',
              onPressed: _busy
                  ? null
                  : () => setState(() {
                      _role = null;
                      _error = null;
                    }),
              icon: const Icon(Icons.arrow_back),
            ),
            const Spacer(),
            Chip(
              avatar: Icon(
                isDoctor
                    ? Icons.medical_services_outlined
                    : Icons.person_outline,
                size: 18,
              ),
              label: Text(isDoctor ? 'Врач' : 'Пациент'),
            ),
          ],
        ),
        const SizedBox(height: 12),
        Text(
          'Вход в систему',
          style: theme.textTheme.headlineSmall?.copyWith(
            fontWeight: FontWeight.w700,
          ),
        ),
        const SizedBox(height: 20),
        TextField(
          controller: _login,
          autofillHints: const [AutofillHints.username],
          textInputAction: TextInputAction.next,
          decoration: const InputDecoration(
            labelText: 'Логин',
            prefixIcon: Icon(Icons.person_outline),
          ),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: _password,
          obscureText: _obscure,
          autofillHints: const [AutofillHints.password],
          onSubmitted: (_) => _busy ? null : _submit(),
          decoration: InputDecoration(
            labelText: 'Пароль',
            prefixIcon: const Icon(Icons.lock_outline),
            suffixIcon: IconButton(
              tooltip: _obscure ? 'Показать пароль' : 'Скрыть пароль',
              onPressed: () => setState(() => _obscure = !_obscure),
              icon: Icon(
                _obscure
                    ? Icons.visibility_outlined
                    : Icons.visibility_off_outlined,
              ),
            ),
          ),
        ),
        if (_error != null) ...[
          const SizedBox(height: 12),
          Text(_error!, style: TextStyle(color: theme.colorScheme.error)),
        ],
        const SizedBox(height: 18),
        FilledButton(
          onPressed: _busy ? null : _submit,
          child: Padding(
            padding: const EdgeInsets.symmetric(vertical: 12),
            child: _busy
                ? const SizedBox(
                    width: 20,
                    height: 20,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Text('Войти'),
          ),
        ),
        const SizedBox(height: 16),
        Text(
          'Данные для демонстрационного входа печатаются в консоли '
          'Go-сервера при запуске.',
          textAlign: TextAlign.center,
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }
}

class _RoleCard extends StatelessWidget {
  const _RoleCard({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.onTap,
  });

  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return Card(
      elevation: 0,
      color: Theme.of(context).colorScheme.surfaceContainerHighest,
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.all(18),
          child: Row(
            children: [
              Icon(icon, size: 30),
              const SizedBox(width: 16),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: const TextStyle(
                        fontSize: 16,
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(subtitle),
                  ],
                ),
              ),
              const Icon(Icons.chevron_right),
            ],
          ),
        ),
      ),
    );
  }
}

class ConsentScreen extends StatefulWidget {
  const ConsentScreen({
    super.key,
    required this.api,
    required this.onAccepted,
    required this.onLogout,
  });

  final ApiClient api;
  final VoidCallback onAccepted;
  final VoidCallback onLogout;

  @override
  State<ConsentScreen> createState() => _ConsentScreenState();
}

class _ConsentScreenState extends State<ConsentScreen> {
  bool _checked = false;
  bool _busy = false;
  String? _error;

  Future<void> _accept() async {
    if (!_checked) {
      return;
    }
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      await widget.api.acceptConsent();
      if (mounted) {
        widget.onAccepted();
      }
    } on APIException catch (error) {
      if (mounted) {
        setState(() => _error = error.message);
      }
    } catch (error) {
      if (mounted) {
        setState(() => _error = 'Не удалось сохранить согласие: $error');
      }
    } finally {
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: const Text('Согласие с политиками'),
        actions: [
          TextButton(onPressed: widget.onLogout, child: const Text('Выйти')),
        ],
      ),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 640),
            child: Card(
              child: Padding(
                padding: const EdgeInsets.all(26),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Перед началом работы',
                      style: theme.textTheme.headlineSmall?.copyWith(
                        fontWeight: FontWeight.w700,
                      ),
                    ),
                    const SizedBox(height: 12),
                    const Text(
                      'Приложение помогает вести наблюдение и поддерживать '
                      'связь с врачом. Оно не ставит диагноз, не оценивает '
                      'экстренность состояния и не заменяет консультацию '
                      'специалиста.',
                    ),
                    const SizedBox(height: 12),
                    const Text(
                      'Персональные данные и записи доступны только вам и '
                      'вашему лечащему врачу. Светофор состояния — это '
                      'вспомогательный индикатор, а не медицинское заключение.',
                    ),
                    const SizedBox(height: 12),
                    const Text(
                      'При ухудшении самочувствия или мыслях о причинении '
                      'вреда себе обратитесь за экстренной помощью.',
                    ),
                    const SizedBox(height: 18),
                    CheckboxListTile(
                      contentPadding: EdgeInsets.zero,
                      value: _checked,
                      onChanged: (value) =>
                          setState(() => _checked = value ?? false),
                      title: const Text(
                        'Я принимаю условия политики конфиденциальности '
                        'и обработки данных (версия v1)',
                      ),
                    ),
                    if (_error != null)
                      Text(
                        _error!,
                        style: TextStyle(color: theme.colorScheme.error),
                      ),
                    const SizedBox(height: 12),
                    SizedBox(
                      width: double.infinity,
                      child: FilledButton(
                        onPressed: !_checked || _busy ? null : _accept,
                        child: Padding(
                          padding: const EdgeInsets.symmetric(vertical: 12),
                          child: _busy
                              ? const SizedBox(
                                  width: 20,
                                  height: 20,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                )
                              : const Text('Принимаю и продолжить'),
                        ),
                      ),
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}

class AnamnesisScreen extends StatefulWidget {
  const AnamnesisScreen({
    super.key,
    required this.api,
    required this.onSaved,
    required this.onSkip,
    required this.onLogout,
  });

  final ApiClient api;
  final ValueChanged<String> onSaved;
  final VoidCallback onSkip;
  final VoidCallback onLogout;

  @override
  State<AnamnesisScreen> createState() => _AnamnesisScreenState();
}

class _AnamnesisScreenState extends State<AnamnesisScreen> {
  final _complaints = TextEditingController();
  final _duration = TextEditingController();
  final _sleep = TextEditingController();
  final _medications = TextEditingController();
  bool _busy = false;
  String? _error;

  @override
  void dispose() {
    _complaints.dispose();
    _duration.dispose();
    _sleep.dispose();
    _medications.dispose();
    super.dispose();
  }

  Future<void> _save() async {
    final parts = <String>[
      if (_complaints.text.trim().isNotEmpty)
        'Жалобы: ${_complaints.text.trim()}',
      if (_duration.text.trim().isNotEmpty)
        'Длительность: ${_duration.text.trim()}',
      if (_sleep.text.trim().isNotEmpty) 'Сон: ${_sleep.text.trim()}',
      if (_medications.text.trim().isNotEmpty)
        'Препараты: ${_medications.text.trim()}',
    ];
    if (parts.isEmpty) {
      setState(() => _error = 'Заполните хотя бы одно поле');
      return;
    }

    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final notes = await widget.api.saveAnamnesis(parts.join('\n'));
      if (mounted) {
        widget.onSaved(notes);
      }
    } on APIException catch (error) {
      if (mounted) {
        setState(() => _error = error.message);
      }
    } catch (error) {
      if (mounted) {
        setState(() => _error = 'Не удалось сохранить анамнез: $error');
      }
    } finally {
      if (mounted) {
        setState(() => _busy = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Первичный сбор анамнеза'),
        actions: [
          TextButton(onPressed: widget.onLogout, child: const Text('Выйти')),
        ],
      ),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: ConstrainedBox(
            constraints: const BoxConstraints(maxWidth: 640),
            child: Card(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      'Расскажите о себе',
                      style: Theme.of(context).textTheme.headlineSmall
                          ?.copyWith(fontWeight: FontWeight.w700),
                    ),
                    const SizedBox(height: 6),
                    const Text(
                      'Эти сведения увидит ваш врач. Их можно дополнить позже.',
                    ),
                    const SizedBox(height: 18),
                    TextField(
                      controller: _complaints,
                      minLines: 3,
                      maxLines: 6,
                      decoration: const InputDecoration(
                        labelText: 'Что вас беспокоит сейчас?',
                        alignLabelWithHint: true,
                      ),
                    ),
                    const SizedBox(height: 12),
                    TextField(
                      controller: _duration,
                      decoration: const InputDecoration(
                        labelText: 'Как давно это продолжается?',
                      ),
                    ),
                    const SizedBox(height: 12),
                    TextField(
                      controller: _sleep,
                      decoration: const InputDecoration(
                        labelText: 'Как вы спите?',
                      ),
                    ),
                    const SizedBox(height: 12),
                    TextField(
                      controller: _medications,
                      decoration: const InputDecoration(
                        labelText: 'Принимаете ли вы препараты?',
                      ),
                    ),
                    if (_error != null) ...[
                      const SizedBox(height: 12),
                      Text(
                        _error!,
                        style: TextStyle(
                          color: Theme.of(context).colorScheme.error,
                        ),
                      ),
                    ],
                    const SizedBox(height: 18),
                    Row(
                      children: [
                        TextButton(
                          onPressed: _busy ? null : widget.onSkip,
                          child: const Text('Заполнить позже'),
                        ),
                        const Spacer(),
                        FilledButton(
                          onPressed: _busy ? null : _save,
                          child: _busy
                              ? const SizedBox(
                                  width: 20,
                                  height: 20,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                )
                              : const Text('Сохранить и продолжить'),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ),
          ),
        ),
      ),
    );
  }
}
