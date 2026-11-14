class UnexpectedServerResponseException implements Exception {
  final int statusCode;

  const UnexpectedServerResponseException(this.statusCode);

  @override
  String toString() {
    return 'Unexpected server response ($statusCode).';
  }
}

class PiNotFoundException implements Exception {
  @override
  String toString() {
    return 'Could not find your Raspberry Pi on the network.';
  }
}
