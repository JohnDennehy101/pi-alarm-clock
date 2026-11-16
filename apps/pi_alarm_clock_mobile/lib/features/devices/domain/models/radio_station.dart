class RadioStation {
  const RadioStation({
    required this.title,
    required this.uri,
    required this.metadata,
  });

  final String title;
  final String uri;
  final String metadata;

  factory RadioStation.fromJson(Map<String, dynamic> json) {
    return RadioStation(
      title: json['Title'] as String,
      uri: json['URI'] as String,
      metadata: json['Metadata'] as String? ?? '',
    );
  }
}
