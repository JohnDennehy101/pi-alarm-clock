import 'package:flutter/material.dart';

class AlarmTimeCard extends StatelessWidget {
  const AlarmTimeCard({
    super.key,
    required this.time,
    required this.enabled,
    required this.onTimeTap,
    required this.onEnabledChanged,
  });

  final TimeOfDay time;
  final bool enabled;
  final VoidCallback onTimeTap;
  final ValueChanged<bool> onEnabledChanged;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final colors = theme.colorScheme;

    return Card(
      elevation: 0,
      color: colors.primaryContainer,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(24)),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 14, horizontal: 20),
        child: Column(
          children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Icon(Icons.alarm, size: 18, color: colors.onPrimaryContainer),
                const SizedBox(width: 4),
                Text(
                  'ALARM TIME',
                  style: theme.textTheme.labelMedium?.copyWith(
                    letterSpacing: 1.5,
                    color: colors.onPrimaryContainer,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 4),
            InkWell(
              onTap: onTimeTap,
              borderRadius: BorderRadius.circular(16),
              child: Padding(
                padding: const EdgeInsets.symmetric(
                  horizontal: 20,
                  vertical: 4,
                ),
                child: Text(
                  time.format(context),
                  style: theme.textTheme.displayLarge?.copyWith(
                    fontWeight: FontWeight.w600,
                    fontSize: 52,
                    color: colors.onPrimaryContainer,
                  ),
                ),
              ),
            ),
            Text(
              'Tap to change time',
              style: theme.textTheme.bodySmall?.copyWith(
                color: colors.onPrimaryContainer.withValues(alpha: 0.7),
              ),
            ),
            const SizedBox(height: 16),
            Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                Text(
                  'Alarm enabled',
                  style: theme.textTheme.bodyMedium?.copyWith(
                    color: colors.onPrimaryContainer,
                  ),
                ),
                const SizedBox(width: 8),
                Switch.adaptive(value: enabled, onChanged: onEnabledChanged),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
