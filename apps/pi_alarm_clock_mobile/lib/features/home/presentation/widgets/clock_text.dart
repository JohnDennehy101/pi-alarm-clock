import 'dart:async';

import 'package:flutter/material.dart';

class ClockText extends StatefulWidget {
  const ClockText({super.key, this.style});

  final TextStyle? style;

  @override
  State<ClockText> createState() => _ClockTextState();
}

class _ClockTextState extends State<ClockText> {
  DateTime _now = DateTime.now();
  Timer? _clockTimer;

  @override
  void initState() {
    super.initState();

    _clockTimer = Timer.periodic(const Duration(seconds: 1), (_) {
      if (!mounted) return;

      setState(() {
        _now = DateTime.now();
      });
    });
  }

  @override
  void dispose() {
    _clockTimer?.cancel();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Text(
      TimeOfDay.fromDateTime(_now).format(context),
      style: widget.style,
    );
  }
}
