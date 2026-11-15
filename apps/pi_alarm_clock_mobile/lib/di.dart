import 'package:kiwi/kiwi.dart';
import 'package:pi_alarm_clock_mobile/app/navigation/app_navigator.dart';
import 'package:pi_alarm_clock_mobile/app/startup/infra/startup_api_service.dart';
import 'package:pi_alarm_clock_mobile/app/startup/infra/startup_device_discovery_service.dart';
import 'package:pi_alarm_clock_mobile/app/startup/interfaces/startup_api.dart';
import 'package:pi_alarm_clock_mobile/app/startup/interfaces/startup_device_discovery.dart';
import 'package:pi_alarm_clock_mobile/app/startup/startup_service.dart';
import 'package:pi_alarm_clock_mobile/features/home/application/home_controller.dart';
import 'package:pi_alarm_clock_mobile/features/home/domain/interfaces/home_api.dart';
import 'package:pi_alarm_clock_mobile/features/home/infra/home_api_service.dart';
import 'package:pi_alarm_clock_mobile/features/pairing/domain/interfaces/pairing_api.dart';
import 'package:pi_alarm_clock_mobile/features/pairing/infra/pairing_api_service.dart';
import 'package:pi_alarm_clock_mobile/features/pairing/pairing_service.dart';
import 'package:pi_alarm_clock_mobile/services/api/api_client.dart';
import 'package:pi_alarm_clock_mobile/services/device_discovery/device_discovery_service.dart';
import 'package:pi_alarm_clock_mobile/widgets/shared/verify_server_dialog.dart';

final KiwiContainer _container = KiwiContainer();

void setupDependencies() {
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

  _container.registerInstance(apiClient);

  _container.registerFactory<DeviceDiscoveryService>(
    (c) => DeviceDiscoveryService(),
  );

  _container.registerFactory<StartupDeviceDiscovery>(
    (c) => StartupDeviceDiscoveryService(c.resolve<DeviceDiscoveryService>()),
  );

  _container.registerFactory<StartupApi>(
    (c) => StartupApiService(c.resolve<ApiClient>()),
  );

  _container.registerFactory<StartupService>(
    (c) => StartupService(
      discovery: c.resolve<StartupDeviceDiscovery>(),
      api: c.resolve<StartupApi>(),
    ),
  );

  _container.registerFactory<PairingApi>(
    (c) => PairingApiService(c.resolve<ApiClient>()),
  );

  _container.registerFactory<PairingService>(
    (c) => PairingService(api: c.resolve<PairingApi>()),
  );

  _container.registerFactory<HomeApi>(
    (c) => HomeApiService(c.resolve<ApiClient>()),
  );

  _container.registerFactory<HomeController>(
    (c) => HomeController(c.resolve<HomeApi>()),
  );
}

T resolve<T>() => _container.resolve<T>();
