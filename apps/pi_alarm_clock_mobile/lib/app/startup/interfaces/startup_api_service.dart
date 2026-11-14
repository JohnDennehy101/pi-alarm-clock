import 'package:http/http.dart' as http;
import 'package:pi_alarm_clock_mobile/services/device_discovery/discovered_device.dart';

abstract interface class StartupApiService {
  void setServerBaseUrlFromDevice(DiscoveredDevice device);
  Future<http.Response> get(String endpoint);
  Future<String?> readAuthToken();
  Future<void> deleteAuthToken();
}
