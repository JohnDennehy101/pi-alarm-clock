import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:pi_alarm_clock_mobile/widgets/shared/page_header.dart';
import 'package:provider/provider.dart';

import 'package:pi_alarm_clock_mobile/features/home/application/home_controller.dart';
import 'package:pi_alarm_clock_mobile/features/home/presentation/widgets/clock_text.dart';
import 'package:pi_alarm_clock_mobile/features/home/presentation/widgets/device_status_section.dart';
import 'package:pi_alarm_clock_mobile/features/home/presentation/widgets/next_alarm_card.dart';

class HomeScreen extends StatelessWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final controller = context.watch<HomeController>();
    final theme = Theme.of(context);

    return RefreshIndicator(
      onRefresh: controller.load,
      child: SingleChildScrollView(
        physics: const AlwaysScrollableScrollPhysics(),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            PageHeader(
              title: 'Overview',
              subtitle: MaterialLocalizations.of(context)
                  .formatFullDate(DateTime.now()),
              trailing: ClockText(
                style: theme.textTheme.headlineMedium?.copyWith(
                  fontWeight: FontWeight.w300,
                ),
              ),
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 8, 20, 16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  NextAlarmCard(onViewAlarms: () => context.go('/alarms')),
                  const SizedBox(height: 24),
                  DeviceStatusSection(
                    isLoading: controller.isLoading,
                    healthStatus: controller.healthStatus,
                    sonosResponse: controller.sonosResponse,
                    hueStatus: controller.hueStatus,
                    healthError: controller.healthError,
                    sonosError: controller.sonosError,
                    hueError: controller.hueError,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
