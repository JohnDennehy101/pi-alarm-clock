class SonosGroup {
  const SonosGroup({
    required this.id,
    required this.coordinatorId,
    required this.members,
  });

  final String id;
  final String coordinatorId;
  final List<SonosGroupMember> members;

  /// Friendly name derived from the speakers in the group.
  String get name => members.map((member) => member.name).join(' + ');

  factory SonosGroup.fromJson(Map<String, dynamic> json) {
    return SonosGroup(
      id: json['id'] as String,
      coordinatorId: json['coordinatorId'] as String,
      members: (json['members'] as List<dynamic>)
          .map(
            (member) =>
                SonosGroupMember.fromJson(member as Map<String, dynamic>),
          )
          .toList(),
    );
  }
}

class SonosGroupMember {
  const SonosGroupMember({required this.speakerId, required this.name});

  final String speakerId;
  final String name;

  factory SonosGroupMember.fromJson(Map<String, dynamic> json) {
    return SonosGroupMember(
      speakerId: json['speakerId'] as String,
      name: json['name'] as String,
    );
  }
}
