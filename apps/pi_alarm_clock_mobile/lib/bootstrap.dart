import 'package:flutter/material.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/application/alarm_controller.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/application/device_options_controller.dart';
import 'package:pi_alarm_clock_mobile/features/home/application/home_controller.dart';
import 'package:pi_alarm_clock_mobile/features/pairing/application/pairing_controller.dart';
import 'package:provider/provider.dart';
import 'package:pi_alarm_clock_mobile/app/app.dart';
import 'package:pi_alarm_clock_mobile/app/startup/startup_service.dart';
import 'package:pi_alarm_clock_mobile/di.dart';
import 'package:pi_alarm_clock_mobile/features/pairing/pairing_service.dart';

Widget bootstrap() {
  setupDependencies();

  final startupService = resolve<StartupService>();
  final pairingService = resolve<PairingService>();

  return MultiProvider(
    providers: [
      Provider<StartupService>.value(value: startupService),
      Provider<PairingService>.value(value: pairingService),
      ChangeNotifierProvider<PairingController>(
        create: (_) => PairingController(pairingService),
      ),
      ChangeNotifierProvider(create: (_) => resolve<HomeController>()..load()),
      ChangeNotifierProvider(create: (_) => resolve<AlarmController>()..load()),
      ChangeNotifierProvider(create: (_) => resolve<DeviceOptionsController>()),
    ],
    child: const PiAlarmClockApp(),
  );
}
