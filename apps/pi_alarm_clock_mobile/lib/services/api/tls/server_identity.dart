part of '../api_client.dart';

class _ServerIdentity {
  final String host;
  final int port;

  const _ServerIdentity({required this.host, required this.port});

  factory _ServerIdentity.fromUri(Uri uri) {
    return _ServerIdentity(host: _normaliseHost(uri.host), port: uri.port);
  }

  String get key => jsonEncode([host, port]);

  bool matches(String host, int port) {
    return this.host == _normaliseHost(host) && this.port == port;
  }

  @override
  bool operator ==(Object other) {
    return other is _ServerIdentity && host == other.host && port == other.port;
  }

  @override
  int get hashCode => Object.hash(host, port);
}
