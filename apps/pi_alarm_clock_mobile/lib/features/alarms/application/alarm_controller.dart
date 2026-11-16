import 'package:flutter/foundation.dart';

import 'package:pi_alarm_clock_mobile/features/alarms/domain/models/alarm.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/infra/alarm_repository.dart';

class AlarmController extends ChangeNotifier {
  AlarmController(this._repository);

  final AlarmRepository _repository;

  List<Alarm> _alarms = [];
  bool _isLoading = false;
  String? _error;

  List<Alarm> get alarms => List.unmodifiable(_alarms);
  bool get isLoading => _isLoading;
  String? get error => _error;

  Future<void> load() async {
    _isLoading = true;
    _error = null;
    notifyListeners();

    try {
      _alarms = await _repository.getAlarms();
    } catch (error) {
      _error = error.toString();
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<bool> create(Alarm alarm) async {
    _error = null;

    try {
      final createdAlarm = await _repository.createAlarm(alarm);
      _alarms = [..._alarms, createdAlarm];
      notifyListeners();
      return true;
    } catch (error) {
      _error = error.toString();
      notifyListeners();
      return false;
    }
  }

  Future<bool> update(Alarm alarm) async {
    _error = null;

    try {
      final updatedAlarm = await _repository.updateAlarm(alarm);

      _alarms = _alarms
          .map(
            (existing) =>
                existing.id == updatedAlarm.id ? updatedAlarm : existing,
          )
          .toList();

      notifyListeners();
      return true;
    } catch (error) {
      _error = error.toString();
      notifyListeners();
      return false;
    }
  }

  Future<bool> delete(String id) async {
    _error = null;

    try {
      await _repository.deleteAlarm(id);
      _alarms = _alarms.where((alarm) => alarm.id != id).toList();
      notifyListeners();
      return true;
    } catch (error) {
      _error = error.toString();
      notifyListeners();
      return false;
    }
  }
}
