import 'package:pi_alarm_clock_mobile/features/alarms/domain/interfaces/alarm_api.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/domain/models/alarm.dart';

class AlarmRepository {
  final AlarmApi api;

  AlarmRepository({required this.api});

  Future<List<Alarm>> getAlarms() => api.getAlarms();

  Future<Alarm> createAlarm(Alarm alarm) => api.createAlarm(alarm);

  Future<Alarm> updateAlarm(Alarm alarm) => api.updateAlarm(alarm);

  Future<void> deleteAlarm(String id) => api.deleteAlarm(id);
}
