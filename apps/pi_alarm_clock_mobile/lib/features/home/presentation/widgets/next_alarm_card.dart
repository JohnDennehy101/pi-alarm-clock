import 'package:flutter/material.dart';

class NextAlarmCard extends StatelessWidget {
  const NextAlarmCard({
    super.key,
    required this.onViewAlarms,
    this.time,
    this.dayLabel,
    this.scheduleLabel,
  });

  final VoidCallback onViewAlarms;
  final String? time;
  final String? dayLabel;
  final String? scheduleLabel;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final hasAlarm = time != null;

    return Card(
      color: colors.primaryContainer,
      elevation: 0,
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: onViewAlarms,
        child: Padding(
          padding: const EdgeInsets.all(18),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(
                    'UP NEXT',
                    style: theme.textTheme.labelMedium?.copyWith(
                      color: colors.onPrimaryContainer,
                      fontWeight: FontWeight.w600,
                      letterSpacing: 1,
                    ),
                  ),
                  Icon(
                    hasAlarm ? Icons.alarm : Icons.alarm_off,
                    color: colors.onPrimaryContainer,
                    size: 20,
                  ),
                ],
              ),
              const SizedBox(height: 12),
              if (hasAlarm) ...[
                Row(
                  mainAxisAlignment: MainAxisAlignment.spaceBetween,
                  children: [
                    Text(
                      time!,
                      style: theme.textTheme.displayMedium?.copyWith(
                        color: colors.onPrimaryContainer,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    Column(
                      crossAxisAlignment: CrossAxisAlignment.end,
                      children: [
                        if (dayLabel != null)
                          Text(
                            dayLabel!,
                            style: theme.textTheme.bodyMedium?.copyWith(
                              color: colors.onPrimaryContainer,
                            ),
                          ),
                        if (scheduleLabel != null)
                          Text(
                            scheduleLabel!,
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: colors.onPrimaryContainer,
                            ),
                          ),
                      ],
                    ),
                  ],
                ),
              ] else ...[
                Text(
                  'No upcoming alarms',
                  style: theme.textTheme.titleLarge?.copyWith(
                    color: colors.onPrimaryContainer,
                  ),
                ),
                const SizedBox(height: 4),
                Text(
                  'Create an alarm to get started.',
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: colors.onPrimaryContainer,
                  ),
                ),
              ],
              const SizedBox(height: 12),
              Divider(color: colors.onPrimaryContainer.withValues(alpha: 0.2)),
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(
                    'View all alarms',
                    style: theme.textTheme.labelLarge?.copyWith(
                      color: colors.onPrimaryContainer,
                    ),
                  ),
                  Icon(
                    Icons.arrow_forward,
                    color: colors.onPrimaryContainer,
                    size: 18,
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}
