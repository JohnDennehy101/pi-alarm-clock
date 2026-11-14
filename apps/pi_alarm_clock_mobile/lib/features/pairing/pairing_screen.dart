import 'package:flutter/material.dart';

import 'package:pi_alarm_clock_mobile/features/home/home_screen.dart';
import 'package:pi_alarm_clock_mobile/features/pairing/infra/pairing_api_service.dart';
import 'package:pi_alarm_clock_mobile/features/pairing/pairing_service.dart';
import 'package:pi_alarm_clock_mobile/services/api/server_not_verified_exception.dart';

class PairingScreen extends StatefulWidget {
  const PairingScreen({super.key});

  @override
  State<PairingScreen> createState() => _PairingScreenState();
}

class _PairingScreenState extends State<PairingScreen> {
  final _pinController = TextEditingController();
  final _nameController = TextEditingController(text: 'My Phone');

  late final PairingService _pairingService;

  bool _isPairing = false;
  String? _error;

  @override
  void initState() {
    super.initState();

    _pairingService = PairingService(api: PairingApiService());
  }

  @override
  void dispose() {
    _pinController.dispose();
    _nameController.dispose();
    super.dispose();
  }

  Future<void> _pairWithPi() async {
    final pin = _pinController.text.trim();
    final deviceName = _nameController.text.trim();

    if (!RegExp(r'^\d{6}$').hasMatch(pin)) {
      setState(() {
        _error = 'Please enter the full 6-digit pairing PIN.';
      });
      return;
    }

    if (deviceName.isEmpty || deviceName.length > 64) {
      setState(() {
        _error = 'Device name must be between 1 and 64 characters.';
      });
      return;
    }

    setState(() {
      _isPairing = true;
      _error = null;
    });

    try {
      final result = await _pairingService.pair(
        pin: pin,
        deviceName: deviceName,
      );

      if (!mounted) return;

      switch (result) {
        case PairingResult.success:
          _navigateToHome();
        case PairingResult.tooManyAttempts:
          _showError('Too many failed attempts. Locked out for 15 minutes.');
        case PairingResult.invalidPin:
          _showError('Invalid pairing PIN.');
        case PairingResult.unexpectedResponse:
          _showError('Unexpected response from Raspberry Pi.');
      }
    } on ServerNotVerifiedException catch (e) {
      _showError(e.message);
    } catch (e) {
      _showError('Failed to reach server: $e');
    } finally {
      if (mounted) {
        setState(() {
          _isPairing = false;
        });
      }
    }
  }

  void _navigateToHome() {
    Navigator.of(context)
        .pushReplacement(MaterialPageRoute(builder: (_) => const HomeScreen()));
  }

  void _showError(String message) {
    if (!mounted) return;

    setState(() {
      _error = message;
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Pair with Raspberry Pi')),
      body: Center(
        child: SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: Column(
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
                'Check your Raspberry Pi startup logs for the active '
                '6-digit pairing PIN.',
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
                maxLength: 64,
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
