import 'package:flutter/material.dart';
import 'package:pi_alarm_clock_mobile/app/navigation/app_navigator.dart';
import 'package:pi_alarm_clock_mobile/features/pairing/pairing_service.dart';

import 'startup/startup_gate.dart';

import 'package:pi_alarm_clock_mobile/app/startup/startup_service.dart';

class PiAlarmClockApp extends StatelessWidget {
  const PiAlarmClockApp({
    super.key,
    required this.startupService,
    required this.pairingService,
  });

  final StartupService startupService;
  final PairingService pairingService;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'Pi Alarm Clock',
      navigatorKey: AppNavigator.key,
      debugShowCheckedModeBanner: false,
      home: StartupGate(
        startupService: startupService,
        pairingService: pairingService,
      ),
    );
  }
}
