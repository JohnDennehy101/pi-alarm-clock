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
