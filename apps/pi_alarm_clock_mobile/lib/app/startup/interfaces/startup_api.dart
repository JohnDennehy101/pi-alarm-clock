import 'package:pi_alarm_clock_mobile/services/device_discovery/discovered_device.dart';

abstract interface class StartupApi {
  void configureServer(DiscoveredDevice device);

  Future<String?> readAuthToken();

  Future<int> checkHealth();

  Future<void> deleteAuthToken();
}
