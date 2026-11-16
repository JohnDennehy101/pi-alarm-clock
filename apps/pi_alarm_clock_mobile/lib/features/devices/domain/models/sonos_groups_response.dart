import 'sonos_group.dart';

class SonosGroupsResponse {
  const SonosGroupsResponse({required this.groups});

  final List<SonosGroup> groups;

  factory SonosGroupsResponse.fromJson(Map<String, dynamic> json) {
    return SonosGroupsResponse(
      groups: (json['groups'] as List<dynamic>)
          .map((group) => SonosGroup.fromJson(group as Map<String, dynamic>))
          .toList(),
    );
  }
}
