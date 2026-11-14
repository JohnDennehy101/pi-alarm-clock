import 'package:pi_alarm_clock_mobile/app/startup/interfaces/startup_api.dart';
import 'package:pi_alarm_clock_mobile/services/api/api_client.dart';
import 'package:pi_alarm_clock_mobile/services/device_discovery/discovered_device.dart';

class StartupApiService implements StartupApi {
  @override
  void configureServer(DiscoveredDevice device) {
    ApiClient.setServerBaseUrlFromDevice(device);
  }

  @override
  Future<String?> readAuthToken() {
    return ApiClient.storage.read(key: 'auth_token');
  }

  @override
  Future<int> checkHealth() async {
    final response = await ApiClient.get('/api/v1/health');
    return response.statusCode;
  }

  @override
  Future<void> deleteAuthToken() async {
    await ApiClient.storage.delete(key: 'auth_token');
  }
}
