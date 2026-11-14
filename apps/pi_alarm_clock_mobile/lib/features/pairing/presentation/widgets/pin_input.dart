import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

class PinInput extends StatefulWidget {
  final int length;
  final bool enabled;
  final ValueChanged<String> onChanged;

  const PinInput({
    super.key,
    this.length = 6,
    this.enabled = true,
    required this.onChanged,
  });

  @override
  State<PinInput> createState() => _PinInputState();
}

class _PinInputState extends State<PinInput> {
  late final List<TextEditingController> _controllers;
  late final List<FocusNode> _focusNodes;

  @override
  void initState() {
    super.initState();

    _controllers = List.generate(widget.length, (_) => TextEditingController());

    _focusNodes = List.generate(widget.length, (_) => FocusNode());
  }

  @override
  void dispose() {
    for (final controller in _controllers) {
      controller.dispose();
    }

    for (final focusNode in _focusNodes) {
      focusNode.dispose();
    }

    super.dispose();
  }

  String get _pin => _controllers.map((controller) => controller.text).join();

  void _handleChanged(int index, String value) {
    final digits = value.replaceAll(RegExp(r'\D'), '');

    if (digits.length > 1) {
      for (var i = index; i < widget.length; i++) {
        _controllers[i].clear();
      }

      final availableDigits = digits.length < widget.length - index
          ? digits.length
          : widget.length - index;

      for (var i = 0; i < availableDigits; i++) {
        _controllers[index + i].text = digits[i];
      }

      final nextIndex = index + availableDigits;

      if (nextIndex < widget.length) {
        _focusNodes[nextIndex].requestFocus();
      } else {
        _focusNodes.last.unfocus();
      }
    } else if (digits.isNotEmpty) {
      _controllers[index].text = digits[0];

      if (index < widget.length - 1) {
        _focusNodes[index + 1].requestFocus();
      } else {
        _focusNodes[index].unfocus();
      }
    } else {
      _controllers[index].clear();

      if (index > 0) {
        _focusNodes[index - 1].requestFocus();
      }
    }

    widget.onChanged(_pin);
  }

  Widget _buildInput(int index) {
    return SizedBox(
      width: 42,
      height: 72,
      child: TextField(
        controller: _controllers[index],
        focusNode: _focusNodes[index],
        enabled: widget.enabled,
        autofocus: index == 0,
        keyboardType: TextInputType.number,
        textAlign: TextAlign.center,
        textAlignVertical: TextAlignVertical.center,
        maxLength: 1,
        style: const TextStyle(
          fontSize: 24,
          fontWeight: FontWeight.bold,
          color: Colors.black,
          height: 1.0,
        ),
        inputFormatters: [FilteringTextInputFormatter.digitsOnly],
        decoration: const InputDecoration(
          counterText: '',
          contentPadding: EdgeInsets.symmetric(vertical: 12),
          isDense: true,
          border: OutlineInputBorder(),
          focusedBorder: OutlineInputBorder(
            borderSide: BorderSide(color: Colors.deepPurple, width: 2),
          ),
        ),
        cursorColor: Colors.deepPurple,
        onChanged: (value) => _handleChanged(index, value),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: List.generate(
        widget.length,
        (index) => Padding(
          padding: const EdgeInsets.symmetric(horizontal: 2),
          child: _buildInput(index),
        ),
      ),
    );
  }
}
