import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
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
              SliverPadding(
                padding: const EdgeInsets.fromLTRB(20, 24, 20, 16),
                sliver: SliverToBoxAdapter(child: _buildHeader(context, theme)),
              ),
              SliverPadding(
                padding: const EdgeInsets.symmetric(horizontal: 20),
                sliver: SliverToBoxAdapter(
                  child: _buildNextAlarm(context, theme, nextAlarm),
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
                SliverFillRemaining(child: _buildError(context, controller))
              else if (alarms.isEmpty)
                SliverFillRemaining(child: _buildEmptyState(theme))
              else
                SliverPadding(
                  padding: const EdgeInsets.symmetric(horizontal: 20),
                  sliver: SliverList.separated(
                    itemCount: alarms.length,
                    separatorBuilder: (_, __) => const SizedBox(height: 12),
                    itemBuilder: (context, index) {
                      final alarm = alarms[index];

                      return _buildAlarmCard(context, theme, alarm);
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

  Widget _buildHeader(BuildContext context, ThemeData theme) {
    final now = DateTime.now();
    final date = MaterialLocalizations.of(context).formatFullDate(now);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Alarms',
          style: theme.textTheme.headlineSmall?.copyWith(
            fontWeight: FontWeight.bold,
          ),
        ),
        const SizedBox(height: 4),
        Text(
          date,
          style: theme.textTheme.bodyMedium?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ],
    );
  }

  Widget _buildNextAlarm(
    BuildContext context,
    ThemeData theme,
    MapEntry<Alarm, DateTime>? entry,
  ) {
    final alarm = entry?.key;
    final occurrence = entry?.value;

    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: theme.colorScheme.primaryContainer,
        borderRadius: BorderRadius.circular(24),
      ),
      child: alarm == null
          ? Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                const Icon(Icons.alarm_off_outlined),
                const SizedBox(height: 12),
                Text('No upcoming alarms', style: theme.textTheme.titleLarge),
                const SizedBox(height: 4),
                const Text('Enable an alarm to see it here.'),
              ],
            )
          : Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text(
                      'NEXT ALARM',
                      style: theme.textTheme.labelMedium?.copyWith(
                        fontWeight: FontWeight.bold,
                        letterSpacing: 1.2,
                      ),
                    ),
                    const Icon(Icons.wb_twilight, size: 28),
                  ],
                ),
                const SizedBox(height: 12),
                Text(
                  alarm.time,
                  style: theme.textTheme.displayMedium?.copyWith(
                    fontWeight: FontWeight.w300,
                  ),
                ),
                const SizedBox(height: 4),
                Text(alarm.name, style: theme.textTheme.titleMedium),
                const SizedBox(height: 8),
                Text(
                  _formatSchedule(context, alarm),
                  style: theme.textTheme.bodySmall,
                ),
                const SizedBox(height: 4),
                Text(
                  MaterialLocalizations.of(context)
                      .formatMediumDate(occurrence!),
                  style: theme.textTheme.bodySmall,
                ),
              ],
            ),
    );
  }

  Widget _buildAlarmCard(BuildContext context, ThemeData theme, Alarm alarm) {
    return Material(
      color: theme.colorScheme.surfaceContainerLow,
      borderRadius: BorderRadius.circular(20),
      child: InkWell(
        borderRadius: BorderRadius.circular(20),
        onTap: () => context.push('/alarms/${alarm.id}/edit'),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      alarm.time,
                      style: theme.textTheme.headlineMedium?.copyWith(
                        fontWeight: FontWeight.w500,
                        color: alarm.enabled
                            ? theme.colorScheme.onSurface
                            : theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                    const SizedBox(height: 4),
                    Text(alarm.name, style: theme.textTheme.titleSmall),
                    const SizedBox(height: 6),
                    Text(
                      _formatSchedule(context, alarm),
                      style: theme.textTheme.labelSmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
              Switch(
                value: alarm.enabled,
                onChanged: (enabled) => _toggleAlarm(context, alarm, enabled),
              ),
            ],
          ),
        ),
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

  Widget _buildEmptyState(ThemeData theme) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.alarm_add_outlined,
              size: 48,
              color: theme.colorScheme.onSurfaceVariant,
            ),
            const SizedBox(height: 16),
            Text('No alarms yet', style: theme.textTheme.titleLarge),
            const SizedBox(height: 8),
            Text(
              'Add an alarm to get started.',
              style: theme.textTheme.bodyMedium?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildError(BuildContext context, AlarmController controller) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.error_outline, size: 40),
            const SizedBox(height: 12),
            const Text('Could not load alarms.'),
            const SizedBox(height: 12),
            FilledButton(
              onPressed: controller.load,
              child: const Text('Retry'),
            ),
          ],
        ),
      ),
    );
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
