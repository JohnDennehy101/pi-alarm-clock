import 'package:flutter/material.dart';

class AlarmNameField extends StatelessWidget {
  const AlarmNameField({
    super.key,
    required this.initialValue,
    required this.onChanged,
    this.validator,
  });

  final String initialValue;
  final ValueChanged<String> onChanged;
  final FormFieldValidator<String>? validator;

  @override
  Widget build(BuildContext context) {
    return TextFormField(
      initialValue: initialValue,
      textCapitalization: TextCapitalization.sentences,
      decoration: InputDecoration(
        labelText: 'Alarm name',
        hintText: 'e.g. Morning alarm',
        prefixIcon: const Icon(Icons.label_outline),
        border: OutlineInputBorder(borderRadius: BorderRadius.circular(16)),
      ),
      onChanged: onChanged,
      validator: validator,
    );
  }
}
