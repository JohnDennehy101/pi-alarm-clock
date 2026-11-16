import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/application/device_options_controller.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/presentation/widgets/alarm_device_section.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/presentation/widgets/alarm_name_field.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/presentation/widgets/alarm_save_button.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/presentation/widgets/alarm_schedule_card.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/presentation/widgets/alarm_time_card.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/presentation/widgets/hue_settings_sheet.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/presentation/widgets/sonos_settings_sheet.dart';
import 'package:provider/provider.dart';

import 'package:pi_alarm_clock_mobile/features/alarms/application/alarm_controller.dart';
import 'package:pi_alarm_clock_mobile/features/alarms/application/alarm_editor_controller.dart';

class AlarmFormScreen extends StatefulWidget {
  const AlarmFormScreen({super.key, this.alarmId});

  final String? alarmId;

  @override
  State<AlarmFormScreen> createState() => _AlarmFormScreenState();
}

class _AlarmFormScreenState extends State<AlarmFormScreen> {
  final _formKey = GlobalKey<FormState>();

  bool _isSaving = false;

  bool get _isEditing => widget.alarmId != null;

  String get _sonosSummary {
    final editor = context.read<AlarmEditorController>();

    if (editor.selectedSonosGroupId == null) {
      return 'Not configured';
    }

    final controller = context.watch<DeviceOptionsController>();
    final matchingGroups = controller.sonosGroups.where(
      (group) => group.id == editor.selectedSonosGroupId,
    );

    final groupName = matchingGroups.isEmpty
        ? 'Sonos group selected'
        : matchingGroups.first.name;

    return editor.selectedRadioName == null
        ? groupName
        : '$groupName · ${editor.selectedRadioName}';
  }

  String get _hueSummary {
    final editor = context.read<AlarmEditorController>();

    if (!editor.hueEnabled) {
      return 'Disabled';
    }

    final count = editor.selectedHueLightIds.length;
    return '$count ${count == 1 ? 'light' : 'lights'} selected';
  }

  String get _scheduleDescription {
    final editor = context.read<AlarmEditorController>();

    if (editor.isOneOff) {
      final date = editor.date;
      if (date == null) return 'Choose a date';
      return 'Once · ${MaterialLocalizations.of(context).formatMediumDate(date)}';
    }

    final selectedDays = editor.selectedDays;

    if (selectedDays.isEmpty) return 'No days selected';
    if (selectedDays.length == 7) return 'Every day';

    const weekdays = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday'];
    if (selectedDays.length == 5 && weekdays.every(selectedDays.contains)) {
      return 'Every weekday';
    }

    if (selectedDays.length == 2 &&
        selectedDays.contains('saturday') &&
        selectedDays.contains('sunday')) {
      return 'Weekends';
    }

    const names = {
      'monday': 'Mon',
      'tuesday': 'Tue',
      'wednesday': 'Wed',
      'thursday': 'Thu',
      'friday': 'Fri',
      'saturday': 'Sat',
      'sunday': 'Sun',
    };

    return AlarmEditorController.weekdays
        .where(selectedDays.contains)
        .map((day) => names[day]!)
        .join(', ');
  }

  Future<void> _selectTime() async {
    final editor = context.read<AlarmEditorController>();

    final selected = await showTimePicker(
      context: context,
      initialTime: TimeOfDay(hour: editor.hour, minute: editor.minute),
    );

    if (selected != null && mounted) {
      editor.setTime(selected.hour, selected.minute);
    }
  }

  Future<void> _selectDate() async {
    final editor = context.read<AlarmEditorController>();
    final today = DateUtils.dateOnly(DateTime.now());

    final selected = await showDatePicker(
      context: context,
      initialDate: editor.date ?? today,
      firstDate: today,
      lastDate: DateTime.now().add(const Duration(days: 365 * 5)),
    );

    if (selected != null && mounted) {
      editor.setDate(DateUtils.dateOnly(selected));
    }
  }

  Future<void> _save() async {
    if (!_formKey.currentState!.validate()) return;

    final editor = context.read<AlarmEditorController>();
    final validationError = editor.validate();

    if (validationError != null) {
      _showMessage(validationError);
      return;
    }

    final alarm = editor.buildAlarm();

    setState(() => _isSaving = true);

    final controller = context.read<AlarmController>();
    final success = _isEditing
        ? await controller.update(alarm)
        : await controller.create(alarm);

    if (!mounted) return;

    setState(() => _isSaving = false);

    if (success) {
      context.pop();
    } else {
      _showMessage(controller.error ?? 'Could not save alarm.');
    }
  }

  void _showMessage(String message) {
    ScaffoldMessenger.of(context)
        .showSnackBar(SnackBar(content: Text(message)));
  }

  @override
  Widget build(BuildContext context) {
    final editor = context.watch<AlarmEditorController>();

    if (editor.isLoading) {
      return Scaffold(
        appBar: AppBar(title: const Text('Edit alarm')),
        body: const Center(child: CircularProgressIndicator()),
      );
    }

    if (_isEditing && editor.notFound) {
      return Scaffold(
        appBar: AppBar(title: const Text('Edit alarm')),
        body: Center(
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(Icons.alarm_off_outlined, size: 40),
              const SizedBox(height: 12),
              const Text('Could not find this alarm.'),
              const SizedBox(height: 12),
              FilledButton(
                onPressed: () => context.pop(),
                child: const Text('Back to alarms'),
              ),
            ],
          ),
        ),
      );
    }

    return Scaffold(
      appBar: AppBar(
        title: Text(_isEditing ? 'Edit alarm' : 'New alarm'),
        actions: [
          if (_isEditing)
            Padding(
              padding: const EdgeInsets.only(right: 16),
              child: Center(
                child: Text(
                  'EDITING',
                  style: Theme.of(context).textTheme.labelSmall?.copyWith(
                    letterSpacing: 1.2,
                    color: Theme.of(context).colorScheme.primary,
                    fontWeight: FontWeight.bold,
                  ),
                ),
              ),
            ),
        ],
      ),
      body: Form(
        key: _formKey,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
          children: [
            AlarmTimeCard(
              time: TimeOfDay(hour: editor.hour, minute: editor.minute),
              enabled: editor.enabled,
              onTimeTap: _selectTime,
              onEnabledChanged: editor.setEnabled,
            ),
            const SizedBox(height: 20),
            AlarmNameField(
              key: ValueKey(editor.name),
              initialValue: editor.name,
              onChanged: editor.setName,
              validator: (value) {
                if (value == null || value.trim().isEmpty) {
                  return 'Enter an alarm name.';
                }
                return null;
              },
            ),
            const SizedBox(height: 20),
            AlarmScheduleCard(
              isOneOff: editor.isOneOff,
              selectedDays: editor.selectedDays,
              date: editor.date,
              scheduleDescription: _scheduleDescription,
              onToggleOneOff: editor.setOneOff,
              onDayToggle: editor.toggleDay,
              onDateTap: _selectDate,
            ),
            const SizedBox(height: 20),
            AlarmDeviceSection(
              sonosSummary: _sonosSummary,
              hueSummary: _hueSummary,
              onSonosTap: _showSonosSettings,
              onHueTap: _showHueSettings,
            ),
          ],
        ),
      ),
      bottomNavigationBar: AlarmSaveButton(
        isSaving: _isSaving,
        label: _isEditing ? 'Save changes' : 'Create alarm',
        onPressed: _save,
      ),
    );
  }

  Future<void> _showSonosSettings() async {
    final controller = context.read<DeviceOptionsController>();

    await controller.loadSonosGroups();

    if (!mounted) return;

    final selectedGroupId = context
        .read<AlarmEditorController>()
        .selectedSonosGroupId;
    if (selectedGroupId != null) {
      await controller.loadSonosRadioStations(selectedGroupId);
    }

    if (!mounted) return;

    final editor = context.read<AlarmEditorController>();

    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      showDragHandle: true,
      builder: (sheetContext) {
        return ChangeNotifierProvider.value(
          value: editor,
          child: const SonosSettingsSheet(),
        );
      },
    );
  }

  Future<void> _showHueSettings() async {
    final controller = context.read<DeviceOptionsController>();

    await controller.loadHueLights();

    if (!mounted) return;

    final editor = context.read<AlarmEditorController>();

    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      showDragHandle: true,
      builder: (sheetContext) {
        return ChangeNotifierProvider.value(
          value: editor,
          child: const HueSettingsSheet(),
        );
      },
    );
  }
}
