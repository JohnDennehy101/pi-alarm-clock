import 'dart:convert';
import 'dart:io';

import 'package:http/http.dart' as http;

import 'package:pi_alarm_clock_mobile/features/home/domain/interfaces/home_api.dart';
import 'package:pi_alarm_clock_mobile/features/home/domain/models/health_status.dart';
import 'package:pi_alarm_clock_mobile/features/home/domain/models/hue_status.dart';
import 'package:pi_alarm_clock_mobile/features/home/domain/models/sonos_groups_response.dart';
import 'package:pi_alarm_clock_mobile/features/home/domain/models/sonos_response.dart';
import 'package:pi_alarm_clock_mobile/services/api/api_client.dart';

class HomeApiService implements HomeApi {
  HomeApiService(this._apiClient);

  final ApiClient _apiClient;

  @override
  Future<HealthStatus> getHealth() async {
    final response = await _apiClient.get('/api/v1/health');

    return HealthStatus.fromJson(_parseResponse(response));
  }

  @override
  Future<SonosResponse> getSonos() async {
    final response = await _apiClient.get('/api/v1/devices/sonos');

    return SonosResponse.fromJson(_parseResponse(response));
  }

  @override
  Future<HueStatus> getHue() async {
    final response = await _apiClient.get('/api/v1/devices/hue');

    return HueStatus.fromJson(_parseResponse(response));
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
}
