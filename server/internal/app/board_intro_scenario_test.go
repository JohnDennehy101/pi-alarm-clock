package app

import (
	"path/filepath"
	"testing"
	"time"
)

func int64Ptr(v int64) *int64 { return &v }

// TestBoardServiceWeatherIntroScenario pins the intended default behaviour:
// weather plays once as a five minute intro when the board session starts,
// then the bus loop (schedule, map, progress) repeats for the rest of the
// session. The progress view rotates routes every 12 seconds.
func TestBoardServiceWeatherIntroScenario(t *testing.T) {
	path := filepath.Join(t.TempDir(), "board-timeline.yml")
	board := NewBoardService("", nil, nil, BoardTimelineConfig{
		Timeline:     nil,
		Catalog:      nil,
		TimelinePath: path,
	})

	if err := board.UpdateViewSettings("weather", int64Ptr(300), int64Ptr(60)); err != nil {
		t.Fatalf("weather: %v", err)
	}
	if err := board.UpdateViewSettings("bus.schedule", int64Ptr(60), int64Ptr(60)); err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if err := board.UpdateViewSettings("bus.map", int64Ptr(30), int64Ptr(15)); err != nil {
		t.Fatalf("map: %v", err)
	}
	if err := board.UpdateViewSettings("bus.progress", int64Ptr(60), int64Ptr(12)); err != nil {
		t.Fatalf("progress: %v", err)
	}

	if err := board.SetTimelineIntro([]string{"weather"}); err != nil {
		t.Fatalf("SetTimelineIntro: %v", err)
	}
	if err := board.SetTimelineOrderViews([]string{"bus.schedule", "bus.map", "bus.progress"}); err != nil {
		t.Fatalf("SetTimelineOrderViews: %v", err)
	}

	timeline, err := board.ResolveTimeline("")
	if err != nil {
		t.Fatalf("ResolveTimeline: %v", err)
	}

	// Steps: 0 = weather intro, 1 = bus.schedule, 2 = bus.map, 3 = bus.progress.
	start := time.Date(2026, 3, 2, 7, 0, 0, 0, time.UTC)

	tests := []struct {
		offset   time.Duration
		wantStep int
		wantTick int
	}{
		{0, 0, 0},                              // weather intro starts
		{2 * time.Minute, 0, 2},                // intro refresh tick 2
		{4*time.Minute + 15*time.Second, 0, 4}, // intro tick 4
		{5 * time.Minute, 1, 0},                // intro consumed: schedule
		{5*time.Minute + 15*time.Second, 1, 0}, // still schedule
		{6 * time.Minute, 2, 0},                // map
		{6*time.Minute + 30*time.Second, 3, 0}, // progress, route 1
		{6*time.Minute + 44*time.Second, 3, 1}, // progress, route 2 (12s slots)
		{6*time.Minute + 58*time.Second, 3, 2}, // progress, route 3
		{7*time.Minute + 30*time.Second, 1, 0}, // loop wrap: schedule, never weather again
		{12*time.Minute + 6*time.Second, 3, 3}, // several wraps later
	}

	for i := range tests {
		test := tests[i]

		t.Run(test.offset.String(), func(t *testing.T) {
			slot := timeline.At(start, start.Add(test.offset))

			if slot.Step != test.wantStep || slot.Tick != test.wantTick {
				t.Fatalf(
					"At(+%s) = {step %d tick %d}, want {step %d tick %d}",
					test.offset, slot.Step, slot.Tick, test.wantStep, test.wantTick,
				)
			}
		})
	}
}
