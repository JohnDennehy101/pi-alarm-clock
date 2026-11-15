import 'package:go_router/go_router.dart';
import 'package:pi_alarm_clock_mobile/app/home/home_screen.dart';

import 'package:pi_alarm_clock_mobile/app/navigation/app_navigator.dart';
import 'package:pi_alarm_clock_mobile/app/startup/startup_gate.dart';
import 'package:pi_alarm_clock_mobile/features/pairing/presentation/pairing_screen.dart';

final GoRouter appRouter = GoRouter(
  navigatorKey: AppNavigator.key,
  initialLocation: '/',
  routes: [
    GoRoute(path: '/', builder: (context, state) => const StartupGate()),
    GoRoute(path: '/home', builder: (context, state) => const HomeScreen()),
    GoRoute(
      path: '/pairing',
      builder: (context, state) => const PairingScreen(),
    ),
  ],
);
