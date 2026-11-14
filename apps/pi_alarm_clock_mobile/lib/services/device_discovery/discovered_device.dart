class DiscoveredDevice {
  final String name;
  final String host;
  final int port;

  const DiscoveredDevice({
    required this.name,
    required this.host,
    required this.port,
  });

  String get baseUrl => Uri(scheme: 'https', host: host, port: port).toString();
}
