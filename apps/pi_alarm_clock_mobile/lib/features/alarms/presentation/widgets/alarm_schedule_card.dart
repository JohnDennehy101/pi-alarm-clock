import 'package:flutter/material.dart';

class AlarmScheduleCard extends StatelessWidget {
  const AlarmScheduleCard({
    super.key,
    required this.isOneOff,
    required this.selectedDays,
    required this.date,
    required this.scheduleDescription,
    required this.onToggleOneOff,
    required this.onDayToggle,
    required this.onDateTap,
  });

  final bool isOneOff;
  final Set<String> selectedDays;
  final DateTime? date;
  final String scheduleDescription;
  final ValueChanged<bool> onToggleOneOff;
  final ValueChanged<String> onDayToggle;
  final VoidCallback onDateTap;

  static const weekdays = [
    'monday',
    'tuesday',
    'wednesday',
    'thursday',
    'friday',
    'saturday',
    'sunday',
  ];
  static const weekdayLabels = ['M', 'T', 'W', 'T', 'F', 'S', 'S'];

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      elevation: 0,
      color: theme.colorScheme.surfaceContainerLow,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(20)),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.calendar_month_outlined),
                const SizedBox(width: 10),
                Text(
                  'Schedule',
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 16),
            SegmentedButton<bool>(
              showSelectedIcon: false,
              segments: const [
                ButtonSegment(
                  value: false,
                  label: Text('Repeat'),
                  icon: Icon(Icons.repeat),
                ),
                ButtonSegment(
                  value: true,
                  label: Text('One-off'),
                  icon: Icon(Icons.event),
                ),
              ],
              selected: {isOneOff},
              onSelectionChanged: (selection) =>
                  onToggleOneOff(selection.first),
            ),
            const SizedBox(height: 20),
            if (isOneOff)
              _DateSelector(date: date, onTap: onDateTap)
            else
              _WeekdaySelector(
                selectedDays: selectedDays,
                onDayToggle: onDayToggle,
              ),
            const SizedBox(height: 12),
            Text(
              scheduleDescription,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.primary,
                fontWeight: FontWeight.w600,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _WeekdaySelector extends StatelessWidget {
  const _WeekdaySelector({
    required this.selectedDays,
    required this.onDayToggle,
  });

  final Set<String> selectedDays;
  final ValueChanged<String> onDayToggle;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      mainAxisAlignment: MainAxisAlignment.spaceBetween,
      children: List.generate(AlarmScheduleCard.weekdays.length, (index) {
        final day = AlarmScheduleCard.weekdays[index];
        final selected = selectedDays.contains(day);
        return InkWell(
          onTap: () => onDayToggle(day),
          customBorder: const CircleBorder(),
          child: AnimatedContainer(
            duration: const Duration(milliseconds: 160),
            width: 38,
            height: 38,
            decoration: BoxDecoration(
              shape: BoxShape.circle,
              color: selected
                  ? theme.colorScheme.primary
                  : theme.colorScheme.surfaceContainerHighest,
              border: Border.all(
                color: selected
                    ? theme.colorScheme.primary
                    : theme.colorScheme.outlineVariant,
              ),
            ),
            alignment: Alignment.center,
            child: Text(
              AlarmScheduleCard.weekdayLabels[index],
              style: theme.textTheme.labelMedium?.copyWith(
                fontWeight: FontWeight.w600,
                color: selected
                    ? theme.colorScheme.onPrimary
                    : theme.colorScheme.onSurface,
              ),
            ),
          ),
        );
      }),
    );
  }
}

class _DateSelector extends StatelessWidget {
  const _DateSelector({required this.date, required this.onTap});

  final DateTime? date;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(14),
      child: Container(
        padding: const EdgeInsets.all(16),
        decoration: BoxDecoration(
          border: Border.all(color: theme.colorScheme.outlineVariant),
          borderRadius: BorderRadius.circular(14),
        ),
        child: Row(
          children: [
            const Icon(Icons.calendar_today_outlined),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    date == null
                        ? 'Choose date'
                        : MaterialLocalizations.of(context)
                              .formatMediumDate(date!),
                    style: theme.textTheme.titleSmall,
                  ),
                  Text(
                    'This alarm will only ring once',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ],
              ),
            ),
            const Icon(Icons.chevron_right),
          ],
        ),
      ),
    );
  }
}
