package app

import (
	"testing"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/display"
)

func TestParseBoardSchedule(t *testing.T) {
	schedule, err := ParseBoardSchedule("weather:5,bus:15")
	if err != nil {
		t.Fatalf("ParseBoardSchedule: %v", err)
	}

	if len(schedule) != 2 {
		t.Fatalf("got %d segments, want 2", len(schedule))
	}

	if schedule[0].Panel != display.PanelWeather || schedule[0].Minutes != 5 {
		t.Errorf("segment 0 = %+v, want weather 5", schedule[0])
	}

	if schedule[1].Panel != display.PanelBus || schedule[1].Minutes != 15 {
		t.Errorf("segment 1 = %+v, want bus 15", schedule[1])
	}
}

func TestParseBoardScheduleDefault(t *testing.T) {
	schedule, err := ParseBoardSchedule("  ")
	if err != nil {
		t.Fatalf("ParseBoardSchedule: %v", err)
	}

	if len(schedule) != 2 {
		t.Fatalf("got %d segments, want the default schedule", len(schedule))
	}
}

func TestParseBoardScheduleInvalid(t *testing.T) {
	for _, spec := range []string{"weather", "weather:x", "unknown:5", "weather:0"} {
		if _, err := ParseBoardSchedule(spec); err == nil {
			t.Errorf("ParseBoardSchedule(%q) = nil error, want error", spec)
		}
	}
}

func TestParseBoardScheduleNestedBusViews(t *testing.T) {
	schedule, err := ParseBoardSchedule("bus:{map:3,schedule:2},weather:1")
	if err != nil {
		t.Fatalf("ParseBoardSchedule: %v", err)
	}

	want := []struct {
		panel   display.Panel
		minutes int
	}{
		{display.PanelBusMap, 3},
		{display.PanelBus, 2},
		{display.PanelWeather, 1},
	}

	if len(schedule) != len(want) {
		t.Fatalf("got %d segments, want %d", len(schedule), len(want))
	}

	for i, expected := range want {
		if schedule[i].Panel != expected.panel ||
			schedule[i].Minutes != expected.minutes {
			t.Errorf(
				"segment %d = %s:%d, want %s:%d",
				i,
				schedule[i].Panel,
				schedule[i].Minutes,
				expected.panel,
				expected.minutes,
			)
		}
	}
}

func TestParseBoardScheduleMapOnlyView(t *testing.T) {
	schedule, err := ParseBoardSchedule("bus:{map:5}")
	if err != nil {
		t.Fatalf("ParseBoardSchedule: %v", err)
	}

	if len(schedule) != 1 || schedule[0].Panel != display.PanelBusMap {
		t.Fatalf("schedule = %+v, want a single map segment", schedule)
	}
}

func TestParseBoardScheduleAllBusViews(t *testing.T) {
	schedule, err := ParseBoardSchedule(
		"bus:{map:2,schedule:2,progress:2},weather:1",
	)
	if err != nil {
		t.Fatalf("ParseBoardSchedule: %v", err)
	}

	want := []struct {
		panel   display.Panel
		minutes int
	}{
		{display.PanelBusMap, 2},
		{display.PanelBus, 2},
		{display.PanelBusProgress, 2},
		{display.PanelWeather, 1},
	}

	if len(schedule) != len(want) {
		t.Fatalf("got %d segments, want %d", len(schedule), len(want))
	}

	for i := range want {
		if schedule[i].Panel != want[i].panel ||
			schedule[i].Minutes != want[i].minutes {
			t.Errorf(
				"segment %d = %s:%d, want %s:%d",
				i,
				schedule[i].Panel,
				schedule[i].Minutes,
				want[i].panel,
				want[i].minutes,
			)
		}
	}
}

func TestParseBoardScheduleInvalidViewGroups(t *testing.T) {
	for _, spec := range []string{
		"bus:{map:2,schedule",        // unterminated group
		"weather:{bus:2}",            // groups only supported for bus
		"bus:{}",                     // empty group
		"bus:{unknown:2}",            // unknown view
		"bus:{map:x},weather:1",      // non-numeric duration
		"bus:{map:0},weather:1",      // zero duration
		"bus:{map:2}bogus,weather:1", // trailing garbage in group
	} {
		if _, err := ParseBoardSchedule(spec); err == nil {
			t.Errorf("ParseBoardSchedule(%q) = nil error, want error", spec)
		}
	}
}

func TestSplitScheduleEntries(t *testing.T) {
	entries := splitScheduleEntries("bus:{map:2,schedule:3},weather:1")

	want := []string{"bus:{map:2,schedule:3}", "weather:1"}

	if len(entries) != len(want) {
		t.Fatalf("got %d entries, want %d", len(entries), len(want))
	}

	for i, entry := range want {
		if entries[i] != entry {
			t.Errorf("entry %d = %q, want %q", i, entries[i], entry)
		}
	}
}

func TestBoardSchedulePanelAt(t *testing.T) {
	schedule, err := ParseBoardSchedule("weather:5,bus:15")
	if err != nil {
		t.Fatalf("ParseBoardSchedule: %v", err)
	}

	tests := []struct {
		minute int
		want   display.Panel
	}{
		{0, display.PanelWeather},
		{4, display.PanelWeather},
		{5, display.PanelBus},
		{19, display.PanelBus},
		{20, display.PanelWeather}, // cycles
		{25, display.PanelBus},
	}

	for _, test := range tests {
		if got := schedule.PanelAt(test.minute); got != test.want {
			t.Errorf("PanelAt(%d) = %s, want %s", test.minute, got, test.want)
		}
	}

	if got := BoardSchedule(nil).PanelAt(7); got != display.PanelWeather {
		t.Errorf("PanelAt on empty schedule = %s, want weather", got)
	}
}
