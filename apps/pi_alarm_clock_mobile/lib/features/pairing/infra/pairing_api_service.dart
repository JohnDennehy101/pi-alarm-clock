import 'dart:convert';

import 'package:pi_alarm_clock_mobile/features/pairing/interfaces/pairing_api.dart';
import 'package:pi_alarm_clock_mobile/services/api/api_client.dart';

class PairingApiService implements PairingApi {
  @override
  Future<int> pair({required String pin, required String deviceName}) async {
    final response = await ApiClient.post('/api/v1/auth/pair', {
      'pin': pin,
      'deviceName': deviceName,
    });

    if (response.statusCode == 201) {
      final data = jsonDecode(response.body);

      if (data is! Map<String, dynamic>) {
        throw const FormatException('Invalid pairing response.');
      }

      if (data['token'] is! String || (data['token'] as String).isEmpty) {
        throw const FormatException('Invalid pairing response.');
      }

      await saveAuthToken(data['token'] as String);
    }

    return response.statusCode;
  }

  Future<void> saveAuthToken(String token) async {
    await ApiClient.storage.write(key: 'auth_token', value: token);
  }
}
