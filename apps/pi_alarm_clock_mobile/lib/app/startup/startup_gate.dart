import 'package:flutter/material.dart';

import 'package:pi_alarm_clock_mobile/app/startup/startup_destination.dart';
import 'package:pi_alarm_clock_mobile/app/startup/startup_service.dart';
import 'package:pi_alarm_clock_mobile/features/pairing/pairing_service.dart';
import 'package:pi_alarm_clock_mobile/services/api/server_not_verified_exception.dart';

import '../../features/home/home_screen.dart';
import '../../features/pairing/pairing_screen.dart';

class StartupGate extends StatefulWidget {
  const StartupGate({
    super.key,
    required this.startupService,
    required this.pairingService,
  });

  final StartupService startupService;
  final PairingService pairingService;

  @override
  State<StartupGate> createState() => _StartupGateState();
}

class _StartupGateState extends State<StartupGate> {
  late final StartupService _startupService;
  late final PairingService _pairingService;

  bool _isLoading = true;
  String? _errorMessage;

  @override
  void initState() {
    super.initState();
    _startupService = widget.startupService;
    _pairingService = widget.pairingService;
    _start();
  }

  Future<void> _start() async {
    try {
      final destination = await _startupService.initialise();

      if (!mounted) return;

      _navigateTo(destination);
    } on ServerNotVerifiedException catch (e) {
      _showError(
        '${e.message}\n\n'
        'Check your Raspberry Pi connection and verify its certificate fingerprint.',
      );
    } catch (e) {
      _showError(e.toString());
    }
  }

  void _navigateTo(StartupDestination destination) {
    final Widget screen = switch (destination) {
      StartupDestination.pairing => PairingScreen(
        pairingService: _pairingService,
        startupService: _startupService,
      ),
      StartupDestination.home => HomeScreen(
        startupService: _startupService,
        pairingService: _pairingService,
      ),
    };

    Navigator.of(context)
        .pushReplacement(MaterialPageRoute(builder: (_) => screen));
  }

  void _showError(String message) {
    if (!mounted) return;

    setState(() {
      _isLoading = false;
      _errorMessage = message;
    });
  }

  void _retry() {
    setState(() {
      _isLoading = true;
      _errorMessage = null;
    });

    _start();
  }

  void _enterPairingMode() {
    Navigator.of(context).pushReplacement(
      MaterialPageRoute(
        builder: (_) => PairingScreen(
          pairingService: _pairingService,
          startupService: _startupService,
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: _isLoading
              ? const Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    CircularProgressIndicator(),
                    SizedBox(height: 16),
                    Text('Connecting to Raspberry Pi...'),
                  ],
                )
              : Column(
                  mainAxisAlignment: MainAxisAlignment.center,
                  children: [
                    const Icon(Icons.wifi_off, size: 64, color: Colors.orange),
                    const SizedBox(height: 16),
                    Text(
                      _errorMessage ?? 'Connection error',
                      textAlign: TextAlign.center,
                      style: const TextStyle(fontSize: 16),
                    ),
                    const SizedBox(height: 24),
                    ElevatedButton(
                      onPressed: _retry,
                      child: const Text('Retry Connection'),
                    ),
                    TextButton(
                      onPressed: _enterPairingMode,
                      child: const Text('Enter Pairing Mode Manually'),
                    ),
                  ],
                ),
        ),
      ),
    );
  }
}
