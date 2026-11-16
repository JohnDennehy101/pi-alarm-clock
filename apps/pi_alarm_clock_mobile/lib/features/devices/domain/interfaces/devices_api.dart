import 'package:pi_alarm_clock_mobile/features/devices/domain/models/hue_lights_response.dart';
import 'package:pi_alarm_clock_mobile/features/devices/domain/models/radio_stations_response.dart';
import 'package:pi_alarm_clock_mobile/features/home/domain/models/sonos_groups_response.dart';

abstract interface class DevicesApi {
  Future<SonosGroupsResponse> getSonosGroups();

  Future<HueLightsResponse> getHueLights();

  Future<RadioStationsResponse> getSonosRadioStations(String groupId);
}
