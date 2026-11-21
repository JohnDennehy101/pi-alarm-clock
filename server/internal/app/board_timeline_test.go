package app

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/display"
)

func TestTimelineAtBoundariesAndTicks(t *testing.T) {
	timeline := &Timeline{
		Steps: []TimelineStep{
			{Panel: display.PanelWeather, Duration: 60 * time.Second, Refresh: 60 * time.Second},
			{Panel: display.PanelBusProgress, Duration: 60 * time.Second, Refresh: 25 * time.Second},
		},
	}

	if err := timeline.validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	start := time.Date(2026, 3, 2, 7, 0, 0, 0, time.UTC)

	tests := []struct {
		offset time.Duration
		panel  display.Panel
		step   int
		tick   int
	}{
		{0, display.PanelWeather, 0, 0},
		{59 * time.Second, display.PanelWeather, 0, 0},
		{60 * time.Second, display.PanelBusProgress, 1, 0},
		{70 * time.Second, display.PanelBusProgress, 1, 0},
		{85 * time.Second, display.PanelBusProgress, 1, 1},
		{110 * time.Second, display.PanelBusProgress, 1, 2},
		{120 * time.Second, display.PanelWeather, 0, 0}, // wraps
		{181 * time.Second, display.PanelBusProgress, 1, 0},
		{205 * time.Second, display.PanelBusProgress, 1, 1},
	}

	for _, test := range tests {
		slot := timeline.At(start, start.Add(test.offset))

		if slot.Panel != test.panel || slot.Step != test.step || slot.Tick != test.tick {
			t.Errorf(
				"At(+%s) = {panel %s step %d tick %d}, want {panel %s step %d tick %d}",
				test.offset,
				slot.Panel,
				slot.Step,
				slot.Tick,
				test.panel,
				test.step,
				test.tick,
			)
		}
	}
}

func TestTimelineNextBoundary(t *testing.T) {
	timeline := Timeline{Steps: []TimelineStep{
		{Panel: display.PanelBusMap, Duration: 120 * time.Second, Refresh: 60 * time.Second},
		{Panel: display.PanelBus, Duration: 120 * time.Second, Refresh: 120 * time.Second},
		{Panel: display.PanelWeather, Duration: 60 * time.Second, Refresh: 60 * time.Second},
	}}

	start := time.Date(2025, 1, 1, 8, 0, 0, 0, time.UTC)

	pinned := time.Time{}

	totalSeconds := int(timeline.Total() / time.Second)

	for elapsed := 0; elapsed <= totalSeconds; elapsed++ {
		now := start.Add(time.Duration(elapsed) * time.Second)

		boundary := timeline.NextBoundary(start, now)

		if !boundary.After(now) {
			t.Fatalf(
				"NextBoundary: elapsed %ds: %s not after now",
				elapsed,
				boundary,
			)
		}

		if !pinned.IsZero() && boundary.Before(pinned) {
			t.Fatalf(
				"NextBoundary: elapsed %ds: %s before previous boundary %s",
				elapsed,
				boundary,
				pinned,
			)
		}

		if boundary.After(pinned) {
			pinned = boundary
		}
	}

	expectedBoundaries := []time.Duration{
		60 * time.Second,
		120 * time.Second,
		240 * time.Second,
		time.Duration(timeline.Total()),
		time.Duration(timeline.Total()) + 60*time.Second,
	}

	boundary := start.Add(-time.Second)

	for elapsed := 0; elapsed <= totalSeconds; elapsed++ {
		now := start.Add(time.Duration(elapsed) * time.Second)

		next := timeline.NextBoundary(start, now)

		if next.After(boundary) {
			boundary = next

			want := start.Add(expectedBoundaries[0])

			if next != want {
				t.Fatalf("NextBoundary: %s, want %s", next, want)
			}

			expectedBoundaries = expectedBoundaries[1:]
		}
	}

	if len(expectedBoundaries) != 0 {
		t.Fatalf("NextBoundary: %d boundaries never reached", len(expectedBoundaries))
	}
}

func TestTimelineValidation(t *testing.T) {
	tests := []struct {
		name     string
		steps    []TimelineStep
		wantFail bool
	}{
		{
			name:     "empty",
			steps:    nil,
			wantFail: true,
		},
		{
			name:     "refresh below floor",
			steps:    []TimelineStep{{Panel: display.PanelWeather, Duration: time.Minute, Refresh: 5 * time.Second}},
			wantFail: true,
		},
		{
			name:     "refresh exceeds duration",
			steps:    []TimelineStep{{Panel: display.PanelWeather, Duration: 30 * time.Second, Refresh: time.Minute}},
			wantFail: true,
		},
		{
			name: "valid",
			steps: []TimelineStep{
				{Panel: display.PanelWeather, Duration: time.Minute, Refresh: 30 * time.Second},
				{Panel: display.PanelBusProgress, Duration: time.Minute, Refresh: 25 * time.Second},
			},
		},
	}

	for _, test := range tests {
		timeline := &Timeline{Steps: test.steps}

		err := timeline.validate()

		if (err != nil) != test.wantFail {
			t.Errorf("%s: validate err = %v, wantFail %v", test.name, err, test.wantFail)
		}
	}
}

func TestTimelineCatalogPresetsAndOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "board-timeline.yml")

	file := `
views:
  weather:
    duration: 1m
  bus:
    map:
      duration: 3m
    progress:
      duration: 1m
      refresh: 20s

order: my-order

presets:
  my-order: [weather, bus.progress, weather, bus.map]
  default: [weather, bus.schedule]
`

	if err := os.WriteFile(path, []byte(file), 0o644); err != nil {
		t.Fatalf("write timeline file: %v", err)
	}

	catalog, err := LoadTimelineCatalog(path)
	if err != nil {
		t.Fatalf("LoadTimelineCatalog: %v", err)
	}

	tests := []struct {
		order           string
		wantPanels      []display.Panel
		wantProgressRej time.Duration
		wantMapDuration time.Duration
	}{
		{
			order:           "",
			wantPanels:      []display.Panel{display.PanelWeather, display.PanelBusProgress, display.PanelWeather, display.PanelBusMap},
			wantProgressRej: 20 * time.Second,
			wantMapDuration: 3 * time.Minute,
		},
		{
			order:      "default",
			wantPanels: []display.Panel{display.PanelWeather, display.PanelBus},
		},
		{
			order:      "interleaved", // builtin, still available
			wantPanels: []display.Panel{display.PanelWeather, display.PanelBusMap, display.PanelWeather, display.PanelBus, display.PanelWeather, display.PanelBusProgress},
		},
		{
			order:      "weather,bus.progress",
			wantPanels: []display.Panel{display.PanelWeather, display.PanelBusProgress},
		},
	}

	for _, test := range tests {
		timeline, err := catalog.Resolve(test.order)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", test.order, err)
		}

		if len(timeline.Steps) != len(test.wantPanels) {
			t.Fatalf(
				"Resolve(%q) produced %d steps, want %d",
				test.order,
				len(timeline.Steps),
				len(test.wantPanels),
			)
		}

		for i, want := range test.wantPanels {
			if timeline.Steps[i].Panel != want {
				t.Errorf(
					"Resolve(%q) step %d = %s, want %s",
					test.order,
					i,
					timeline.Steps[i].Panel,
					want,
				)
			}
		}

		if test.wantProgressRej > 0 {
			for _, step := range timeline.Steps {
				if step.Panel == display.PanelBusProgress && step.Refresh != test.wantProgressRej {
					t.Errorf(
						"progress refresh = %s, want %s",
						step.Refresh,
						test.wantProgressRej,
					)
				}
			}
		}

		if test.wantMapDuration > 0 {
			for _, step := range timeline.Steps {
				if step.Panel == display.PanelBusMap && step.Duration != test.wantMapDuration {
					t.Errorf(
						"map duration = %s, want %s",
						step.Duration,
						test.wantMapDuration,
					)
				}
			}
		}
	}
}

func TestTimelineCatalogDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "board-timeline.yml")

	// Order only: everything else comes from built-in defaults.
	if err := os.WriteFile(path, []byte("order: [weather, bus.progress]\n"), 0o644); err != nil {
		t.Fatalf("write timeline file: %v", err)
	}

	catalog, err := LoadTimelineCatalog(path)
	if err != nil {
		t.Fatalf("LoadTimelineCatalog: %v", err)
	}

	timeline, err := catalog.Resolve("")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if timeline.Steps[0].Panel != display.PanelWeather ||
		timeline.Steps[0].Duration != 2*time.Minute ||
		timeline.Steps[0].Refresh != 60*time.Second {
		t.Errorf("weather defaults = %+v", timeline.Steps[0])
	}

	if timeline.Steps[1].Panel != display.PanelBusProgress ||
		timeline.Steps[1].Duration != 2*time.Minute ||
		timeline.Steps[1].Refresh != 25*time.Second {
		t.Errorf("progress defaults = %+v", timeline.Steps[1])
	}
}

func TestTimelineCatalogMissingFile(t *testing.T) {
	catalog, err := LoadTimelineCatalog(filepath.Join(t.TempDir(), "board-timeline.yml"))
	if err != nil {
		t.Fatalf("LoadTimelineCatalog: %v", err)
	}

	if catalog != nil {
		t.Fatal("missing file should return a nil catalog")
	}
}

func TestTimelineCatalogErrors(t *testing.T) {
	tests := []struct {
		name string
		file string
	}{
		{
			name: "unknown view in order",
			file: "order: [weather, mup]\n",
		},
		{
			name: "unknown preset",
			file: "order: no-such-preset\n",
		},
		{
			name: "unknown view in views",
			file: "views:\n  mup:\n    duration: 1m\n",
		},
		{
			name: "invalid duration",
			file: "views:\n  weather:\n    duration: soon\n",
		},
		{
			name: "refresh below floor",
			file: "views:\n  progress:\n    refresh: 2s\n",
		},
		{
			name: "bad yaml",
			file: "views: [unclosed\n",
		},
		{
			name: "empty preset list",
			file: "presets:\n  empty: []\norder: empty\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "board-timeline.yml")
			if err := os.WriteFile(path, []byte(test.file), 0o644); err != nil {
				t.Fatalf("write: %v", err)
			}

			if _, err := LoadTimelineCatalog(path); err == nil {
				t.Errorf("LoadTimelineCatalog(%q) = nil error, want error", test.file)
			}
		})
	}
}

func TestTimelineCatalogUnknownResolve(t *testing.T) {
	path := filepath.Join(t.TempDir(), "board-timeline.yml")

	if err := os.WriteFile(path, []byte("order: [weather]\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	catalog, err := LoadTimelineCatalog(path)
	if err != nil {
		t.Fatalf("LoadTimelineCatalog: %v", err)
	}

	if _, err := catalog.Resolve("does-not-exist"); err == nil {
		t.Error("Resolve(unknown preset) = nil error, want error")
	}
}

func TestDefaultTimelineCatalog(t *testing.T) {
	catalog := DefaultTimelineCatalog()

	timeline, err := catalog.Resolve("")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	want := []display.Panel{
		display.PanelWeather,
		display.PanelBusMap,
		display.PanelBus,
		display.PanelBusProgress,
	}

	if len(timeline.Steps) != len(want) {
		t.Fatalf("steps = %d, want %d", len(timeline.Steps), len(want))
	}

	for i, panel := range want {
		if timeline.Steps[i].Panel != panel {
			t.Errorf("step %d = %s, want %s", i, timeline.Steps[i].Panel, panel)
		}
	}
}

func TestNewTimelineFromLegacySchedule(t *testing.T) {
	schedule, err := ParseBoardSchedule("bus:{map:2,schedule:2,progress:3},weather:1")
	if err != nil {
		t.Fatalf("ParseBoardSchedule: %v", err)
	}

	timeline := NewTimelineFromLegacySchedule(schedule)
	if err := timeline.validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	want := []TimelineStep{
		{Panel: display.PanelBusMap, Duration: 2 * time.Minute, Refresh: time.Minute},
		{Panel: display.PanelBus, Duration: 2 * time.Minute, Refresh: time.Minute},
		{Panel: display.PanelBusProgress, Duration: 3 * time.Minute, Refresh: time.Minute},
		{Panel: display.PanelWeather, Duration: time.Minute, Refresh: time.Minute},
	}

	if len(timeline.Steps) != len(want) {
		t.Fatalf("steps = %d, want %d", len(timeline.Steps), len(want))
	}

	for i, step := range want {
		if timeline.Steps[i] != step {
			t.Errorf("step %d = %+v, want %+v", i, timeline.Steps[i], step)
		}
	}
}

func TestBoardServiceResolveTimeline(t *testing.T) {
	// Without a catalog, alarms always fall back to the server default.
	board := NewBoardService("", nil, nil, nil, nil)

	timeline, err := board.ResolveTimeline("bus-focus")
	if err != nil {
		t.Fatalf("ResolveTimeline without catalog: %v", err)
	}

	if timeline == nil || timeline.Steps[0].Panel != display.PanelWeather {
		t.Fatalf("ResolveTimeline without catalog: %+v", timeline)
	}

	board = NewBoardService("", nil, nil, nil, DefaultTimelineCatalog())

	timeline, err = board.ResolveTimeline("")
	if err != nil || timeline == nil {
		t.Fatalf("ResolveTimeline default order: %v, %v", timeline, err)
	}

	if timeline.Steps[0].Panel != display.PanelWeather {
		t.Fatalf("default order starts with %+v", timeline.Steps[0].Panel)
	}

	timeline, err = board.ResolveTimeline("interleaved")
	if err != nil {
		t.Fatalf("ResolveTimeline preset: %v", err)
	}

	var panels []display.Panel
	for _, step := range timeline.Steps {
		panels = append(panels, step.Panel)
	}

	if len(panels) != 6 || panels[0] != display.PanelWeather ||
		panels[1] != display.PanelBusMap {
		t.Fatalf("interleaved preset = %v", panels)
	}

	if _, err := board.ResolveTimeline("no-such-preset"); err == nil {
		t.Fatalf("unknown preset did not fail")
	}
}
