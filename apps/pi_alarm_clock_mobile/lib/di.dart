import 'package:kiwi/kiwi.dart';
import 'package:pi_alarm_clock_mobile/app/navigation/app_navigator.dart';
import 'package:pi_alarm_clock_mobile/app/startup/infra/startup_api_service.dart';
import 'package:pi_alarm_clock_mobile/app/startup/infra/startup_device_discovery_service.dart';
import 'package:pi_alarm_clock_mobile/app/startup/interfaces/startup_api.dart';
import 'package:pi_alarm_clock_mobile/app/startup/interfaces/startup_device_discovery.dart';
import 'package:pi_alarm_clock_mobile/app/startup/startup_service.dart';
import 'package:pi_alarm_clock_mobile/features/pairing/domain/interfaces/pairing_api.dart';
import 'package:pi_alarm_clock_mobile/features/pairing/infra/pairing_api_service.dart';
import 'package:pi_alarm_clock_mobile/features/pairing/pairing_service.dart';
import 'package:pi_alarm_clock_mobile/services/api/api_client.dart';
import 'package:pi_alarm_clock_mobile/services/device_discovery/device_discovery_service.dart';
import 'package:pi_alarm_clock_mobile/widgets/shared/verify_server_dialog.dart';

void setupDependencies() {
  final container = KiwiContainer();

  final apiClient = ApiClient()
    ..fingerprintVerifier = (fingerprint, {required identityChanged}) async {
      final context = AppNavigator.context;
      if (context == null) return false;

      return showVerifyServerDialog(
        context,
        fingerprint: fingerprint,
        identityChanged: identityChanged,
      );
    };

  container.registerInstance(apiClient);

  container.registerFactory<DeviceDiscoveryService>(
    (c) => DeviceDiscoveryService(),
  );

  container.registerFactory<StartupDeviceDiscovery>(
    (c) => StartupDeviceDiscoveryService(c.resolve<DeviceDiscoveryService>()),
  );

  container.registerFactory<StartupApi>(
    (c) => StartupApiService(c.resolve<ApiClient>()),
  );

  container.registerFactory<StartupService>(
    (c) => StartupService(
      discovery: c.resolve<StartupDeviceDiscovery>(),
      api: c.resolve<StartupApi>(),
    ),
  );

  container.registerFactory<PairingApi>(
    (c) => PairingApiService(c.resolve<ApiClient>()),
  );

  container.registerFactory<PairingService>(
    (c) => PairingService(api: c.resolve<PairingApi>()),
  );
}
