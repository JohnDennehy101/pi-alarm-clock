import 'dart:convert';
import 'dart:io';

import 'package:http/http.dart' as http;
import 'package:pi_alarm_clock_mobile/features/devices/domain/interfaces/devices_api.dart';
import 'package:pi_alarm_clock_mobile/features/devices/domain/models/hue_lights_response.dart';
import 'package:pi_alarm_clock_mobile/features/devices/domain/models/radio_stations_response.dart';
import 'package:pi_alarm_clock_mobile/features/home/domain/models/sonos_groups_response.dart';
import 'package:pi_alarm_clock_mobile/services/api/api_client.dart';

class DevicesApiService implements DevicesApi {
  DevicesApiService(this._apiClient);

  final ApiClient _apiClient;

  @override
  Future<SonosGroupsResponse> getSonosGroups() async {
    final response = await _apiClient.get('/api/v1/devices/sonos/groups');

    return SonosGroupsResponse.fromJson(_parseResponse(response));
  }

  @override
  Future<HueLightsResponse> getHueLights() async {
    final response = await _apiClient.get('/api/v1/devices/hue/lights');

    return HueLightsResponse.fromJson(_parseResponse(response));
  }

  Map<String, dynamic> _parseResponse(http.Response response) {
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw HttpException(
        'API request failed (${response.statusCode}): ${response.body}',
      );
    }

    final decoded = jsonDecode(response.body);

    if (decoded is! Map<String, dynamic>) {
      throw const FormatException(
        'Expected the API response to be a JSON object.',
      );
    }

    return decoded;
  }

  @override
  Future<RadioStationsResponse> getSonosRadioStations(String groupId) async {
    final encodedGroupId = Uri.encodeComponent(groupId);

    final response = await _apiClient.get(
      '/api/v1/devices/sonos/groups/$encodedGroupId/radio-stations',
    );

    return RadioStationsResponse.fromJson(_parseResponse(response));
  }
}
