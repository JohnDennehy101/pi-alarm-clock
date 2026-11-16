class SonosSpeaker {
  const SonosSpeaker({
    required this.id,
    required this.name,
    required this.ip,
    required this.manufacturer,
    required this.modelName,
    required this.modelNumber,
    required this.modelDescription,
  });

  final String id;
  final String name;
  final String ip;
  final String manufacturer;
  final String modelName;
  final String modelNumber;
  final String modelDescription;

  factory SonosSpeaker.fromJson(Map<String, dynamic> json) {
    return SonosSpeaker(
      id: json['id'] as String,
      name: json['name'] as String,
      ip: json['ip'] as String,
      manufacturer: json['manufacturer'] as String,
      modelName: json['modelName'] as String,
      modelNumber: json['modelNumber'] as String,
      modelDescription: json['modelDescription'] as String,
    );
  }
}
