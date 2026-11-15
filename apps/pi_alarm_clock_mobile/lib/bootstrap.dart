import 'package:flutter/material.dart';
import 'package:kiwi/kiwi.dart';
import 'package:pi_alarm_clock_mobile/features/pairing/application/pairing_controller.dart';
import 'package:provider/provider.dart';
import 'package:pi_alarm_clock_mobile/app/app.dart';
import 'package:pi_alarm_clock_mobile/app/startup/startup_service.dart';
import 'package:pi_alarm_clock_mobile/di.dart';
import 'package:pi_alarm_clock_mobile/features/pairing/pairing_service.dart';

Widget bootstrap() {
  setupDependencies();

  final container = KiwiContainer();
  final startupService = container.resolve<StartupService>();
  final pairingService = container.resolve<PairingService>();

  return MultiProvider(
    providers: [
      Provider<StartupService>.value(value: startupService),
      Provider<PairingService>.value(value: pairingService),
      ChangeNotifierProvider<PairingController>(
        create: (_) => PairingController(pairingService),
      ),
    ],
    child: const PiAlarmClockApp(),
  );
}
