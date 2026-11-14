import 'package:pi_alarm_clock_mobile/app/startup/startup_destination.dart';
import 'package:pi_alarm_clock_mobile/app/startup/interfaces/startup_api.dart';
import 'package:pi_alarm_clock_mobile/app/startup/interfaces/startup_device_discovery.dart';
import 'package:pi_alarm_clock_mobile/app/startup/exceptions/startup_exceptions.dart';

class StartupService {
  final StartupDeviceDiscovery discovery;
  final StartupApi api;

  StartupService({required this.discovery, required this.api});

  Future<StartupDestination> initialise() async {
    final devices = await discovery.discover();

    if (devices.isEmpty) {
      throw PiNotFoundException();
    }

    api.configureServer(devices.first);

    final token = await api.readAuthToken();

    if (token == null) {
      return StartupDestination.pairing;
    }

    final statusCode = await api.checkHealth();

    switch (statusCode) {
      case 200:
        return StartupDestination.home;

      case 401:
        await api.deleteAuthToken();
        return StartupDestination.pairing;

      default:
        throw UnexpectedServerResponseException(statusCode);
    }
  }
}
