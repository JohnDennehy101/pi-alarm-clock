import 'package:flutter/material.dart';

import '../../app/startup/startup_gate.dart';
import '../../services/api/api_client.dart';

class HomeScreen extends StatelessWidget {
  const HomeScreen({super.key});

  Future<void> _logout(BuildContext context) async {
    final confirmed = await _confirmLogout(context);

    if (confirmed != true || !context.mounted) return;

    try {
      await ApiClient.logout();
    } catch (e) {
      if (!context.mounted) return;

      _showMessage(context, 'Logout failed: $e');
      return;
    }

    if (!context.mounted) return;

    _returnToStartup(context);
  }

  Future<bool?> _confirmLogout(BuildContext context) {
    return showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: const Text('Unpair device?'),
        content: const Text(
          'This will disconnect the app from your Raspberry Pi. '
          'You will need to pair it again to reconnect.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () => Navigator.of(context).pop(true),
            child: const Text('Unpair'),
          ),
        ],
      ),
    );
  }

  void _returnToStartup(BuildContext context) {
    Navigator.of(
      context,
    ).pushReplacement(MaterialPageRoute(builder: (_) => const StartupGate()));
  }

  void _showMessage(BuildContext context, String message) {
    ScaffoldMessenger.of(context)
        .showSnackBar(SnackBar(content: Text(message)));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Pi Alarm Clock'),
        actions: [
          IconButton(
            icon: const Icon(Icons.logout),
            tooltip: 'Unpair / Logout',
            onPressed: () => _logout(context),
          ),
        ],
      ),
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Icon(Icons.alarm, size: 80, color: Colors.deepPurple),
              const SizedBox(height: 24),
              const Text(
                'Connected & Authenticated!',
                style: TextStyle(fontSize: 22, fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 12),
              const Text(
                'Your Flutter app is securely linked to your Raspberry Pi backend.',
                textAlign: TextAlign.center,
                style: TextStyle(color: Colors.grey),
              ),
              const SizedBox(height: 32),
              ElevatedButton.icon(
                onPressed: () =>
                    _showMessage(context, 'Connected to Pi successfully!'),
                icon: const Icon(Icons.refresh),
                label: const Text('Test API Connection'),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
