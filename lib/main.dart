import 'dart:io';

import 'package:flutter/material.dart';

import 'data/api/api_client.dart';
import 'data/api/models.dart';
import 'ui/doctor_app.dart';
import 'ui/login_screen.dart';
import 'ui/patient_app.dart';

void main() {
  runApp(CareApp(api: ApiClient(baseUrl: _apiBaseUrl())));
}

String _apiBaseUrl() {
  const configured = String.fromEnvironment('API_BASE_URL');
  if (configured.isNotEmpty) {
    return configured;
  }
  // Android-эмулятор обращается к localhost компьютера через 10.0.2.2.
  if (Platform.isAndroid) {
    return 'http://10.0.2.2:8080';
  }
  return 'http://localhost:8080';
}

class CareApp extends StatefulWidget {
  const CareApp({super.key, required this.api});

  final ApiClient api;

  @override
  State<CareApp> createState() => _CareAppState();
}

class _CareAppState extends State<CareApp> {
  AuthSession? _session;
  bool _skipAnamnesis = false;

  void _logout() {
    widget.api.setToken(null);
    setState(() {
      _session = null;
      _skipAnamnesis = false;
    });
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Сопровождение пациента',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xFF39796B)),
        useMaterial3: true,
        scaffoldBackgroundColor: const Color(0xFFF6F8F7),
        inputDecorationTheme: const InputDecorationTheme(
          border: OutlineInputBorder(),
        ),
      ),
      home: _buildHome(),
    );
  }

  Widget _buildHome() {
    final session = _session;
    if (session == null) {
      return LoginScreen(
        api: widget.api,
        onLoggedIn: (loggedIn) {
          setState(() {
            _session = loggedIn;
            _skipAnamnesis = false;
          });
        },
      );
    }

    if (session.isDoctor) {
      return DoctorApp(api: widget.api, session: session, onLogout: _logout);
    }

    if (!session.consentAccepted) {
      return ConsentScreen(
        api: widget.api,
        onAccepted: () {
          setState(() {
            _session = session.copyWith(consentAccepted: true);
          });
        },
        onLogout: _logout,
      );
    }

    if (session.anamnesis.trim().isEmpty && !_skipAnamnesis) {
      return AnamnesisScreen(
        api: widget.api,
        onSaved: (notes) {
          setState(() {
            _session = session.copyWith(anamnesis: notes);
          });
        },
        onSkip: () => setState(() => _skipAnamnesis = true),
        onLogout: _logout,
      );
    }

    return PatientApp(api: widget.api, session: session, onLogout: _logout);
  }
}
