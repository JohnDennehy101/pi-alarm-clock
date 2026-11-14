import 'package:flutter/material.dart';
import 'package:pi_alarm_clock_mobile/app/navigation/app_navigator.dart';

import 'startup/startup_gate.dart';

class PiAlarmClockApp extends StatelessWidget {
  const PiAlarmClockApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Pi Alarm Clock',
      navigatorKey: AppNavigator.key,
      debugShowCheckedModeBanner: false,
      home: const StartupGate(),
    );
  }
}
