import 'radio_station.dart';

class RadioStationsResponse {
  const RadioStationsResponse({required this.stations});

  final List<RadioStation> stations;

  List<RadioStation> get playableStations =>
      stations.where((station) => station.uri.isNotEmpty).toList();

  factory RadioStationsResponse.fromJson(Map<String, dynamic> json) {
    return RadioStationsResponse(
      stations: (json['stations'] as List<dynamic>)
          .map(
            (station) => RadioStation.fromJson(station as Map<String, dynamic>),
          )
          .toList(),
    );
  }
}
