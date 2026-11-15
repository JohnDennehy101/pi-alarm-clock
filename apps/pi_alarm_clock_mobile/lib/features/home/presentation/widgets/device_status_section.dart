import 'package:flutter/material.dart';

import 'package:pi_alarm_clock_mobile/features/home/domain/models/health_status.dart';
import 'package:pi_alarm_clock_mobile/features/home/domain/models/hue_status.dart';
import 'package:pi_alarm_clock_mobile/features/home/domain/models/sonos_response.dart';
import 'package:pi_alarm_clock_mobile/widgets/shared/device_status_tile.dart';

class DeviceStatusSection extends StatelessWidget {
  const DeviceStatusSection({
    super.key,
    required this.isLoading,
    required this.healthStatus,
    required this.sonosResponse,
    required this.hueStatus,
    required this.healthError,
    required this.sonosError,
    required this.hueError,
  });

  final bool isLoading;

  final HealthStatus? healthStatus;
  final SonosResponse? sonosResponse;
  final HueStatus? hueStatus;

  final String? healthError;
  final String? sonosError;
  final String? hueError;

  @override
  Widget build(BuildContext context) {
    final piConnected = healthStatus?.status == 'ok';
    final sonosConnected = sonosResponse?.status.connected ?? false;
    final hueConnected = hueStatus?.connected ?? false;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          'Device status',
          style: Theme.of(context).textTheme.titleLarge
              ?.copyWith(fontWeight: FontWeight.w600),
        ),
        const SizedBox(height: 16),
        Row(
          children: [
            Expanded(
              child: DeviceStatusTile(
                icon: Icons.memory,
                title: 'Raspberry Pi',
                status: _getStatusLabel(
                  connected: piConnected,
                  hasData: healthStatus != null,
                  error: healthError,
                  isLoading: isLoading,
                ),
                connected: piConnected,
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: DeviceStatusTile(
                icon: Icons.speaker,
                title: 'Audio',
                status: _getStatusLabel(
                  connected: sonosConnected,
                  hasData: sonosResponse != null,
                  error: sonosError,
                  isLoading: isLoading,
                  available: sonosResponse?.status.available,
                ),
                connected: sonosConnected,
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: DeviceStatusTile(
                icon: Icons.lightbulb_outline,
                title: 'Philips Hue',
                status: _getStatusLabel(
                  connected: hueConnected,
                  hasData: hueStatus != null,
                  error: hueError,
                  isLoading: isLoading,
                  available: hueStatus?.available,
                ),
                connected: hueConnected,
              ),
            ),
          ],
        ),
      ],
    );
  }

  String _getStatusLabel({
    required bool connected,
    required bool hasData,
    required String? error,
    required bool isLoading,
    bool? available,
  }) {
    if (connected) return 'Connected';

    if (error != null && !hasData) return 'Unavailable';

    if (isLoading && !hasData) return 'Checking';

    if (!hasData) return 'Unknown';

    if (available == true) return 'Available';

    return 'Not connected';
  }
}
