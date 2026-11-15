import 'package:flutter/material.dart';

class AlarmFormScreen extends StatelessWidget {
  const AlarmFormScreen({super.key, this.alarmId});

  final String? alarmId;

  @override
  Widget build(BuildContext context) {
    final isEditing = alarmId != null;

    return Scaffold(
      appBar: AppBar(title: Text(isEditing ? 'Edit alarm' : 'New alarm')),
      body: Center(
        child: Text(
          isEditing ? 'Edit alarm: $alarmId' : 'Alarm form coming soon',
        ),
      ),
    );
  }
}
