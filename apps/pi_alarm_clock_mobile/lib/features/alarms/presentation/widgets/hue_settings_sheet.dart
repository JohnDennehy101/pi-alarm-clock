import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../application/alarm_editor_controller.dart';
import '../../application/device_options_controller.dart';

class HueSettingsSheet extends StatelessWidget {
  const HueSettingsSheet({super.key});

  @override
  Widget build(BuildContext context) {
    final editor = context.watch<AlarmEditorController>();
    final controller = context.watch<DeviceOptionsController>();

    return SizedBox(
      height: MediaQuery.of(context).size.height * 0.85,
      child: Column(
        children: [
          const Padding(
            padding: EdgeInsets.fromLTRB(24, 8, 24, 16),
            child: Align(
              alignment: Alignment.centerLeft,
              child: Text(
                'Philips Hue settings',
                style: TextStyle(fontSize: 20, fontWeight: FontWeight.w600),
              ),
            ),
          ),
          const Divider(height: 1),
          Expanded(
            child: SingleChildScrollView(
              padding: const EdgeInsets.all(24),
              child: _buildOptions(context, editor, controller),
            ),
          ),
          const Divider(height: 1),
          Padding(
            padding: const EdgeInsets.all(16),
            child: SizedBox(
              width: double.infinity,
              child: FilledButton(
                onPressed: () => Navigator.pop(context),
                child: const Text('Done'),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildOptions(
    BuildContext context,
    AlarmEditorController editor,
    DeviceOptionsController controller,
  ) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SwitchListTile(
          contentPadding: EdgeInsets.zero,
          title: const Text('Enable Hue lighting'),
          value: editor.hueEnabled,
          onChanged: (value) async {
            editor.setHueEnabled(value);

            if (value) {
              await controller.loadHueLights();
            }
          },
        ),

        if (editor.hueEnabled) ...[
          const SizedBox(height: 16),

          if (controller.isLoadingHueLights)
            const Center(
              child: Padding(
                padding: EdgeInsets.all(16),
                child: CircularProgressIndicator(),
              ),
            )
          else if (controller.hueError != null)
            Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(controller.hueError!),
                TextButton(
                  onPressed: () => controller.loadHueLights(force: true),
                  child: const Text('Retry'),
                ),
              ],
            )
          else if (controller.hueLights.isEmpty)
            const Text('No Hue lights found.')
          else ...[
            const Text(
              'Select lights',
              style: TextStyle(fontWeight: FontWeight.w500),
            ),
            const SizedBox(height: 8),

            ...controller.hueLights.map((light) {
              final selected = editor.selectedHueLightIds.contains(light.id);

              return CheckboxListTile(
                contentPadding: EdgeInsets.zero,
                title: Text(light.name),
                value: selected,
                onChanged: (value) {
                  editor.toggleHueLight(light.id, value ?? false);
                },
              );
            }),
          ],

          const SizedBox(height: 24),

          // Starting brightness
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              const Text(
                'Starting brightness',
                style: TextStyle(fontWeight: FontWeight.w500),
              ),
              Text('${editor.hueStartBrightness}%'),
            ],
          ),
          Slider(
            value: editor.hueStartBrightness.toDouble(),
            min: 0,
            max: 100,
            divisions: 100,
            label: '${editor.hueStartBrightness}%',
            onChanged: (value) => editor.setHueStartBrightness(value.round()),
          ),

          const SizedBox(height: 8),

          // Target brightness
          Row(
            mainAxisAlignment: MainAxisAlignment.spaceBetween,
            children: [
              const Text(
                'Target brightness',
                style: TextStyle(fontWeight: FontWeight.w500),
              ),
              Text('${editor.hueBrightness}%'),
            ],
          ),
          Slider(
            value: editor.hueBrightness.toDouble(),
            min: 0,
            max: 100,
            divisions: 100,
            label: '${editor.hueBrightness}%',
            onChanged: (value) => editor.setHueBrightness(value.round()),
          ),

          // Gradual brightness
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            title: const Text('Gradually increase brightness'),
            value: editor.gradualBrightness,
            onChanged: editor.setGradualBrightness,
          ),

          if (editor.gradualBrightness)
            DropdownButtonFormField<int>(
              initialValue: editor.brightnessDuration,
              decoration: const InputDecoration(
                labelText: 'Brightness increase duration',
                border: OutlineInputBorder(),
              ),
              items: [1, 2, 5, 10, 15, 30].map((duration) {
                return DropdownMenuItem<int>(
                  value: duration,
                  child: Text('$duration minutes'),
                );
              }).toList(),
              onChanged: (value) {
                if (value != null) {
                  editor.setBrightnessDuration(value);
                }
              },
            ),
        ],
      ],
    );
  }
}
