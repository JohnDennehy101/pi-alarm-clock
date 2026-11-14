abstract interface class PairingApi {
  Future<int> pair({required String pin, required String deviceName});
}
