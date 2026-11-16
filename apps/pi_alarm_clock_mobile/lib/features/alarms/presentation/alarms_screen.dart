import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/presentation/widgets/alarm_empty_state.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/presentation/widgets/alarm_error_state.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/presentation/widgets/alarm_list_item.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/presentation/widgets/alarm_next_card.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/presentation/widgets/alarms_header.dart';
import 'package:provider/provider.dart';

import 'package:pi_alarm_clock_mobile/features/alarms/application/alarm_controller.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/domain/models/alarm.dart';

class AlarmsScreen extends StatelessWidget {
  const AlarmsScreen({super.key});

  @override
  Widget build(BuildContext context) {
    final controller = context.watch<AlarmController>();
    final theme = Theme.of(context);
    final alarms = controller.alarms;
    final nextAlarm = _findNextAlarm(alarms);

    return Scaffold(
      body: SafeArea(
        child: RefreshIndicator(
          onRefresh: controller.load,
          child: CustomScrollView(
            physics: const AlwaysScrollableScrollPhysics(),
            slivers: [
              const SliverPadding(
                padding: EdgeInsets.fromLTRB(20, 24, 20, 16),
                sliver: SliverToBoxAdapter(child: AlarmsHeader()),
              ),
              SliverPadding(
                padding: const EdgeInsets.symmetric(horizontal: 20),
                sliver: SliverToBoxAdapter(
                  child: AlarmNextCard(
                    alarm: nextAlarm?.key,
                    occurrence: nextAlarm?.value,
                    scheduleText: nextAlarm == null
                        ? null
                        : _formatSchedule(context, nextAlarm.key),
                  ),
                ),
              ),
              SliverPadding(
                padding: const EdgeInsets.fromLTRB(20, 28, 20, 12),
                sliver: SliverToBoxAdapter(
                  child: Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Text(
                        'Your alarms',
                        style: theme.textTheme.titleLarge?.copyWith(
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                      Text(
                        '${alarms.length} ${alarms.length == 1 ? 'alarm' : 'alarms'}',
                        style: theme.textTheme.bodySmall?.copyWith(
                          color: theme.colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
              if (controller.isLoading && alarms.isEmpty)
                const SliverFillRemaining(
                  child: Center(child: CircularProgressIndicator()),
                )
              else if (controller.error != null && alarms.isEmpty)
                SliverFillRemaining(
                  child: AlarmErrorState(onRetry: controller.load),
                )
              else if (alarms.isEmpty)
                const SliverFillRemaining(child: AlarmEmptyState())
              else
                SliverPadding(
                  padding: const EdgeInsets.symmetric(horizontal: 20),
                  sliver: SliverList.separated(
                    itemCount: alarms.length,
                    separatorBuilder: (_, _) => const SizedBox(height: 12),
                    itemBuilder: (context, index) {
                      final alarm = alarms[index];

                      return AlarmListItem(
                        alarm: alarm,
                        scheduleText: _formatSchedule(context, alarm),
                        onTap: () => context.push('/alarms/${alarm.id}/edit'),
                        onEnabledChanged: (enabled) =>
                            _toggleAlarm(context, alarm, enabled),
                      );
                    },
                  ),
                ),
              const SliverToBoxAdapter(child: SizedBox(height: 100)),
            ],
          ),
        ),
      ),
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => context.push('/alarms/new'),
        icon: const Icon(Icons.add),
        label: const Text('Add alarm'),
      ),
    );
  }

  Future<void> _toggleAlarm(
    BuildContext context,
    Alarm alarm,
    bool enabled,
  ) async {
    final controller = context.read<AlarmController>();

    final success = await controller.update(alarm.copyWith(enabled: enabled));

    if (!success && context.mounted) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(controller.error ?? 'Could not update alarm.')),
      );
    }
  }

  MapEntry<Alarm, DateTime>? _findNextAlarm(List<Alarm> alarms) {
    final now = DateTime.now();
    MapEntry<Alarm, DateTime>? next;

    for (final alarm in alarms.where((alarm) => alarm.enabled)) {
      final occurrence = _nextOccurrence(alarm, now);

      if (occurrence == null) continue;

      if (next == null || occurrence.isBefore(next.value)) {
        next = MapEntry(alarm, occurrence);
      }
    }

    return next;
  }

  DateTime? _nextOccurrence(Alarm alarm, DateTime now) {
    final timeParts = alarm.time.split(':');

    if (timeParts.length != 2) return null;

    final hour = int.tryParse(timeParts[0]);
    final minute = int.tryParse(timeParts[1]);

    if (hour == null || minute == null) return null;
    if (hour < 0 || hour > 23 || minute < 0 || minute > 59) {
      return null;
    }

    if (alarm.date != null) {
      final date = alarm.date!;

      final occurrence = DateTime(
        date.year,
        date.month,
        date.day,
        hour,
        minute,
      );

      return occurrence.isAfter(now) ? occurrence : null;
    }

    const weekdays = {
      'monday': DateTime.monday,
      'tuesday': DateTime.tuesday,
      'wednesday': DateTime.wednesday,
      'thursday': DateTime.thursday,
      'friday': DateTime.friday,
      'saturday': DateTime.saturday,
      'sunday': DateTime.sunday,
    };

    final selectedDays = alarm.days
        .map((day) => weekdays[day.toLowerCase()])
        .whereType<int>()
        .toSet();

    if (selectedDays.isEmpty) return null;

    for (var offset = 0; offset <= 7; offset++) {
      final date = DateTime(now.year, now.month, now.day + offset);

      if (!selectedDays.contains(date.weekday)) continue;

      final occurrence = DateTime(
        date.year,
        date.month,
        date.day,
        hour,
        minute,
      );

      if (occurrence.isAfter(now)) {
        return occurrence;
      }
    }

    return null;
  }

  String _formatSchedule(BuildContext context, Alarm alarm) {
    if (alarm.date != null) {
      return 'Once · ${MaterialLocalizations.of(context).formatMediumDate(alarm.date!)}';
    }

    const abbreviations = {
      'monday': 'MON',
      'tuesday': 'TUE',
      'wednesday': 'WED',
      'thursday': 'THU',
      'friday': 'FRI',
      'saturday': 'SAT',
      'sunday': 'SUN',
    };

    return alarm.days
        .map((day) => abbreviations[day.toLowerCase()] ?? day.toUpperCase())
        .join(' · ');
  }
}
