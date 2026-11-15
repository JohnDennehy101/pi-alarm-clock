class SonosSpeaker {
  const SonosSpeaker({required this.name, required this.ip});

  final String name;
  final String ip;

  factory SonosSpeaker.fromJson(Map<String, dynamic> json) {
    return SonosSpeaker(name: json['Name'] as String, ip: json['IP'] as String);
  }
}
