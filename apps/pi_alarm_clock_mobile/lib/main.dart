import 'package:flutter/material.dart';

import 'screens/startup_gate.dart';

void main() {
  runApp(const PiAlarmApp());
}

class PiAlarmApp extends StatelessWidget {
  const PiAlarmApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Pi Alarm Clock',
      debugShowCheckedModeBanner: false,
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: Colors.deepPurple),
        useMaterial3: true,
      ),
      home: const StartupGate(),
    );
  }
}
