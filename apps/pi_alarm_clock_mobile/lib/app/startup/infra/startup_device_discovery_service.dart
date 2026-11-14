import 'package:pi_alarm_clock_mobile/app/startup/interfaces/startup_device_discovery.dart';
import 'package:pi_alarm_clock_mobile/services/device_discovery/device_discovery_service.dart';
import 'package:pi_alarm_clock_mobile/services/device_discovery/discovered_device.dart';

class StartupDeviceDiscoveryService implements StartupDeviceDiscovery {
  final DeviceDiscoveryService _discovery;

  StartupDeviceDiscoveryService(this._discovery);

  @override
  Future<List<DiscoveredDevice>> discover() {
    return _discovery.discover();
  }
}
