import 'package:flutter/foundation.dart';

import 'package:pi_alarm_clock_mobile/features/pairing/pairing_service.dart';
import 'package:pi_alarm_clock_mobile/services/api/server_not_verified_exception.dart';

class PairingController extends ChangeNotifier {
  PairingController(this._pairingService);

  final PairingService _pairingService;

  String _pin = '';
  String _deviceName = '';
  bool _isPairing = false;
  String? _error;

  String get pin => _pin;

  String get deviceName => _deviceName;

  bool get isPairing => _isPairing;

  String? get error => _error;

  bool get canPair {
    return RegExp(r'^\d{6}$').hasMatch(_pin) &&
        _deviceName.trim().isNotEmpty &&
        _deviceName.trim().length <= 64 &&
        !_isPairing;
  }

  void updatePin(String value) {
    _pin = value;
    _error = null;
    notifyListeners();
  }

  void updateDeviceName(String value) {
    _deviceName = value;
    _error = null;
    notifyListeners();
  }

  Future<bool> pair() async {
    if (!canPair) return false;

    _isPairing = true;
    _error = null;
    notifyListeners();

    try {
      final result = await _pairingService.pair(
        pin: _pin,
        deviceName: _deviceName.trim(),
      );

      switch (result) {
        case PairingResult.success:
          return true;

        case PairingResult.tooManyAttempts:
          _error = 'Too many failed attempts. Locked out for 15 minutes.';
          break;

        case PairingResult.invalidPin:
          _error = 'Invalid pairing PIN.';
          break;

        case PairingResult.unexpectedResponse:
          _error = 'Unexpected response from Raspberry Pi.';
          break;
      }
    } on ServerNotVerifiedException catch (e) {
      _error = e.message;
    } catch (e) {
      _error = 'Failed to reach server: $e';
    } finally {
      _isPairing = false;
      notifyListeners();
    }

    return false;
  }
}
