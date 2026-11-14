import 'dart:math';

import 'package:flutter/material.dart';

String _groupHex(String hex) {
  final groups = <String>[];
  for (var i = 0; i < hex.length; i += 8) {
    groups.add(hex.substring(i, min(i + 8, hex.length)).toUpperCase());
  }
  return groups.join('  ');
}

/// Trust-on-first-use verification: shows the presented certificate's
/// SHA-256 fingerprint for the user to compare against the Pi's startup
/// log line "Cert SHA-256: ...". Returns true only if the user explicitly
/// confirms.
Future<bool> showVerifyServerDialog(
  BuildContext context, {
  required String fingerprint,
  required bool identityChanged,
}) async {
  final confirmed = await showDialog<bool>(
    context: context,
    barrierDismissible: false,
    builder: (dialogContext) => AlertDialog(
      icon: Icon(
        identityChanged ? Icons.gpp_maybe : Icons.fingerprint,
        size: 48,
        color: identityChanged ? Colors.orange : Colors.deepPurple,
      ),
      title: Text(
        identityChanged
            ? 'Server identity changed'
            : 'New server detected',
      ),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (identityChanged) ...[
            const Text(
              'The certificate this server presented does NOT match the one '
              'previously trusted. This happens after a Pi certificate '
              'regeneration — but it can also mean someone is impersonating '
              'your Pi. Only continue if you personally re-generated it.',
            ),
            const SizedBox(height: 16),
          ] else
            const Text(
              'First contact with this server. Compare this fingerprint '
              'with the "Cert SHA-256" line in the Raspberry Pi startup log:',
            ),
          const SizedBox(height: 16),
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: Colors.grey.shade100,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(color: Colors.grey.shade300),
            ),
            child: SelectableText(
              _groupHex(fingerprint),
              style: const TextStyle(
                fontFamily: 'monospace',
                fontSize: 13,
                fontWeight: FontWeight.w600,
              ),
            ),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(dialogContext, false),
          child: const Text('Refuse'),
        ),
        FilledButton(
          onPressed: () => Navigator.pop(dialogContext, true),
          child: const Text('Verified, Trust & Connect'),
        ),
      ],
    ),
  );

  return confirmed ?? false;
}
