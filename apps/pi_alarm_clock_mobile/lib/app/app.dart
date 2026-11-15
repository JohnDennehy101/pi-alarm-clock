import 'package:flutter/material.dart';
import 'package:pi_alarm_clock_mobile/app/navigation/app_router.dart';

class PiAlarmClockApp extends StatelessWidget {
  const PiAlarmClockApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'Pi Alarm Clock',
      routerConfig: appRouter,
      debugShowCheckedModeBanner: false,
    );
  }
}
