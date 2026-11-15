import 'package:flutter/foundation.dart';

import 'package:pi_alarm_clock_mobile/features/home/domain/interfaces/home_api.dart';
import 'package:pi_alarm_clock_mobile/features/home/domain/models/health_status.dart';
import 'package:pi_alarm_clock_mobile/features/home/domain/models/hue_status.dart';
import 'package:pi_alarm_clock_mobile/features/home/domain/models/sonos_response.dart';

class HomeController extends ChangeNotifier {
  HomeController(this._homeApi);

  final HomeApi _homeApi;

  HealthStatus? _healthStatus;
  SonosResponse? _sonosResponse;
  HueStatus? _hueStatus;

  String? _healthError;
  String? _sonosError;
  String? _hueError;

  bool _isLoading = false;

  HealthStatus? get healthStatus => _healthStatus;
  SonosResponse? get sonosResponse => _sonosResponse;
  HueStatus? get hueStatus => _hueStatus;

  String? get healthError => _healthError;
  String? get sonosError => _sonosError;
  String? get hueError => _hueError;

  bool get isLoading => _isLoading;

  Future<void> load() async {
    if (_isLoading) return;

    _isLoading = true;

    _healthError = null;
    _sonosError = null;
    _hueError = null;

    notifyListeners();

    await Future.wait([_loadHealth(), _loadSonos(), _loadHue()]);

    _isLoading = false;

    notifyListeners();
  }

  Future<void> _loadHealth() async {
    try {
      _healthStatus = await _homeApi.getHealth();
    } catch (error) {
      _healthError = error.toString();
    }
  }

  Future<void> _loadSonos() async {
    try {
      _sonosResponse = await _homeApi.getSonos();
    } catch (error) {
      _sonosError = error.toString();
    }
  }

  Future<void> _loadHue() async {
    try {
      _hueStatus = await _homeApi.getHue();
    } catch (error) {
      _hueError = error.toString();
    }
  }
}
