import '../models/alarm.dart';

abstract class AlarmApi {
  Future<List<Alarm>> getAlarms();

  Future<Alarm> createAlarm(Alarm alarm);

  Future<Alarm> updateAlarm(Alarm alarm);

  Future<void> deleteAlarm(String id);
}
