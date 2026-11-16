class Alarm {
  const Alarm({
    required this.id,
    required this.name,
    required this.time,
    required this.enabled,
    required this.days,
    this.date,
    required this.sonos,
    required this.hue,
    this.createdAt,
    this.updatedAt,
  });

  final String id;
  final String name;
  final String time;
  final bool enabled;

  final List<String> days;

  final DateTime? date;

  final SonosSettings sonos;
  final HueSettings hue;

  final DateTime? createdAt;
  final DateTime? updatedAt;

  Alarm copyWith({
    String? name,
    String? time,
    bool? enabled,
    List<String>? days,
    DateTime? date,
    SonosSettings? sonos,
    HueSettings? hue,
  }) {
    return Alarm(
      id: id,
      name: name ?? this.name,
      time: time ?? this.time,
      enabled: enabled ?? this.enabled,
      days: days ?? this.days,
      date: date ?? this.date,
      sonos: sonos ?? this.sonos,
      hue: hue ?? this.hue,
      createdAt: createdAt,
      updatedAt: updatedAt,
    );
  }

  factory Alarm.fromJson(Map<String, dynamic> json) {
    return Alarm(
      id: json['id'] as String,
      name: json['name'] as String,
      time: json['time'] as String,
      enabled: json['enabled'] as bool,
      days: (json['days'] as List<dynamic>?)?.cast<String>() ?? [],
      date: json['date'] != null
          ? DateTime.parse(json['date'] as String)
          : null,
      sonos: SonosSettings.fromJson(json['sonos'] as Map<String, dynamic>),
      hue: HueSettings.fromJson(json['hue'] as Map<String, dynamic>),
      createdAt: json['createdAt'] != null
          ? DateTime.parse(json['createdAt'] as String)
          : null,
      updatedAt: json['updatedAt'] != null
          ? DateTime.parse(json['updatedAt'] as String)
          : null,
    );
  }

  Map<String, dynamic> toJson() {
    return {
      'id': id,
      'name': name,
      'time': time,
      'enabled': enabled,
      'days': days,
      'date': date == null
          ? null
          : '${date!.year.toString().padLeft(4, '0')}-'
                '${date!.month.toString().padLeft(2, '0')}-'
                '${date!.day.toString().padLeft(2, '0')}',
      'sonos': sonos.toJson(),
      'hue': hue.toJson(),
      if (createdAt != null) 'createdAt': createdAt!.toIso8601String(),
      if (updatedAt != null) 'updatedAt': updatedAt!.toIso8601String(),
    };
  }
}

class SonosSettings {
  const SonosSettings({
    required this.target,
    required this.source,
    required this.volume,
    required this.gradualVolume,
    required this.volumeDuration,
  });

  final SonosTarget target;
  final AudioSource source;
  final int volume;
  final bool gradualVolume;
  final int volumeDuration;

  factory SonosSettings.fromJson(Map<String, dynamic> json) {
    return SonosSettings(
      target: SonosTarget.fromJson(json['target'] as Map<String, dynamic>),
      source: AudioSource.fromJson(json['source'] as Map<String, dynamic>),
      volume: json['volume'] as int,
      gradualVolume: json['gradualVolume'] as bool,
      volumeDuration: json['volumeDuration'] as int,
    );
  }

  Map<String, dynamic> toJson() => {
    'target': target.toJson(),
    'source': source.toJson(),
    'volume': volume,
    'gradualVolume': gradualVolume,
    'volumeDuration': volumeDuration,
  };
}

class SonosTarget {
  const SonosTarget({required this.type, required this.id});

  final String type;
  final String id;

  factory SonosTarget.fromJson(Map<String, dynamic> json) {
    return SonosTarget(type: json['type'] as String, id: json['id'] as String);
  }

  Map<String, dynamic> toJson() => {'type': type, 'id': id};
}

class AudioSource {
  const AudioSource({required this.type, this.uri, this.name});

  final String type;
  final String? uri;
  final String? name;

  factory AudioSource.fromJson(Map<String, dynamic> json) {
    return AudioSource(
      type: json['type'] as String,
      uri: json['uri'] as String?,
      name: json['name'] as String?,
    );
  }

  Map<String, dynamic> toJson() => {
    'type': type,
    if (uri != null) 'uri': uri,
    if (name != null) 'name': name,
  };
}

class HueSettings {
  const HueSettings({
    required this.enabled,
    required this.lightIds,
    required this.startBrightness,
    required this.brightness,
    required this.gradualBrightness,
    required this.brightnessDuration,
    this.colourTemperature,
  });

  final bool enabled;
  final List<String> lightIds;
  final int startBrightness;
  final int brightness;
  final bool gradualBrightness;
  final int brightnessDuration;
  final int? colourTemperature;

  factory HueSettings.fromJson(Map<String, dynamic> json) {
    return HueSettings(
      enabled: json['enabled'] as bool,
      lightIds: (json['lightIds'] as List<dynamic>?)?.cast<String>() ?? [],
      startBrightness: json['startBrightness'] as int,
      brightness: json['brightness'] as int,
      gradualBrightness: json['gradualBrightness'] as bool,
      brightnessDuration: json['brightnessDuration'] as int,
      colourTemperature: json['colourTemperature'] as int?,
    );
  }

  Map<String, dynamic> toJson() => {
    'enabled': enabled,
    'lightIds': lightIds,
    'startBrightness': startBrightness,
    'brightness': brightness,
    'gradualBrightness': gradualBrightness,
    'brightnessDuration': brightnessDuration,
    if (colourTemperature != null) 'colourTemperature': colourTemperature,
  };
}
