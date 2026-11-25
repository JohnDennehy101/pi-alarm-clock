package app

import (
	"testing"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/display"
)

// TestTimelineWalkAlarmBoardSession walks the board cast refresh loop the
// way ChromecastService.refreshBoardCast does — start at the image time,
// repeatedly sleep to timeline.NextBoundary and render timeline.At — for a
// session shaped like an alarm: preset order with per-view overrides, a
// long one-shot intro and a display window much longer than the intro.
//
// Regression guard for the alarm board behaviour: the intro plays once,
// the loop keeps cycling across the intro's end until the window closes,
// and refresh ticks land where they were configured.
func TestTimelineWalkAlarmBoardSession(t *testing.T) {
	catalog := DefaultTimelineCatalog()

	entries, err := entriesFromStrings([]string{
		"bus.schedule:1m:1m",
		"bus.map:1m:1m",
		"bus.progress:1m:15s",
	})
	if err != nil {
		t.Fatalf("preset order: %v", err)
	}

	if err := catalog.putPreset("morning-commute", entries); err != nil {
		t.Fatalf("preset: %v", err)
	}

	timeline, err := catalog.ResolveForCast("morning-commute", "weather:10m")
	if err != nil {
		t.Fatalf("ResolveForCast: %v", err)
	}

	start := time.Date(2026, 11, 25, 7, 1, 0, 0, time.UTC)
	deadline := start.Add(45 * time.Minute)

	type frame struct {
		minutes float64
		panel   display.Panel
		tick    int
	}

	var seq []frame

	offset := time.Duration(0)
	for {
		now := start.Add(offset)
		boundary := timeline.NextBoundary(start, now)
		if !boundary.Before(deadline) {
			break
		}

		offset = boundary.Sub(start)
		slot := timeline.At(start, start.Add(offset))
		seq = append(seq, frame{
			minutes: float64(offset) / float64(time.Minute),
			panel:   slot.Panel,
			tick:    slot.Tick,
		})

		if offset >= 45*time.Minute {
			break
		}
	}

	want := []frame{
		{1, display.PanelWeather, 1},
		{2, display.PanelWeather, 2},
		{3, display.PanelWeather, 3},
		{4, display.PanelWeather, 4},
		{5, display.PanelWeather, 5},
		{6, display.PanelWeather, 6},
		{7, display.PanelWeather, 7},
		{8, display.PanelWeather, 8},
		{9, display.PanelWeather, 9},
		{10, display.PanelBus, 0},         // intro over: schedule
		{11, display.PanelBusMap, 0},     // map
		{12, display.PanelBusProgress, 0}, // progress, refreshing every 15s
		{12.25, display.PanelBusProgress, 1},
		{12.5, display.PanelBusProgress, 2},
		{12.75, display.PanelBusProgress, 3},
		{13, display.PanelBus, 0}, // back to schedule
		{14, display.PanelBusMap, 0},
		{15, display.PanelBusProgress, 0},
	}

	if len(seq) < len(want) {
		t.Fatalf("walk produced %d frames, want at least %d", len(seq), len(want))
	}

	for i, expect := range want {
		got := seq[i]
		if got.panel != expect.panel || got.tick != expect.tick || got.minutes != expect.minutes {
			t.Errorf("frame %d = +%g min %s tick %d, want +%g min %s tick %d",
				i,
				got.minutes, got.panel, got.tick,
				expect.minutes, expect.panel, expect.tick,
			)
		}
	}

	last := seq[len(seq)-1]
	// The walk must run until the display window ends, not until some
	// boundary inside it.
	if last.minutes < 44 {
		t.Errorf(
			"walk stopped at +%g min; the loop should keep cycling until the display window (~45 min)",
			last.minutes,
		)
	}
}
