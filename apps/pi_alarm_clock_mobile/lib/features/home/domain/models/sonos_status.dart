class SonosStatus {
  const SonosStatus({
    required this.available,
    required this.connected,
    this.selectedSpeaker,
    this.error,
  });

  final bool available;
  final bool connected;
  final String? selectedSpeaker;
  final String? error;

  factory SonosStatus.fromJson(Map<String, dynamic> json) {
    return SonosStatus(
      available: json['available'] as bool,
      connected: json['connected'] as bool,
      selectedSpeaker: json['selectedSpeaker'] as String?,
      error: json['error'] as String?,
    );
  }
}
