import 'package:flutter/material.dart';

class AlarmDeviceSection extends StatelessWidget {
  const AlarmDeviceSection({
    super.key,
    required this.sonosSummary,
    required this.hueSummary,
    required this.onSonosTap,
    required this.onHueTap,
  });

  final String sonosSummary;
  final String hueSummary;
  final VoidCallback onSonosTap;
  final VoidCallback onHueTap;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text(
          'Device(s) configuration',
          style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600),
        ),
        const SizedBox(height: 4),
        Text(
          'Choose how you want to wake up.',
          style: TextStyle(
            color: Theme.of(context).colorScheme.onSurfaceVariant,
          ),
        ),
        const SizedBox(height: 16),
        _DeviceCard(
          icon: Icons.speaker_rounded,
          title: 'Sonos',
          subtitle: sonosSummary,
          onTap: onSonosTap,
        ),
        const SizedBox(height: 12),
        _DeviceCard(
          icon: Icons.lightbulb_outline_rounded,
          title: 'Philips Hue',
          subtitle: hueSummary,
          onTap: onHueTap,
        ),
      ],
    );
  }
}

class _DeviceCard extends StatelessWidget {
  const _DeviceCard({
    required this.icon,
    required this.title,
    required this.subtitle,
    required this.onTap,
  });

  final IconData icon;
  final String title;
  final String subtitle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      margin: EdgeInsets.zero,
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Row(
            children: [
              Container(
                width: 44,
                height: 44,
                decoration: BoxDecoration(
                  color: theme.colorScheme.secondaryContainer,
                  borderRadius: BorderRadius.circular(12),
                ),
                child: Icon(icon),
              ),
              const SizedBox(width: 14),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      title,
                      style: const TextStyle(fontWeight: FontWeight.w600),
                    ),
                    const SizedBox(height: 3),
                    Text(subtitle, style: theme.textTheme.bodySmall),
                  ],
                ),
              ),
              const SizedBox(width: 8),
              const Icon(Icons.tune_rounded),
            ],
          ),
        ),
      ),
    );
  }
}
