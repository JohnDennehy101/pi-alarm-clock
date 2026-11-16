import 'package:flutter/foundation.dart';
import 'package:uuid/uuid.dart';

import 'package:pi_alarm_clock_mobile/features/alarms/application/alarm_controller.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/domain/models/alarm.dart';

class AlarmEditorController extends ChangeNotifier {
  AlarmEditorController(this._alarmController);

  final AlarmController _alarmController;

  static const weekdays = [
    'monday',
    'tuesday',
    'wednesday',
    'thursday',
    'friday',
    'saturday',
    'sunday',
  ];

  String _name = '';
  int _hour = 7;
  int _minute = 30;
  bool _enabled = true;
  bool _isOneOff = false;
  DateTime? _date;
  final Set<String> _selectedDays = {};

  // Sonos settings
  String? _selectedSonosGroupId;
  String? _selectedRadioUri;
  String? _selectedRadioName;
  int _sonosVolume = 15;
  bool _gradualVolume = false;
  int _volumeDuration = 2;

  // Hue settings
  bool _hueEnabled = false;
  final Set<String> _selectedHueLightIds = {};
  int _hueStartBrightness = 5;
  int _hueBrightness = 60;
  bool _gradualBrightness = false;
  int _brightnessDuration = 2;

  // Existing alarm state
  Alarm? _existingAlarm;
  bool _isLoading = false;
  bool _notFound = false;

  String get name => _name;
  int get hour => _hour;
  int get minute => _minute;
  bool get enabled => _enabled;
  bool get isOneOff => _isOneOff;
  DateTime? get date => _date;
  Set<String> get selectedDays => Set.unmodifiable(_selectedDays);

  String? get selectedSonosGroupId => _selectedSonosGroupId;
  String? get selectedRadioUri => _selectedRadioUri;
  String? get selectedRadioName => _selectedRadioName;
  int get sonosVolume => _sonosVolume;
  bool get gradualVolume => _gradualVolume;
  int get volumeDuration => _volumeDuration;

  bool get hueEnabled => _hueEnabled;
  Set<String> get selectedHueLightIds => Set.unmodifiable(_selectedHueLightIds);
  int get hueStartBrightness => _hueStartBrightness;
  int get hueBrightness => _hueBrightness;
  bool get gradualBrightness => _gradualBrightness;
  int get brightnessDuration => _brightnessDuration;

  Alarm? get existingAlarm => _existingAlarm;
  bool get isLoading => _isLoading;
  bool get notFound => _notFound;
  bool get isEditing => _existingAlarm != null;

  void setName(String value) {
    _name = value;
  }

  void setTime(int hour, int minute) {
    _hour = hour;
    _minute = minute;
    notifyListeners();
  }

  void setEnabled(bool value) {
    _enabled = value;
    notifyListeners();
  }

  void setOneOff(bool value) {
    _isOneOff = value;
    notifyListeners();
  }

  void setDate(DateTime? value) {
    _date = value;
    notifyListeners();
  }

  void toggleDay(String day) {
    if (!_selectedDays.add(day)) {
      _selectedDays.remove(day);
    }
    notifyListeners();
  }

  void setSonosGroup(String? groupId) {
    _selectedSonosGroupId = groupId;
    _selectedRadioUri = null;
    _selectedRadioName = null;
    notifyListeners();
  }

  void setRadioStation(String? uri, String? name) {
    _selectedRadioUri = uri;
    _selectedRadioName = name;
    notifyListeners();
  }

  void setSonosVolume(int value) {
    _sonosVolume = value;
    notifyListeners();
  }

  void setGradualVolume(bool value) {
    _gradualVolume = value;
    notifyListeners();
  }

  void setVolumeDuration(int value) {
    _volumeDuration = value;
    notifyListeners();
  }

  void setHueEnabled(bool value) {
    _hueEnabled = value;
    notifyListeners();
  }

  void toggleHueLight(String lightId, bool selected) {
    if (selected) {
      _selectedHueLightIds.add(lightId);
    } else {
      _selectedHueLightIds.remove(lightId);
    }
    notifyListeners();
  }

  void setHueStartBrightness(int value) {
    _hueStartBrightness = value;
    notifyListeners();
  }

  void setHueBrightness(int value) {
    _hueBrightness = value;
    notifyListeners();
  }

  void setGradualBrightness(bool value) {
    _gradualBrightness = value;
    notifyListeners();
  }

  void setBrightnessDuration(int value) {
    _brightnessDuration = value;
    notifyListeners();
  }

  Future<void> load(String? alarmId) async {
    if (alarmId == null) return;

    _isLoading = true;
    _notFound = false;
    notifyListeners();

    var alarm = _findAlarm();

    if (alarm == null) {
      await _alarmController.load();
      alarm = _findAlarm();
    }

    if (alarm == null) {
      _notFound = true;
      _isLoading = false;
      notifyListeners();
      return;
    }

    _populateFromAlarm(alarm);

    _isLoading = false;
    notifyListeners();
  }

  Alarm? _findAlarm() {
    for (final alarm in _alarmController.alarms) {
      if (alarm.id == _existingAlarm?.id) return alarm;
    }
    return null;
  }

  void _populateFromAlarm(Alarm alarm) {
    _existingAlarm = alarm;

    _name = alarm.name;

    final timeParts = alarm.time.split(':');
    _hour = int.parse(timeParts[0]);
    _minute = int.parse(timeParts[1]);

    _enabled = alarm.enabled;
    _isOneOff = alarm.date != null;
    _date = alarm.date;

    _selectedDays
      ..clear()
      ..addAll(alarm.days);

    final sonos = alarm.sonos;
    _selectedSonosGroupId = sonos.target.id;
    _selectedRadioUri = sonos.source.uri;
    _selectedRadioName = sonos.source.name;
    _sonosVolume = sonos.volume;
    _gradualVolume = sonos.gradualVolume;
    _volumeDuration = sonos.volumeDuration;

    final hue = alarm.hue;
    _hueEnabled = hue.enabled;
    _selectedHueLightIds
      ..clear()
      ..addAll(hue.lightIds);
    _hueStartBrightness = hue.startBrightness;
    _hueBrightness = hue.brightness;
    _gradualBrightness = hue.gradualBrightness;
    _brightnessDuration = hue.brightnessDuration;
  }

  String? validate() {
    if (_isOneOff && _date == null) {
      return 'Please select a date.';
    }

    if (!_isOneOff && _selectedDays.isEmpty) {
      return 'Please select at least one day.';
    }

    if (_selectedSonosGroupId == null ||
        _selectedRadioUri == null ||
        _selectedRadioUri!.isEmpty) {
      return 'Select a Sonos group and radio station.';
    }

    if (_hueEnabled && _selectedHueLightIds.isEmpty) {
      return 'Select at least one Hue light.';
    }

    return null;
  }

  Alarm buildAlarm() {
    final existing = _existingAlarm;

    final sonos = SonosSettings(
      target: SonosTarget(type: 'group', id: _selectedSonosGroupId!),
      source: AudioSource(
        type: 'radio',
        uri: _selectedRadioUri,
        name: _selectedRadioName,
      ),
      volume: _sonosVolume,
      gradualVolume: _gradualVolume,
      volumeDuration: _volumeDuration,
    );

    final hue = HueSettings(
      enabled: _hueEnabled,
      lightIds: _hueEnabled ? _selectedHueLightIds.toList() : [],
      startBrightness: _hueStartBrightness,
      brightness: _hueBrightness,
      gradualBrightness: _gradualBrightness,
      brightnessDuration: _brightnessDuration,
    );

    return Alarm(
      id: existing?.id ?? const Uuid().v4(),
      name: _name.trim(),
      time:
          '${_hour.toString().padLeft(2, '0')}:'
          '${_minute.toString().padLeft(2, '0')}',
      enabled: _enabled,
      days: _isOneOff ? [] : weekdays.where(_selectedDays.contains).toList(),
      date: _isOneOff ? _date : null,
      sonos: sonos,
      hue: hue,
      createdAt: existing?.createdAt,
      updatedAt: existing?.updatedAt,
    );
  }
}
