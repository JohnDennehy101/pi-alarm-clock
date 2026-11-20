package bus

import (
	"encoding/binary"
	"testing"
	"time"
)

func appendStringField(msg []byte, fieldNumber int, value string) []byte {
	msg = binary.AppendUvarint(msg, uint64(fieldNumber)<<3|2)
	msg = binary.AppendUvarint(msg, uint64(len(value)))

	return append(msg, value...)
}

func appendMessageField(msg []byte, fieldNumber int, value []byte) []byte {
	msg = binary.AppendUvarint(msg, uint64(fieldNumber)<<3|2)
	msg = binary.AppendUvarint(msg, uint64(len(value)))

	return append(msg, value...)
}

func appendVarintField(msg []byte, fieldNumber int, value uint64) []byte {
	msg = binary.AppendUvarint(msg, uint64(fieldNumber)<<3|0)

	return binary.AppendUvarint(msg, value)
}

func stopTimeUpdate(stopID string, at uint64, fieldNumber int) []byte {
	update := appendStringField(nil, 4, stopID)

	event := appendVarintField(nil, 2, at)
	update = appendMessageField(update, fieldNumber, event)

	return update
}

func TestParseRealtimeFeed(t *testing.T) {
	tripDescriptor := appendStringField(nil, 1, "trip-7")

	tripUpdate := appendMessageField(nil, 1, tripDescriptor)
	tripUpdate = appendMessageField(tripUpdate, 2, stopTimeUpdate("stop-1", 1700000000, 3))

	// Skipped stop: present in the feed, ignored by the parser.
	skipped := stopTimeUpdate("stop-2", 1700000100, 2)
	skipped = appendVarintField(skipped, 5, 1)
	tripUpdate = appendMessageField(tripUpdate, 2, skipped)

	// Departure missing, falls back to arrival time.
	tripUpdate = appendMessageField(tripUpdate, 2, stopTimeUpdate("stop-3", 1700000200, 2))

	entity := appendMessageField(nil, 3, tripUpdate)

	feed := appendMessageField(nil, 2, entity)

	updates, err := parseRealtimeFeed(feed)
	if err != nil {
		t.Fatalf("parseRealtimeFeed: %v", err)
	}

	expected := []RealtimeUpdate{
		{TripID: "trip-7", StopID: "stop-1", Time: time.Unix(1700000000, 0)},
		{TripID: "trip-7", StopID: "stop-3", Time: time.Unix(1700000200, 0)},
	}

	if len(updates) != len(expected) {
		t.Fatalf("got %d updates, want %d: %+v", len(updates), len(expected), updates)
	}

	for i := range expected {
		if updates[i] != expected[i] {
			t.Errorf("update %d = %+v, want %+v", i, updates[i], expected[i])
		}
	}
}

func TestParseRealtimeFeedTruncated(t *testing.T) {
	if _, err := parseRealtimeFeed([]byte{0x12, 0x7f}); err == nil {
		t.Fatal("expected error for truncated payload, got nil")
	}
}
