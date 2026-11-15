import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';

import 'package:pi_alarm_clock_mobile/features/pairing/application/pairing_controller.dart';
import 'package:pi_alarm_clock_mobile/features/pairing/presentation/widgets/pin_input.dart';

class PairingScreen extends StatelessWidget {
  const PairingScreen({super.key});

  Future<void> _pairWithPi(
    BuildContext context,
    PairingController controller,
  ) async {
    final success = await controller.pair();

    if (!context.mounted || !success) return;

    context.go('/home');
  }

  @override
  Widget build(BuildContext context) {
    final controller = context.watch<PairingController>();

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

              PinInput(
                length: 6,
                enabled: !controller.isPairing,
                onChanged: controller.updatePin,
              ),

              const SizedBox(height: 24),

              TextField(
                enabled: !controller.isPairing,
                maxLength: 64,
                onChanged: controller.updateDeviceName,
                decoration: const InputDecoration(
                  labelText: 'Device Name',
                  border: OutlineInputBorder(),
                  prefixIcon: Icon(Icons.phone_android),
                ),
              ),

              if (controller.error != null) ...[
                const SizedBox(height: 16),
                Container(
                  padding: const EdgeInsets.all(12),
                  decoration: BoxDecoration(
                    color: Colors.red.shade50,
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: Colors.red.shade200),
                  ),
                  child: Text(
                    controller.error!,
                    style: TextStyle(color: Colors.red.shade800),
                    textAlign: TextAlign.center,
                  ),
                ),
              ],

              const SizedBox(height: 24),

              ElevatedButton(
                onPressed: controller.canPair
                    ? () => _pairWithPi(context, controller)
                    : null,
                style: ElevatedButton.styleFrom(
                  padding: const EdgeInsets.symmetric(vertical: 16),
                  backgroundColor: Colors.deepPurple,
                  foregroundColor: Colors.white,
                ),
                child: controller.isPairing
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
