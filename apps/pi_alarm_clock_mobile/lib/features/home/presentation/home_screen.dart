import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:pi_alarm_clock_mobile/widgets/shared/page_header.dart';
import 'package:provider/provider.dart';

import 'package:pi_alarm_clock_mobile/features/alarms/application/alarm_controller.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/domain/models/alarm.dart';
import 'package:pi_alarm_clock_mobile/features/home/application/home_controller.dart';
import 'package:pi_alarm_clock_mobile/features/home/presentation/widgets/clock_text.dart';
import 'package:pi_alarm_clock_mobile/features/home/presentation/widgets/device_status_section.dart';
import 'package:pi_alarm_clock_mobile/features/home/presentation/widgets/next_alarm_card.dart';

class HomeScreen extends StatelessWidget {
  const HomeScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final homeController = context.watch<HomeController>();
    final alarmController = context.watch<AlarmController>();
    final theme = Theme.of(context);

    final nextAlarm = _findNextAlarm(alarmController.alarms, DateTime.now());

    return RefreshIndicator(
      onRefresh: () async {
        await Future.wait([homeController.load(), alarmController.load()]);
      },
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
                  NextAlarmCard(
                    onViewAlarms: () => context.go('/alarms'),
                    time: nextAlarm?.alarm.time,
                    dayLabel: nextAlarm?.dayLabel,
                    scheduleLabel: nextAlarm?.scheduleLabel,
                  ),
                  const SizedBox(height: 24),
                  DeviceStatusSection(
                    isLoading: homeController.isLoading,
                    healthStatus: homeController.healthStatus,
                    sonosResponse: homeController.sonosResponse,
                    hueStatus: homeController.hueStatus,
                    healthError: homeController.healthError,
                    sonosError: homeController.sonosError,
                    hueError: homeController.hueError,
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  _NextAlarm? _findNextAlarm(List<Alarm> alarms, DateTime now) {
    _NextAlarm? next;

    for (final alarm in alarms) {
      if (!alarm.enabled) continue;

      final parts = alarm.time.split(':');
      if (parts.length != 2) continue;

      final hour = int.tryParse(parts[0]);
      final minute = int.tryParse(parts[1]);

      if (hour == null ||
          minute == null ||
          hour < 0 ||
          hour > 23 ||
          minute < 0 ||
          minute > 59) {
        continue;
      }

      DateTime? occurrence;

      if (alarm.date != null) {
        final date = alarm.date!;
        final candidate = DateTime(
          date.year,
          date.month,
          date.day,
          hour,
          minute,
        );

        if (candidate.isAfter(now)) {
          occurrence = candidate;
        }
      } else if (alarm.days.isNotEmpty) {
        final scheduledDays = alarm.days
            .map((day) => day.toLowerCase())
            .toSet();

        for (var offset = 0; offset <= 7; offset++) {
          final date = DateTime(now.year, now.month, now.day + offset);

          final weekday = _weekdayName(date.weekday).toLowerCase();

          if (!scheduledDays.contains(weekday)) continue;

          final candidate = DateTime(
            date.year,
            date.month,
            date.day,
            hour,
            minute,
          );

          if (candidate.isAfter(now)) {
            occurrence = candidate;
            break;
          }
        }
      }

      if (occurrence == null) continue;

      if (next == null || occurrence.isBefore(next.occurrence)) {
        next = _NextAlarm(
          alarm: alarm,
          occurrence: occurrence,
          dayLabel: _weekdayName(occurrence.weekday),
          scheduleLabel: _scheduleLabel(alarm),
        );
      }
    }

    return next;
  }

  String _scheduleLabel(Alarm alarm) {
    if (alarm.date != null) return 'One-off';

    final days = alarm.days.map((day) => day.toLowerCase()).toSet();

    const weekdays = {'monday', 'tuesday', 'wednesday', 'thursday', 'friday'};

    const weekends = {'saturday', 'sunday'};

    if (days.length == 7) return 'Every day';
    if (days.containsAll(weekdays) && days.length == 5) {
      return 'Every weekday';
    }
    if (days.containsAll(weekends) && days.length == 2) {
      return 'Weekends';
    }

    final names = [
      for (var day = 1; day <= 7; day++)
        if (days.contains(_weekdayName(day).toLowerCase())) _weekdayName(day),
    ];

    return 'Every ${names.join(', ')}';
  }

  String _weekdayName(int weekday) {
    const names = [
      'Monday',
      'Tuesday',
      'Wednesday',
      'Thursday',
      'Friday',
      'Saturday',
      'Sunday',
    ];

    return names[weekday - 1];
  }
}

class _NextAlarm {
  const _NextAlarm({
    required this.alarm,
    required this.occurrence,
    required this.dayLabel,
    required this.scheduleLabel,
  });

  final Alarm alarm;
  final DateTime occurrence;
  final String dayLabel;
  final String scheduleLabel;
}
