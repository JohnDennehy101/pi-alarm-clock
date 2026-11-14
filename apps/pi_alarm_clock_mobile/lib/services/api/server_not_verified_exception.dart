class ServerNotVerifiedException implements Exception {
  final String message;
  final String? fingerprint;

  ServerNotVerifiedException(this.message, {this.fingerprint});

  @override
  String toString() => message;
}
