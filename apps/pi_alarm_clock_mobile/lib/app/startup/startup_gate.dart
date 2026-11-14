import 'package:flutter/material.dart';

import 'package:pi_alarm_clock_mobile/app/navigation/app_navigator.dart';
import 'package:pi_alarm_clock_mobile/app/startup/startup_destination.dart';
import 'package:pi_alarm_clock_mobile/app/startup/startup_service.dart';
import 'package:pi_alarm_clock_mobile/app/startup/infra/startup_api_service.dart';
import 'package:pi_alarm_clock_mobile/app/startup/infra/startup_device_discovery_service.dart';
import 'package:pi_alarm_clock_mobile/services/api/api_client.dart';
import 'package:pi_alarm_clock_mobile/services/api/server_not_verified_exception.dart';
import 'package:pi_alarm_clock_mobile/services/device_discovery/device_discovery_service.dart';
import 'package:pi_alarm_clock_mobile/widgets/verify_server_dialog.dart';

import '../../features/home/home_screen.dart';
import '../../features/pairing/pairing_screen.dart';

class StartupGate extends StatefulWidget {
  const StartupGate({super.key});

  @override
  State<StartupGate> createState() => _StartupGateState();
}

class _StartupGateState extends State<StartupGate> {
  late final StartupService _startupService;

  bool _isLoading = true;
  String? _errorMessage;

  @override
  void initState() {
    super.initState();

    _startupService = StartupService(
      discovery: StartupDeviceDiscoveryService(DeviceDiscoveryService()),
      api: StartupApiService(),
    );

    _configureCertificateVerification();

    _start();
  }

  void _configureCertificateVerification() {
    ApiClient.fingerprintVerifier =
        (fingerprint, {required identityChanged}) async {
          final context = AppNavigator.context;

          if (context == null) {
            return false;
          }

          return showVerifyServerDialog(
            context,
            fingerprint: fingerprint,
            identityChanged: identityChanged,
          );
        };
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
      StartupDestination.pairing => const PairingScreen(),
      StartupDestination.home => const HomeScreen(),
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
    Navigator.of(
      context,
    ).pushReplacement(MaterialPageRoute(builder: (_) => const PairingScreen()));
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
