import 'dart:async';

import 'package:nsd/nsd.dart';

import 'discovered_device.dart';

class DeviceDiscoveryService {
  static const String _serviceType = '_pialarmclock._tcp';

  static const Duration _defaultTimeout = Duration(seconds: 6);

  Future<List<DiscoveredDevice>> discover({
    Duration timeout = _defaultTimeout,
  }) async {
    final devices = <String, DiscoveredDevice>{};

    final discovery = await startDiscovery(
      _serviceType,
      autoResolve: true,
      ipLookupType: IpLookupType.any,
    );

    try {
      discovery.addServiceListener((service, status) {
        _handleServiceEvent(service, status, devices);
      });

      await Future.delayed(timeout);
    } finally {
      await stopDiscovery(discovery);
    }

    return devices.values.toList();
  }

  void _handleServiceEvent(
    Service service,
    ServiceStatus status,
    Map<String, DiscoveredDevice> devices,
  ) {
    switch (status) {
      case ServiceStatus.found:
        _onServiceFound(service, devices);

      case ServiceStatus.lost:
        _onServiceLost(service, devices);
    }
  }

  void _onServiceFound(Service service, Map<String, DiscoveredDevice> devices) {
    final host = service.host;
    final port = service.port;

    if (host == null || port == null) return;

    final key = '$host:$port';

    devices[key] = DiscoveredDevice(
      name: service.name ?? 'Pi Alarm Clock',
      host: host,
      port: port,
    );
  }

  void _onServiceLost(Service service, Map<String, DiscoveredDevice> devices) {
    final host = service.host;
    final port = service.port;

    if (host == null || port == null) return;

    devices.remove('$host:$port');
  }
}
