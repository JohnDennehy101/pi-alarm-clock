class HueStatus {
  const HueStatus({
    required this.available,
    required this.connected,
    this.bridgeIP,
    this.error,
  });

  final bool available;
  final bool connected;
  final String? bridgeIP;
  final String? error;

  factory HueStatus.fromJson(Map<String, dynamic> json) {
    return HueStatus(
      available: json['available'] as bool,
      connected: json['connected'] as bool,
      bridgeIP: json['bridgeIP'] as String?,
      error: json['error'] as String?,
    );
  }
}
