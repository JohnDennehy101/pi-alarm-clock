import 'package:flutter/material.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/domain/models/alarm.dart';

import '../infra/alarm_repository.dart';

class AlarmController extends ChangeNotifier {
  final AlarmRepository repository;

  AlarmController(this.repository);

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
      _alarms = await repository.getAlarms();
    } catch (e) {
      _error = e.toString();
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Future<bool> create(Alarm alarm) async {
    try {
      final created = await repository.createAlarm(alarm);
      _alarms = [..._alarms, created];
      notifyListeners();
      return true;
    } catch (e) {
      _error = e.toString();
      notifyListeners();
      return false;
    }
  }

  Future<bool> update(Alarm alarm) async {
    try {
      final updated = await repository.updateAlarm(alarm);

      _alarms = _alarms
          .map((existing) => existing.id == updated.id ? updated : existing)
          .toList();

      notifyListeners();
      return true;
    } catch (e) {
      _error = e.toString();
      notifyListeners();
      return false;
    }
  }

  Future<bool> delete(String id) async {
    try {
      await repository.deleteAlarm(id);
      _alarms = _alarms.where((alarm) => alarm.id != id).toList();
      notifyListeners();
      return true;
    } catch (e) {
      _error = e.toString();
      notifyListeners();
      return false;
    }
  }
}
