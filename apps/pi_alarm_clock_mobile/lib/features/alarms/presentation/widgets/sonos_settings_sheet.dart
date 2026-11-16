import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../application/alarm_editor_controller.dart';
import '../../application/device_options_controller.dart';

class SonosSettingsSheet extends StatelessWidget {
  const SonosSettingsSheet({super.key});

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
                'Sonos settings',
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
    if (controller.isLoadingSonosGroups) {
      return const Center(
        child: Padding(
          padding: EdgeInsets.all(16),
          child: CircularProgressIndicator(),
        ),
      );
    }

    if (controller.sonosError != null) {
      return Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(controller.sonosError!),
          TextButton(
            onPressed: () => controller.loadSonosGroups(force: true),
            child: const Text('Retry'),
          ),
        ],
      );
    }

    if (controller.sonosGroups.isEmpty) {
      return const Padding(
        padding: EdgeInsets.all(16),
        child: Text('No Sonos groups found.'),
      );
    }

    final selectedGroupId = editor.selectedSonosGroupId;

    final selectedGroupExists = controller.sonosGroups.any(
      (group) => group.id == selectedGroupId,
    );

    final radioStations = selectedGroupId == null
        ? []
        : controller.radioStationsForGroup(selectedGroupId);

    final isLoadingStations =
        selectedGroupId != null &&
        controller.isLoadingRadioStations(selectedGroupId);

    final radioError = selectedGroupId == null
        ? null
        : controller.radioStationsError(selectedGroupId);

    final selectedStationExists = radioStations.any(
      (station) => station.uri == editor.selectedRadioUri,
    );

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Sonos group
        DropdownButtonFormField<String>(
          initialValue: selectedGroupExists ? selectedGroupId : null,
          decoration: const InputDecoration(
            labelText: 'Sonos speaker group',
            border: OutlineInputBorder(),
          ),
          items: controller.sonosGroups.map((group) {
            return DropdownMenuItem<String>(
              value: group.id,
              child: Text(group.name),
            );
          }).toList(),
          onChanged: (groupId) {
            editor.setSonosGroup(groupId);

            if (groupId != null) {
              controller.loadSonosRadioStations(groupId);
            }
          },
        ),

        const SizedBox(height: 16),

        // Radio station
        if (selectedGroupId == null)
          const Text('Select a Sonos group to load its radio stations.')
        else if (isLoadingStations)
          const Center(child: CircularProgressIndicator())
        else if (radioError != null)
          Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(radioError),
              TextButton(
                onPressed: () {
                  controller.loadSonosRadioStations(
                    selectedGroupId,
                    force: true,
                  );
                },
                child: const Text('Retry'),
              ),
            ],
          )
        else if (radioStations.isEmpty)
          const Text('No playable radio stations found.')
        else
          DropdownButtonFormField<String>(
            initialValue: selectedStationExists
                ? editor.selectedRadioUri
                : null,
            decoration: const InputDecoration(
              labelText: 'Radio station',
              border: OutlineInputBorder(),
            ),
            items: radioStations.map((station) {
              return DropdownMenuItem<String>(
                value: station.uri,
                child: Text(station.title),
              );
            }).toList(),
            onChanged: (uri) {
              if (uri == null) return;

              final station = radioStations.firstWhere(
                (station) => station.uri == uri,
              );

              editor.setRadioStation(station.uri, station.title);
            },
          ),

        const SizedBox(height: 24),

        // Volume
        Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            const Text(
              'Alarm volume',
              style: TextStyle(fontWeight: FontWeight.w500),
            ),
            Text('${editor.sonosVolume}%'),
          ],
        ),
        Slider(
          value: editor.sonosVolume.toDouble(),
          min: 0,
          max: 100,
          divisions: 100,
          label: '${editor.sonosVolume}%',
          onChanged: (value) => editor.setSonosVolume(value.round()),
        ),

        // Gradual volume
        SwitchListTile(
          contentPadding: EdgeInsets.zero,
          title: const Text('Gradually increase volume'),
          value: editor.gradualVolume,
          onChanged: editor.setGradualVolume,
        ),

        if (editor.gradualVolume)
          DropdownButtonFormField<int>(
            initialValue: editor.volumeDuration,
            decoration: const InputDecoration(
              labelText: 'Volume increase duration',
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
                editor.setVolumeDuration(value);
              }
            },
          ),
      ],
    );
  }
}
