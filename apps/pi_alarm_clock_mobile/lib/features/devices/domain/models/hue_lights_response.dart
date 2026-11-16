import 'hue_light.dart';

class HueLightsResponse {
  const HueLightsResponse({required this.lights});

  final List<HueLight> lights;

  factory HueLightsResponse.fromJson(Map<String, dynamic> json) {
    return HueLightsResponse(
      lights: json.entries.map((entry) {
        return HueLight.fromJson(
          entry.key,
          entry.value as Map<String, dynamic>,
        );
      }).toList(),
    );
  }
}
