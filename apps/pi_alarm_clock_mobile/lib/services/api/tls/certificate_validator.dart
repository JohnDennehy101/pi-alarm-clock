part of '../api_client.dart';

enum _SanResult { ok, mismatch, parseError }

_SanResult _certCoversHost(io.X509Certificate cert, String host) {
  try {
    final sans = _extractSubjectAltNames(cert);
    if (sans == null || sans.isEmpty) {
      return _SanResult.mismatch;
    }

    final normalisedHost = _normaliseHost(host);
    final hostIsIp = io.InternetAddress.tryParse(normalisedHost) != null;

    for (final entry in sans) {
      final san = entry.trim();
      if (san.isEmpty) continue;

      if (_sanEntryMatches(san, normalisedHost, hostIsIp)) {
        return _SanResult.ok;
      }
    }

    return _SanResult.mismatch;
  } catch (_) {
    return _SanResult.parseError;
  }
}

List<String>? _extractSubjectAltNames(io.X509Certificate cert) {
  final pem =
      '''
-----BEGIN CERTIFICATE-----
${base64.encode(cert.der)}
-----END CERTIFICATE-----
''';
  final parsed = X509Utils.x509CertificateFromPem(pem);
  return parsed.tbsCertificate?.extensions?.subjectAlternativNames;
}

bool _sanEntryMatches(String san, String normalisedHost, bool hostIsIp) {
  final rawIp = io.InternetAddress.tryParse(san);
  if (rawIp != null) {
    return hostIsIp && _ipAddressesMatch(san, normalisedHost);
  }

  final separator = san.indexOf(':');
  if (separator > 0) {
    return _typedSanEntryMatches(san, separator, normalisedHost, hostIsIp);
  }

  return !hostIsIp && _hostMatches(san, normalisedHost);
}

bool _typedSanEntryMatches(
  String san,
  int separator,
  String normalisedHost,
  bool hostIsIp,
) {
  final type = san.substring(0, separator).trim().toUpperCase();
  final value = san.substring(separator + 1).trim();
  if (value.isEmpty) return false;

  switch (type) {
    case 'DNS':
    case 'DNS NAME':
      return !hostIsIp && _hostMatches(value, normalisedHost);
    case 'IP':
    case 'IP ADDRESS':
      return hostIsIp && _ipAddressesMatch(value, normalisedHost);
    default:
      return false;
  }
}

bool _ipAddressesMatch(String a, String b) {
  final first = io.InternetAddress.tryParse(a);
  final second = io.InternetAddress.tryParse(b);
  if (first == null || second == null) return false;
  if (first.type != second.type) return false;
  return _bytesEqual(first.rawAddress, second.rawAddress);
}

bool _bytesEqual(List<int> a, List<int> b) {
  if (a.length != b.length) return false;
  for (var i = 0; i < a.length; i++) {
    if (a[i] != b[i]) return false;
  }
  return true;
}
