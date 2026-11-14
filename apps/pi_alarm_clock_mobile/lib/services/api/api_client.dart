import 'dart:async';
import 'dart:convert';
import 'dart:io' as io;

import 'package:basic_utils/basic_utils.dart';
import 'package:crypto/crypto.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:http/http.dart' as http;
import 'package:http/io_client.dart';
import 'package:pi_alarm_clock_mobile/services/api/server_not_verified_exception.dart';
import 'package:pi_alarm_clock_mobile/services/device_discovery/discovered_device.dart';
part 'tls/server_identity.dart';
part 'tls/certificate_pin.dart';
part 'tls/certificate_validator.dart';
part 'tls/certificate_utils.dart';

class _Handshake {
  final _ServerIdentity identity;

  _Handshake(this.identity);

  String? presentedFingerprint;
  bool identityChanged = false;
  bool invalidCertificate = false;
  String? invalidReason;
}

class ApiClient {
  static const storage = FlutterSecureStorage(
    mOptions: MacOsOptions(useDataProtectionKeyChain: false),
  );

  static const _tokenKey = 'auth_token';
  static const _fingerprintKey = 'server_fingerprint';

  static const _requestTimeout = Duration(seconds: 6);
  static const _connectionTimeout = Duration(seconds: 4);

  static String? _serverBaseUrl;

  static String? get serverBaseUrl => _serverBaseUrl;

  static void setServerBaseUrl(String url) {
    final Uri uri;

    try {
      uri = Uri.parse(url);
    } catch (_) {
      throw ArgumentError('Invalid server URL.');
    }

    if (_isValidHttpsUri(uri) == false) {
      throw ArgumentError('Invalid HTTPS server URL.');
    }

    final newIdentity = _ServerIdentity.fromUri(uri);

    final normalised = uri.replace(
      scheme: 'https',
      host: uri.host.toLowerCase(),
      path: '',
    );

    final newUrl = normalised.toString();

    final oldUrl = _serverBaseUrl;

    if (oldUrl != null) {
      final oldIdentity = _ServerIdentity.fromUri(Uri.parse(oldUrl));

      if (oldIdentity == newIdentity) {
        _serverBaseUrl = newUrl;
        return;
      }
    }

    _serverBaseUrl = newUrl;

    _pairingGeneration++;

    _pendingVerifications.clear();
  }

  static bool _isValidHttpsUri(Uri uri) =>
      _hasHttpsScheme(uri) &&
      _hasHost(uri) &&
      _hasNoUserInfo(uri) &&
      _hasNoQueryOrFragment(uri) &&
      _hasValidPath(uri) &&
      _hasValidPort(uri);

  static bool _hasHttpsScheme(Uri uri) => uri.scheme.toLowerCase() == 'https';

  static bool _hasHost(Uri uri) => uri.host.isNotEmpty;

  static bool _hasNoUserInfo(Uri uri) => uri.userInfo.isEmpty;

  static bool _hasNoQueryOrFragment(Uri uri) =>
      !uri.hasQuery && !uri.hasFragment;

  static bool _hasValidPath(Uri uri) => uri.path.isEmpty || uri.path == '/';

  static bool _hasValidPort(Uri uri) => uri.port >= 1 && uri.port <= 65535;

  static void setServerBaseUrlFromDevice(DiscoveredDevice device) {
    final baseUrl = getBaseUrl(device);
    setServerBaseUrl(baseUrl);
  }

  static String getBaseUrl(DiscoveredDevice device) {
    if (io.Platform.isMacOS) {
      return 'https://localhost:${device.port}';
    }

    return Uri(
      scheme: 'https',
      host: device.host,
      port: device.port,
    ).toString();
  }

  static Future<bool> Function(
    String fingerprint, {
    required bool identityChanged,
  })?
  fingerprintVerifier;

  static _Pin? _pin;

  static Future<void>? _initialization;

  static int _pairingGeneration = 0;

  static final Map<String, Future<String?>> _pendingVerifications = {};

  static Future<void> _pinWrites = Future<void>.value();

  static Future<void> _init() async {
    final existing = _initialization;

    if (existing != null) {
      return existing;
    }

    final generation = _pairingGeneration;
    final future = _loadPin(generation);

    _initialization = future;

    try {
      await future;
    } catch (e) {
      if (identical(_initialization, future)) {
        _initialization = null;
      }

      rethrow;
    }
  }

  static Future<void> _loadPin(int generation) async {
    final stored = await storage.read(key: _fingerprintKey);

    if (generation != _pairingGeneration) return;

    _pin = _Pin.decode(stored);
  }

  static io.HttpClient _createPinnedClient(_Handshake handshake) {
    final context = io.SecurityContext(withTrustedRoots: false);

    final client = io.HttpClient(context: context)
      ..connectionTimeout = _connectionTimeout;

    client.badCertificateCallback =
        (io.X509Certificate cert, String host, int port) {
          return _verifyCertificate(cert, host, port, handshake);
        };

    return client;
  }

  static bool _verifyCertificate(
    io.X509Certificate cert,
    String host,
    int port,
    _Handshake handshake,
  ) {
    if (!handshake.identity.matches(host, port)) {
      handshake.invalidCertificate = true;
      handshake.invalidReason =
          'The TLS server identity does not match the configured endpoint.';
      return false;
    }

    final now = DateTime.now();

    if (now.isBefore(cert.startValidity) || now.isAfter(cert.endValidity)) {
      handshake.invalidCertificate = true;
      handshake.invalidReason =
          'Server certificate is expired or not yet valid.';
      return false;
    }

    final sanResult = _certCoversHost(cert, host);

    if (sanResult != _SanResult.ok) {
      handshake.invalidCertificate = true;
      handshake.invalidReason = sanResult == _SanResult.parseError
          ? 'Server certificate could not be parsed.'
          : 'Server certificate does not contain a valid SAN for $host.';

      return false;
    }

    final fingerprint = sha256.convert(cert.der).toString();

    final pin = _pin;

    if (pin != null &&
        pin.host == handshake.identity.host &&
        pin.port == handshake.identity.port &&
        pin.fingerprint == fingerprint) {
      return true;
    }

    handshake.presentedFingerprint = fingerprint;

    handshake.identityChanged =
        pin != null &&
        pin.host == handshake.identity.host &&
        pin.port == handshake.identity.port;

    return false;
  }

  static Future<String?> _prompt(
    _ServerIdentity identity,
    String fingerprint,
    bool identityChanged,
  ) {
    final key = identity.key;

    final existing = _pendingVerifications[key];

    if (existing != null) {
      return existing;
    }

    final verifier = fingerprintVerifier;

    if (verifier == null) {
      return Future<String?>.value(null);
    }

    final future = verifier(
      fingerprint,
      identityChanged: identityChanged,
    ).then<String?>((confirmed) => confirmed ? fingerprint : null);

    _pendingVerifications[key] = future;

    future.whenComplete(() {
      if (identical(_pendingVerifications[key], future)) {
        _pendingVerifications.remove(key);
      }
    }).ignore();

    return future;
  }

  static Future<void> _trustFingerprint(_Pin pin, int generation) {
    final operation = _pinWrites.catchError((_) {}).then((_) async {
      if (generation != _pairingGeneration) {
        throw ServerNotVerifiedException('Device pairing was cancelled.');
      }

      await storage.write(key: _fingerprintKey, value: pin.encode());

      if (generation != _pairingGeneration) {
        await storage.delete(key: _fingerprintKey);

        throw ServerNotVerifiedException('Device pairing was cancelled.');
      }

      _pin = pin;
    });

    _pinWrites = operation;

    return operation;
  }

  static Never _throwRetryCertificateError(_Handshake handshake) {
    if (handshake.invalidCertificate) {
      final reason =
          handshake.invalidReason ?? 'Server certificate is invalid.';

      throw ServerNotVerifiedException('$reason Connection refused.');
    }

    final fingerprint = handshake.presentedFingerprint!;

    throw ServerNotVerifiedException(
      'The server certificate changed during pairing. '
      'Connection refused.',
      fingerprint: fingerprint,
    );
  }

  static Future<http.Response> _send(
    Future<http.Response> Function() attempt,
    _Handshake handshake,
    int generation,
  ) async {
    await _init();

    try {
      return await attempt().timeout(_requestTimeout);
    } catch (e) {
      if (handshake.invalidCertificate) {
        final reason =
            handshake.invalidReason ?? 'Server certificate is invalid.';

        handshake.invalidCertificate = false;
        handshake.invalidReason = null;

        throw ServerNotVerifiedException('$reason Connection refused.');
      }

      final pending = handshake.presentedFingerprint;

      if (pending == null) rethrow;

      handshake.presentedFingerprint = null;

      final identityChanged = handshake.identityChanged;
      handshake.identityChanged = false;

      if (generation != _pairingGeneration) {
        throw ServerNotVerifiedException(
          'Server configuration changed. Please retry.',
          fingerprint: pending,
        );
      }

      if (fingerprintVerifier == null) {
        throw ServerNotVerifiedException(
          'Server certificate not trusted yet. '
          'Pair first to verify it.',
          fingerprint: pending,
        );
      }

      final confirmed = await _prompt(
        handshake.identity,
        pending,
        identityChanged,
      );

      if (confirmed == null || confirmed != pending) {
        throw ServerNotVerifiedException(
          identityChanged
              ? 'Server identity changed and was not verified. '
                    'Connection refused.'
              : 'Server certificate was not verified. '
                    'Connection refused.',
          fingerprint: pending,
        );
      }

      if (generation != _pairingGeneration) {
        throw ServerNotVerifiedException(
          'Device pairing was cancelled.',
          fingerprint: pending,
        );
      }

      final pin = _Pin(
        host: handshake.identity.host,
        port: handshake.identity.port,
        fingerprint: confirmed,
      );

      await _trustFingerprint(pin, generation);

      if (generation != _pairingGeneration) {
        throw ServerNotVerifiedException('Device pairing was cancelled.');
      }

      try {
        return await attempt().timeout(_requestTimeout);
      } catch (_) {
        if (handshake.invalidCertificate ||
            handshake.presentedFingerprint != null) {
          _throwRetryCertificateError(handshake);
        }

        rethrow;
      }
    }
  }

  static Uri _buildRequestUri(String baseUrl, String endpoint) {
    if (!endpoint.startsWith('/') ||
        endpoint.startsWith('//') ||
        endpoint.contains(r'\')) {
      throw ArgumentError(
        'Endpoint must be an absolute API path starting with one "/".',
      );
    }

    final Uri path;

    try {
      path = Uri.parse(endpoint);
    } catch (_) {
      throw ArgumentError('Invalid API endpoint.');
    }

    if (path.hasScheme ||
        path.hasAuthority ||
        path.userInfo.isNotEmpty ||
        path.hasFragment) {
      throw ArgumentError('Invalid API endpoint.');
    }

    if (path.pathSegments.any((segment) => segment == '.' || segment == '..')) {
      throw ArgumentError('Endpoint path traversal is not permitted.');
    }

    return Uri.parse(baseUrl).resolveUri(path);
  }

  static Future<http.Response> get(String endpoint) async {
    final baseUrl = _serverBaseUrl;

    if (baseUrl == null) {
      throw StateError('serverBaseUrl not set. Call setServerBaseUrl() first.');
    }

    final uri = _buildRequestUri(baseUrl, endpoint);
    final identity = _ServerIdentity.fromUri(uri);
    final generation = _pairingGeneration;

    final handshake = _Handshake(identity);
    final client = IOClient(_createPinnedClient(handshake));

    try {
      return await _send(
        () async {
          final token = await storage.read(key: _tokenKey);

          return client.get(
            uri,
            headers: {if (token != null) 'Authorization': 'Bearer $token'},
          );
        },
        handshake,
        generation,
      );
    } finally {
      client.close();
    }
  }

  static Future<http.Response> post(
    String endpoint,
    Map<String, dynamic> body,
  ) async {
    final baseUrl = _serverBaseUrl;

    if (baseUrl == null) {
      throw StateError('serverBaseUrl not set. Call setServerBaseUrl() first.');
    }

    final uri = _buildRequestUri(baseUrl, endpoint);
    final identity = _ServerIdentity.fromUri(uri);
    final generation = _pairingGeneration;

    final handshake = _Handshake(identity);
    final client = IOClient(_createPinnedClient(handshake));

    try {
      return await _send(
        () async {
          final token = await storage.read(key: _tokenKey);

          return client.post(
            uri,
            headers: {
              'Content-Type': 'application/json',
              if (token != null) 'Authorization': 'Bearer $token',
            },
            body: jsonEncode(body),
          );
        },
        handshake,
        generation,
      );
    } finally {
      client.close();
    }
  }

  static Future<http.Response> delete(String endpoint) async {
    final baseUrl = _serverBaseUrl;

    if (baseUrl == null) {
      throw StateError('serverBaseUrl not set. Call setServerBaseUrl() first.');
    }

    final uri = _buildRequestUri(baseUrl, endpoint);
    final identity = _ServerIdentity.fromUri(uri);
    final generation = _pairingGeneration;

    final handshake = _Handshake(identity);
    final client = IOClient(_createPinnedClient(handshake));

    try {
      return await _send(
        () async {
          final token = await storage.read(key: _tokenKey);

          return client.delete(
            uri,
            headers: {if (token != null) 'Authorization': 'Bearer $token'},
          );
        },
        handshake,
        generation,
      );
    } finally {
      client.close();
    }
  }

  static Future<void> forgetDevice() async {
    _pairingGeneration++;

    _pin = null;
    _initialization = null;
    _pendingVerifications.clear();
    _serverBaseUrl = null;

    try {
      await _pinWrites;
    } catch (_) {}

    await storage.delete(key: _tokenKey);
    await storage.delete(key: _fingerprintKey);
  }

  static const _deviceIdKey = 'device_id';

  static Future<void> logout() async {
    final deviceId = await storage.read(key: _deviceIdKey);

    try {
      if (deviceId != null && _serverBaseUrl != null) {
        await delete('/api/v1/devices/$deviceId');
      }
    } finally {
      await forgetDevice();
      await storage.delete(key: _deviceIdKey);
    }
  }
}
