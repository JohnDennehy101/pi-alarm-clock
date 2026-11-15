import 'package:flutter/material.dart';
import 'package:pi_alarm_clock_mobile/app/navigation/app_router.dart';

class PiAlarmClockApp extends StatelessWidget {
  const PiAlarmClockApp({super.key});

  @override
  Widget build(BuildContext context) {
    return MaterialApp.router(
      title: 'Pi Alarm Clock',
      routerConfig: appRouter,
      theme: ThemeData(
        useMaterial3: true,
        colorScheme: ColorScheme.fromSeed(
          seedColor: const Color(0xFF456B85),
          brightness: Brightness.light,
        ),
      ),
      debugShowCheckedModeBanner: false,
    );
  }
}
