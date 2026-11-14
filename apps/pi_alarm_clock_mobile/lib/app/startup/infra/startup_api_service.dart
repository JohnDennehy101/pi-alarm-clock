import 'package:pi_alarm_clock_mobile/app/startup/interfaces/startup_api.dart';
import 'package:pi_alarm_clock_mobile/services/api/api_client.dart';
import 'package:pi_alarm_clock_mobile/services/device_discovery/discovered_device.dart';

class StartupApiService implements StartupApi {
  final ApiClient _apiClient;

  StartupApiService(this._apiClient);

  @override
  void configureServer(DiscoveredDevice device) {
    _apiClient.setServerBaseUrlFromDevice(device);
  }

  @override
  Future<String?> readAuthToken() {
    return _apiClient.readAuthToken();
  }

  @override
  Future<int> checkHealth() async {
    final response = await _apiClient.get('/api/v1/health');
    return response.statusCode;
  }

  @override
  Future<void> deleteAuthToken() async {
    await _apiClient.deleteAuthToken();
  }

  @override
  Future<void> logout() async {
    await _apiClient.logout();
  }
}
