import 'package:pi_alarm_clock_mobile/features/home/domain/models/health_status.dart';
import 'package:pi_alarm_clock_mobile/features/home/domain/models/hue_status.dart';
import 'package:pi_alarm_clock_mobile/features/home/domain/models/sonos_response.dart';

abstract interface class HomeApi {
  Future<HealthStatus> getHealth();

  Future<SonosResponse> getSonos();

  Future<HueStatus> getHue();
}
