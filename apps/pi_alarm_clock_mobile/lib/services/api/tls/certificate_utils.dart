part of '../api_client.dart';

String _normaliseHost(String host) {
  final value = host.toLowerCase();

  if (value.endsWith('.') && !value.contains(':')) {
    return value.substring(0, value.length - 1);
  }

  return value;
}

bool _isValidFingerprint(String fingerprint) {
  return RegExp(r'^[0-9a-f]{64}$').hasMatch(fingerprint);
}

bool _hostMatches(String pattern, String host) {
  pattern = _normaliseHost(pattern);
  host = _normaliseHost(host);

  if (pattern == host) return true;

  if (!pattern.startsWith('*.')) return false;

  final suffix = pattern.substring(1);

  return host.endsWith(suffix) &&
      host.length > suffix.length &&
      !host.substring(0, host.length - suffix.length).contains('.');
}
