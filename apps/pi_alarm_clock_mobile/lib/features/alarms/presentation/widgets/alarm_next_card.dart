import 'package:flutter/material.dart';

import '../../domain/models/alarm.dart';

class AlarmNextCard extends StatelessWidget {
  const AlarmNextCard({
    super.key,
    required this.alarm,
    required this.occurrence,
    required this.scheduleText,
  });

  final Alarm? alarm;
  final DateTime? occurrence;
  final String? scheduleText;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

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
                  alarm!.time,
                  style: theme.textTheme.displayMedium?.copyWith(
                    fontWeight: FontWeight.w300,
                  ),
                ),
                const SizedBox(height: 4),
                Text(alarm!.name, style: theme.textTheme.titleMedium),
                const SizedBox(height: 8),
                Text(scheduleText ?? '', style: theme.textTheme.bodySmall),
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
}
