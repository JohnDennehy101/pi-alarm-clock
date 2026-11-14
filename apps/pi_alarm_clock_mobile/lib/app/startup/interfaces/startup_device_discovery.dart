import 'package:pi_alarm_clock_mobile/services/device_discovery/discovered_device.dart';

abstract interface class StartupDeviceDiscovery {
  Future<List<DiscoveredDevice>> discover();
}
