import 'package:flutter/material.dart';
import 'package:pi_alarm_clock_mobile/services/api/server_not_verified_exception.dart';
import 'package:pi_alarm_clock_mobile/services/device_discovery/device_discovery_service.dart';
import 'package:pi_alarm_clock_mobile/widgets/verify_server_dialog.dart';

import '../services/api/api_client.dart';
import 'pairing_screen.dart';
import 'home_screen.dart';

class StartupGate extends StatefulWidget {
  const StartupGate({super.key});

  @override
  State<StartupGate> createState() => _StartupGateState();
}

class _StartupGateState extends State<StartupGate> {
  bool _isLoading = true;
  String? _errorMessage;

  @override
  void initState() {
    super.initState();
    _checkAuthAndNetwork();
  }

  Future<void> _checkAuthAndNetwork() async {
    try {
      final discovery = DeviceDiscoveryService();
      final devices = await discovery.discover();

      if (!mounted) return;

      if (devices.isEmpty) {
        setState(() {
          _isLoading = false;
          _errorMessage = 'Could not find your Raspberry Pi on the network.';
        });
        return;
      }

      ApiClient.setServerBaseUrlFromDevice(devices.first);

      ApiClient.fingerprintVerifier =
          (fingerprint, {required identityChanged}) => showVerifyServerDialog(
            context,
            fingerprint: fingerprint,
            identityChanged: identityChanged,
          );

      final token = await ApiClient.storage.read(key: 'auth_token');

      if (!mounted) return;

      if (token == null) {
        _navigateTo(const PairingScreen());
        return;
      }

      final response = await ApiClient.get('/api/v1/health');

      if (!mounted) return;

      if (response.statusCode == 200) {
        _navigateTo(const HomeScreen());
      } else if (response.statusCode == 401) {
        await ApiClient.storage.delete(key: 'auth_token');

        if (!mounted) return;

        _navigateTo(const PairingScreen());
      } else {
        setState(() {
          _isLoading = false;
          _errorMessage =
              'Unexpected server response (${response.statusCode}).';
        });
      }
    } on ServerNotVerifiedException catch (e) {
      if (!mounted) return;

      setState(() {
        _isLoading = false;
        _errorMessage =
            '${e.message}\n\nCheck the Raspberry Pi is on your '
            'network, verify its certificate fingerprint, then retry.';
      });
    } catch (e) {
      if (!mounted) return;

      setState(() {
        _isLoading = false;
        _errorMessage =
            'Could not connect to Pi alarm clock. Ensure you are on home Wi-Fi.\n(${e.toString()})';
      });
    }
  }

  void _navigateTo(Widget screen) {
    if (!mounted) return;

    Navigator.of(context)
        .pushReplacement(MaterialPageRoute(builder: (_) => screen));
  }

  void _navigateToPairing() {
    _navigateTo(const PairingScreen());
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(24.0),
          child: _isLoading && _errorMessage == null
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
                      onPressed: () {
                        setState(() {
                          _isLoading = true;
                          _errorMessage = null;
                        });

                        _checkAuthAndNetwork();
                      },
                      child: const Text('Retry Connection'),
                    ),
                    TextButton(
                      onPressed: _navigateToPairing,
                      child: const Text('Enter Pairing Mode Manually'),
                    ),
                  ],
                ),
        ),
      ),
    );
  }
}
