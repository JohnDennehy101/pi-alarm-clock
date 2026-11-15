class HealthStatus {
  const HealthStatus({required this.status});

  final String status;

  factory HealthStatus.fromJson(Map<String, dynamic> json) {
    return HealthStatus(status: json['status'] as String);
  }
}
