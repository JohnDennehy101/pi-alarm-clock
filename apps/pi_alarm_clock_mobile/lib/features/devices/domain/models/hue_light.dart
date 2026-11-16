class HueLight {
  const HueLight({
    required this.id,
    required this.name,
    required this.isOn,
    this.brightness,
  });

  final String id;
  final String name;
  final bool isOn;
  final double? brightness;

  factory HueLight.fromJson(String id, Map<String, dynamic> json) {
    final metadata = json['metadata'] as Map<String, dynamic>;
    final on = json['on'] as Map<String, dynamic>;
    final dimming = json['dimming'] as Map<String, dynamic>?;

    return HueLight(
      id: id,
      name: metadata['name'] as String,
      isOn: on['on'] as bool,
      brightness: (dimming?['brightness'] as num?)?.toDouble(),
    );
  }
}
