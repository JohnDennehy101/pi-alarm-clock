import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:pi_alarm_clock_mobile/app/startup/startup_gate.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/application/alarm_controller.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/application/alarm_editor_controller.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/presentation/alarms_form_screen.dart';
import 'package:provider/provider.dart';

import 'package:pi_alarm_clock_mobile/features/alarms/presentation/alarms_screen.dart';
import 'package:pi_alarm_clock_mobile/features/home/presentation/home_screen.dart';
import 'package:pi_alarm_clock_mobile/features/settings/presentation/settings_screen.dart';

import '../../features/pairing/application/pairing_controller.dart';
import '../../features/pairing/pairing_service.dart';
import '../../features/pairing/presentation/pairing_screen.dart';

import 'app_navigator.dart';
import 'main_scaffold.dart';

final GoRouter appRouter = GoRouter(
  navigatorKey: AppNavigator.key,
  initialLocation: '/startup',
  routes: [
    GoRoute(path: '/startup', builder: (context, state) => const StartupGate()),
    GoRoute(
      path: '/pairing',
      builder: (context, state) {
        return ChangeNotifierProvider(
          create: (context) =>
              PairingController(context.read<PairingService>()),
          child: const PairingScreen(),
        );
      },
    ),

    StatefulShellRoute.indexedStack(
      builder: (context, state, navigationShell) {
        return MainScaffold(navigationShell: navigationShell);
      },
      branches: [
        StatefulShellBranch(
          routes: [
            GoRoute(path: '/', builder: (context, state) => const HomeScreen()),
          ],
        ),
        StatefulShellBranch(
          routes: [
            GoRoute(
              path: '/alarms',
              builder: (context, state) => const AlarmsScreen(),
              routes: [
                GoRoute(
                  path: 'new',
                  builder: (context, state) => _buildAlarmForm(context),
                ),
                GoRoute(
                  path: ':id/edit',
                  builder: (context, state) => _buildAlarmForm(
                    context,
                    alarmId: state.pathParameters['id']!,
                  ),
                ),
              ],
            ),
          ],
        ),

        StatefulShellBranch(
          routes: [
            GoRoute(
              path: '/settings',
              builder: (context, state) => const SettingsScreen(),
            ),
          ],
        ),
      ],
    ),
  ],
);

Widget _buildAlarmForm(BuildContext context, {String? alarmId}) {
  return ChangeNotifierProvider(
    create: (context) =>
        AlarmEditorController(context.read<AlarmController>())..load(alarmId),
    child: AlarmFormScreen(alarmId: alarmId),
  );
}
