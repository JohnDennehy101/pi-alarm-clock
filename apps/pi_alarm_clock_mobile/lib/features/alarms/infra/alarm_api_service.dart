import 'dart:convert';
import 'dart:io';

import 'package:http/http.dart' as http;
import 'package:pi_alarm_clock_mobile/features/alarms/domain/interfaces/alarm_api.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/domain/models/alarm.dart';

import 'package:pi_alarm_clock_mobile/services/api/api_client.dart';

class AlarmApiService implements AlarmApi {
  AlarmApiService(this._apiClient);

  final ApiClient _apiClient;

  @override
  Future<List<Alarm>> getAlarms() async {
    final response = await _apiClient.get('/api/v1/alarms');
    final json = _parseResponse(response);

    final alarms = json['alarms'];

    if (alarms is! List) {
      throw const FormatException(
        'Expected the alarms response to contain an alarms array.',
      );
    }

    return alarms.map((item) {
      if (item is! Map<String, dynamic>) {
        throw const FormatException('Expected each alarm to be a JSON object.');
      }

      return Alarm.fromJson(item);
    }).toList();
  }

  @override
  Future<Alarm> createAlarm(Alarm alarm) async {
    final response = await _apiClient.post('/api/v1/alarms', alarm.toJson());

    return Alarm.fromJson(_parseAlarmResponse(response));
  }

  @override
  Future<Alarm> updateAlarm(Alarm alarm) async {
    final response = await _apiClient.put(
      '/api/v1/alarms/${alarm.id}',
      alarm.toJson(),
    );

    return Alarm.fromJson(_parseAlarmResponse(response));
  }

  @override
  Future<void> deleteAlarm(String id) async {
    final response = await _apiClient.delete('/api/v1/alarms/$id');
    _parseResponse(response);
  }

  Map<String, dynamic> _parseAlarmResponse(http.Response response) {
    final json = _parseResponse(response);
    final alarm = json['alarm'];

    if (alarm is! Map<String, dynamic>) {
      throw const FormatException(
        'Expected the API response to contain an alarm object.',
      );
    }

    return alarm;
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
