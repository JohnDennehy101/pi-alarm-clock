import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:pi_alarm_clock_mobile/screens/home_screen.dart';
import 'package:pi_alarm_clock_mobile/services/api/server_not_verified_exception.dart';
import 'package:pi_alarm_clock_mobile/widgets/verify_server_dialog.dart';

import '../services/api/api_client.dart';

class PairingScreen extends StatefulWidget {
  const PairingScreen({super.key});

  static void navigateHereFromGate() {}

  @override
  State<PairingScreen> createState() => _PairingScreenState();
}

class _PairingScreenState extends State<PairingScreen> {
  final _pinController = TextEditingController();
  final _nameController = TextEditingController(text: 'My Phone');
  bool _isPairing = false;
  String? _error;

  @override
  void dispose() {
    _pinController.dispose();
    _nameController.dispose();
    super.dispose();
  }

  Future<void> _pairWithPi() async {
    if (_pinController.text.trim().length != 6) {
      setState(() => _error = 'Please enter the full 6-digit pairing PIN.');
      return;
    }

    setState(() {
      _isPairing = true;
      _error = null;
    });

    ApiClient.fingerprintVerifier = (fingerprint, {required identityChanged}) =>
        showVerifyServerDialog(
          context,
          fingerprint: fingerprint,
          identityChanged: identityChanged,
        );

    try {
      final response = await ApiClient.post('/api/v1/auth/pair', {
        'pin': _pinController.text.trim(),
        'deviceName': _nameController.text.trim(),
      });

      if (response.statusCode == 201) {
        final data = jsonDecode(response.body);
        final token = data['token'];

        await ApiClient.storage.write(key: 'auth_token', value: token);

        if (!mounted) return;
        Navigator.of(context).pushReplacement(
          MaterialPageRoute(builder: (_) => const HomeScreen()),
        );
      } else if (response.statusCode == 429) {
        setState(
          () => _error = 'Too many failed attempts. Locked out for 15 minutes.',
        );
      } else {
        setState(
          () => _error =
              'Invalid PIN or connection error (${response.statusCode})',
        );
      }
    } on ServerNotVerifiedException catch (e) {
      setState(() => _error = e.message);
    } catch (e) {
      setState(() => _error = 'Failed to reach server: $e');
    } finally {
      if (mounted) {
        setState(() => _isPairing = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Pair with Raspberry Pi')),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24.0),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Icon(
                Icons.devices_other,
                size: 64,
                color: Colors.deepPurple,
              ),
              const SizedBox(height: 16),
              const Text(
                'Enter Pairing PIN',
                textAlign: TextAlign.center,
                style: TextStyle(fontSize: 22, fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 8),
              const Text(
                'Check your Raspberry Pi startup logs for the active 6-digit pairing PIN.',
                textAlign: TextAlign.center,
                style: TextStyle(color: Colors.grey, fontSize: 14),
              ),
              const SizedBox(height: 32),

              TextField(
                controller: _pinController,
                keyboardType: TextInputType.number,
                maxLength: 6,
                decoration: const InputDecoration(
                  labelText: '6-Digit PIN',
                  border: OutlineInputBorder(),
                  prefixIcon: Icon(Icons.lock_outline),
                  counterText: '',
                ),
              ),
              const SizedBox(height: 16),

              TextField(
                controller: _nameController,
                decoration: const InputDecoration(
                  labelText: 'Device Name',
                  border: OutlineInputBorder(),
                  prefixIcon: Icon(Icons.phone_android),
                ),
              ),

              if (_error != null) ...[
                const SizedBox(height: 16),
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: Colors.red.shade50,
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: Colors.red.shade200),
                  ),
                  child: Text(
                    _error!,
                    style: TextStyle(color: Colors.red.shade800),
                    textAlign: TextAlign.center,
                  ),
                ),
              ],

              const SizedBox(height: 24),

              ElevatedButton(
                onPressed: _isPairing ? null : _pairWithPi,
                style: ElevatedButton.styleFrom(
                  padding: const EdgeInsets.symmetric(vertical: 16),
                  backgroundColor: Colors.deepPurple,
                  foregroundColor: Colors.white,
                ),
                child: _isPairing
                    ? const SizedBox(
                        height: 20,
                        width: 20,
                        child: CircularProgressIndicator(
                          strokeWidth: 2,
                          color: Colors.white,
                        ),
                      )
                    : const Text('Pair Device', style: TextStyle(fontSize: 16)),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
