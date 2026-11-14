import 'package:pi_alarm_clock_mobile/features/pairing/interfaces/pairing_api.dart';

enum PairingResult { success, tooManyAttempts, invalidPin, unexpectedResponse }

class PairingService {
  final PairingApi api;

  PairingService({required this.api});

  Future<PairingResult> pair({
    required String pin,
    required String deviceName,
  }) async {
    final statusCode = await api.pair(pin: pin, deviceName: deviceName);

    switch (statusCode) {
      case 201:
        return PairingResult.success;

      case 429:
        return PairingResult.tooManyAttempts;

      case 400:
      case 401:
        return PairingResult.invalidPin;

      default:
        return PairingResult.unexpectedResponse;
    }
  }
}
