import 'package:pi_alarm_clock_mobile/features/home/domain/models/sonos_speaker.dart';
import 'package:pi_alarm_clock_mobile/features/home/domain/models/sonos_status.dart';

class SonosResponse {
  const SonosResponse({required this.status, required this.speakers});

  final SonosStatus status;
  final List<SonosSpeaker> speakers;

  factory SonosResponse.fromJson(Map<String, dynamic> json) {
    return SonosResponse(
      status: SonosStatus.fromJson(json['status'] as Map<String, dynamic>),
      speakers: (json['speakers'] as List<dynamic>? ?? [])
          .map(
            (speaker) => SonosSpeaker.fromJson(speaker as Map<String, dynamic>),
          )
          .toList(),
    );
  }
}
