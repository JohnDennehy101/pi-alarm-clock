import 'package:flutter/foundation.dart';

import 'package:pi_alarm_clock_mobile/features/devices/domain/interfaces/devices_api.dart';
import 'package:pi_alarm_clock_mobile/features/devices/domain/models/hue_light.dart';
import 'package:pi_alarm_clock_mobile/features/devices/domain/models/radio_station.dart';
import 'package:pi_alarm_clock_mobile/features/home/domain/models/sonos_group.dart';

class DeviceOptionsController extends ChangeNotifier {
  DeviceOptionsController(this._devicesApi);

  final DevicesApi _devicesApi;

  List<SonosGroup> _sonosGroups = [];
  List<HueLight> _hueLights = [];

  bool _isLoadingSonosGroups = false;
  bool _isLoadingHueLights = false;

  bool _sonosGroupsLoaded = false;
  bool _hueLightsLoaded = false;

  final Map<String, List<RadioStation>> _radioStations = {};
  final Set<String> _loadingRadioGroups = {};
  final Map<String, String> _radioErrors = {};

  String? _sonosError;
  String? _hueError;

  List<RadioStation> radioStationsForGroup(String groupId) =>
      List.unmodifiable(_radioStations[groupId] ?? []);

  bool isLoadingRadioStations(String groupId) =>
      _loadingRadioGroups.contains(groupId);

  String? radioStationsError(String groupId) => _radioErrors[groupId];

  List<SonosGroup> get sonosGroups => List.unmodifiable(_sonosGroups);
  List<HueLight> get hueLights => List.unmodifiable(_hueLights);

  bool get isLoadingSonosGroups => _isLoadingSonosGroups;
  bool get isLoadingHueLights => _isLoadingHueLights;

  String? get sonosError => _sonosError;
  String? get hueError => _hueError;

  Future<void> loadSonosGroups({bool force = false}) async {
    if (_isLoadingSonosGroups) return;
    if (_sonosGroupsLoaded && !force) return;

    _isLoadingSonosGroups = true;
    _sonosError = null;
    notifyListeners();

    try {
      final response = await _devicesApi.getSonosGroups();

      _sonosGroups = response.groups;
      _sonosGroupsLoaded = true;
    } catch (error) {
      _sonosError = error.toString();
    } finally {
      _isLoadingSonosGroups = false;
      notifyListeners();
    }
  }

  Future<void> loadSonosRadioStations(
    String groupId, {
    bool force = false,
  }) async {
    if (_loadingRadioGroups.contains(groupId)) return;

    if (_radioStations.containsKey(groupId) && !force) return;

    _loadingRadioGroups.add(groupId);
    _radioErrors.remove(groupId);
    notifyListeners();

    try {
      final response = await _devicesApi.getSonosRadioStations(groupId);

      _radioStations[groupId] = response.playableStations;
    } catch (error) {
      _radioErrors[groupId] = error.toString();
    } finally {
      _loadingRadioGroups.remove(groupId);
      notifyListeners();
    }
  }

  Future<void> loadHueLights({bool force = false}) async {
    if (_isLoadingHueLights) return;
    if (_hueLightsLoaded && !force) return;

    _isLoadingHueLights = true;
    _hueError = null;
    notifyListeners();

    try {
      final response = await _devicesApi.getHueLights();

      _hueLights = response.lights;
      _hueLightsLoaded = true;
    } catch (error) {
      _hueError = error.toString();
    } finally {
      _isLoadingHueLights = false;
      notifyListeners();
    }
  }
}
