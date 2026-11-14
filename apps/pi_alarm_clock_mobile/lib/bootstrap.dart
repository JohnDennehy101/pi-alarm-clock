import 'package:flutter/material.dart';
import 'package:kiwi/kiwi.dart';
import 'package:pi_alarm_clock_mobile/app/app.dart';
import 'package:pi_alarm_clock_mobile/app/startup/startup_service.dart';
import 'package:pi_alarm_clock_mobile/di.dart';
import 'package:pi_alarm_clock_mobile/features/pairing/pairing_service.dart';

Widget bootstrap() {
  setupDependencies();

  final container = KiwiContainer();
  final startupService = container.resolve<StartupService>();
  final pairingService = container.resolve<PairingService>();

  return PiAlarmClockApp(
    startupService: startupService,
    pairingService: pairingService,
  );
}
