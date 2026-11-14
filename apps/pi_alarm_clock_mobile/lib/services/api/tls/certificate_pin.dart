part of '../api_client.dart';

class _Pin {
  final String host;
  final int port;
  final String fingerprint;

  const _Pin({
    required this.host,
    required this.port,
    required this.fingerprint,
  });

  String encode() {
    return jsonEncode({'host': host, 'port': port, 'fingerprint': fingerprint});
  }

  static _Pin? decode(String? raw) {
    final jsonMap = _parseJsonMap(raw);
    if (jsonMap == null) return null;

    return _extractAndValidatePin(jsonMap);
  }

  static Map<String, dynamic>? _parseJsonMap(String? raw) {
    if (raw == null || raw.trim().isEmpty) return null;

    try {
      final decoded = jsonDecode(raw);
      if (decoded is Map<String, dynamic>) {
        return decoded;
      }
    } catch (_) {
      // JSON decoding failed or incorrect type
    }
    return null;
  }

  static _Pin? _extractAndValidatePin(Map<String, dynamic> map) {
    final host = map['host'];
    final port = map['port'];
    final fingerprint = map['fingerprint'];

    if (host is! String) {
      return null;
    }

    if (port is! int || fingerprint is! String) {
      return null;
    }

    final normalisedHost = _normaliseHost(host);
    final normalisedFingerprint = fingerprint.toLowerCase();

    if (normalisedFingerprint.isEmpty) return null;
    if (port < 1 || port > 65535) return null;
    if (!_isValidFingerprint(normalisedFingerprint)) return null;

    return _Pin(
      host: normalisedHost,
      port: port,
      fingerprint: normalisedFingerprint,
    );
  }
}
