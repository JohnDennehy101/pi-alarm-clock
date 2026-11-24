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
	board := NewBoardService("", nil, nil, BoardTimelineConfig{
		Timeline:     nil,
		Catalog:      nil,
		TimelinePath: "",
	})

	timeline, err := board.ResolveTimeline("bus-focus")
	if err != nil {
		t.Fatalf("ResolveTimeline without catalog: %v", err)
	}

	if timeline == nil || timeline.Steps[0].Panel != display.PanelWeather {
		t.Fatalf("ResolveTimeline without catalog: %+v", timeline)
	}

	board = NewBoardService("", nil, nil, BoardTimelineConfig{
		Timeline:     nil,
		Catalog:      DefaultTimelineCatalog(),
		TimelinePath: "",
	})

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

func TestTimelineAtWithIntro(t *testing.T) {
	timeline := &Timeline{
		Intro: []TimelineStep{
			{Panel: display.PanelWeather, Duration: 60 * time.Second, Refresh: 30 * time.Second},
		},
		Steps: []TimelineStep{
			{Panel: display.PanelBus, Duration: 120 * time.Second, Refresh: 60 * time.Second},
			{Panel: display.PanelBusProgress, Duration: 60 * time.Second, Refresh: 30 * time.Second},
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
		{29 * time.Second, display.PanelWeather, 0, 0},
		{30 * time.Second, display.PanelWeather, 0, 1},
		{59 * time.Second, display.PanelWeather, 0, 1},
		{60 * time.Second, display.PanelBus, 1, 0},
		{90 * time.Second, display.PanelBus, 1, 0},
		{120 * time.Second, display.PanelBus, 1, 1},
		{180 * time.Second, display.PanelBusProgress, 2, 0},
		{210 * time.Second, display.PanelBusProgress, 2, 1},
		{240 * time.Second, display.PanelBus, 1, 0},         // first wrap: intro never replays
		{420 * time.Second, display.PanelBus, 1, 0},         // second wrap
		{580 * time.Second, display.PanelBusProgress, 2, 1}, // third wrap mid-loop
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

	// The intro must never appear again once consumed.
	for offset := 60; offset <= 2_400; offset += 60 {
		if slot := timeline.At(start, start.Add(time.Duration(offset)*time.Second)); slot.Panel == display.PanelWeather {
			t.Fatalf("At(+%ds) = weather, intro must not repeat", offset)
		}
	}

	if timeline.IntroTotal() != 60*time.Second {
		t.Fatalf("IntroTotal = %s, want 60s", timeline.IntroTotal())
	}

	if d := timeline.Duration(BoardSlot{Step: 0}); d != 60*time.Second {
		t.Fatalf("Duration(intro step 0) = %s, want 60s", d)
	}

	if d := timeline.Duration(BoardSlot{Step: 1}); d != 120*time.Second {
		t.Fatalf("Duration(loop step 1) = %s, want 120s", d)
	}

	if d := timeline.Duration(BoardSlot{Step: 3}); d != 0 {
		t.Fatalf("Duration(out of range) = %s, want 0", d)
	}
}

func TestTimelineNextBoundaryWithIntro(t *testing.T) {
	timeline := Timeline{
		Intro: []TimelineStep{
			{Panel: display.PanelWeather, Duration: 30 * time.Second, Refresh: 30 * time.Second},
		},
		Steps: []TimelineStep{
			{Panel: display.PanelBusMap, Duration: 120 * time.Second, Refresh: 60 * time.Second},
			{Panel: display.PanelBus, Duration: 120 * time.Second, Refresh: 120 * time.Second},
		},
	}

	if err := timeline.validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}

	start := time.Date(2026, 3, 2, 7, 0, 0, 0, time.UTC)

	expectedBoundaries := []time.Duration{
		30 * time.Second,  // intro step ends
		90 * time.Second,  // map refresh tick
		150 * time.Second, // map ends
		270 * time.Second, // bus ends, first loop wrap
		330 * time.Second, // map refresh tick, cycle 2
		390 * time.Second, // map ends, cycle 2
		510 * time.Second, // bus ends, cycle 3
		570 * time.Second, // map refresh tick, cycle 3
	}

	boundaryIndex := 0
	pinned := time.Time{}

	totalSeconds := int((timeline.IntroTotal() + 2*timeline.Total()) / time.Second)

	for elapsed := 0; elapsed <= totalSeconds+15; elapsed++ {
		now := start.Add(time.Duration(elapsed) * time.Second)

		next := timeline.NextBoundary(start, now)

		if !next.After(now) {
			t.Fatalf("NextBoundary: elapsed %ds: %s not after now", elapsed, next)
		}

		if !pinned.IsZero() && next.Before(pinned) {
			t.Fatalf("NextBoundary: elapsed %ds: %s before previous boundary %s", elapsed, next, pinned)
		}

		if next.After(pinned) {
			if boundaryIndex >= len(expectedBoundaries) {
				t.Fatalf("unexpected extra boundary %s at elapsed %ds", next, elapsed)
			}

			want := start.Add(expectedBoundaries[boundaryIndex])
			if next != want {
				t.Fatalf("boundary %d = %s, want %s", boundaryIndex, next, want)
			}

			pinned = next
			boundaryIndex++
		} else {
			pinned = next
		}
	}

	if boundaryIndex != len(expectedBoundaries) {
		t.Fatalf("reached %d boundaries, want %d", boundaryIndex, len(expectedBoundaries))
	}
}

func TestTimelineIntroValidation(t *testing.T) {
	validSteps := []TimelineStep{
		{Panel: display.PanelBus, Duration: time.Minute, Refresh: 30 * time.Second},
	}

	tests := []struct {
		name     string
		intro    []TimelineStep
		steps    []TimelineStep
		wantFail bool
	}{
		{
			name:  "no intro is valid",
			intro: nil,
			steps: validSteps,
		},
		{
			name:  "valid intro",
			intro: []TimelineStep{{Panel: display.PanelWeather, Duration: 5 * time.Minute, Refresh: 60 * time.Second}},
			steps: validSteps,
		},
		{
			name:     "intro refresh below floor",
			intro:    []TimelineStep{{Panel: display.PanelWeather, Duration: 5 * time.Minute, Refresh: 5 * time.Second}},
			steps:    validSteps,
			wantFail: true,
		},
		{
			name:     "intro refresh exceeds duration",
			intro:    []TimelineStep{{Panel: display.PanelWeather, Duration: 30 * time.Second, Refresh: 60 * time.Second}},
			steps:    validSteps,
			wantFail: true,
		},
		{
			name:     "intro alone cannot replace the loop",
			intro:    []TimelineStep{{Panel: display.PanelWeather, Duration: 5 * time.Minute, Refresh: 60 * time.Second}},
			steps:    nil,
			wantFail: true,
		},
	}

	for _, test := range tests {
		timeline := &Timeline{Intro: test.intro, Steps: test.steps}

		err := timeline.validate()

		if (err != nil) != test.wantFail {
			t.Errorf("%s: validate err = %v, wantFail %v", test.name, err, test.wantFail)
		}
	}
}

func TestTimelineCatalogIntroDecode(t *testing.T) {
	tests := []struct {
		name      string
		file      string
		wantIntro []string
		wantError bool
	}{
		{
			name:      "no intro section",
			file:      "order: [bus.schedule]\n",
			wantIntro: nil,
		},
		{
			name:      "empty intro list",
			file:      "order: [bus.schedule]\nintro: []\n",
			wantIntro: nil,
		},
		{
			name:      "null intro",
			file:      "order: [bus.schedule]\nintro: ~\n",
			wantIntro: nil,
		},
		{
			name:      "intro view list",
			file:      "order: [bus.schedule]\nintro: [weather, bus.map]\n",
			wantIntro: []string{"weather", "bus.map"},
		},
		{
			name:      "intro from preset name",
			file:      "order: [bus.schedule]\nintro: bus-focus\n",
			wantIntro: []string{"bus.map", "bus.schedule", "bus.progress", "weather"},
		},
		{
			name:      "unknown view in intro",
			file:      "order: [bus.schedule]\nintro: [nonsense]\n",
			wantError: true,
		},
	}

	for _, test := range tests {
		path := filepath.Join(t.TempDir(), "board-timeline.yml")

		if err := os.WriteFile(path, []byte(test.file), 0o644); err != nil {
			t.Fatalf("%s: write: %v", test.name, err)
		}

		catalog, err := LoadTimelineCatalog(path)

		if test.wantError {
			if err == nil {
				t.Errorf("%s: LoadTimelineCatalog err = nil, want error", test.name)
			}

			continue
		}

		if err != nil {
			t.Fatalf("%s: LoadTimelineCatalog: %v", test.name, err)
		}

		if len(catalog.intro) != len(test.wantIntro) {
			t.Fatalf("%s: intro = %v, want %v", test.name, catalog.intro, test.wantIntro)
		}

		for i := range test.wantIntro {
			if catalog.intro[i].Name != test.wantIntro[i] {
				t.Errorf("%s: intro[%d] = %s, want %s", test.name, i, catalog.intro[i], test.wantIntro[i])
			}
		}

		timeline, err := catalog.Resolve("")
		if err != nil {
			t.Fatalf("%s: Resolve: %v", test.name, err)
		}

		if len(timeline.Intro) != len(test.wantIntro) {
			t.Fatalf("%s: timeline intro = %d steps, want %d", test.name, len(timeline.Intro), len(test.wantIntro))
		}
	}
}

func TestTimelineCatalogResolveForCast(t *testing.T) {
	catalog := DefaultTimelineCatalog()
	catalog.intro = entriesForNames([]string{"weather"}) // global intro must NOT leak into casts

	introPanels := func(tl *Timeline) []display.Panel {
		panels := make([]display.Panel, 0, len(tl.Intro))
		for _, step := range tl.Intro {
			panels = append(panels, step.Panel)
		}
		return panels
	}

	// Opt-in is explicit: empty intro plays nothing, even when the
	// catalog has a configured global intro.
	timeline, err := catalog.ResolveForCast("bus-focus", "")
	if err != nil {
		t.Fatalf("ResolveForCast(no intro): %v", err)
	}

	if len(timeline.Intro) != 0 {
		t.Fatalf("cast with empty intro = %+v, want no intro", timeline.Intro)
	}

	if len(timeline.Steps) == 0 {
		t.Fatal("cast with empty intro lost the loop")
	}

	// A single view name works (the main case: weather first).
	timeline, err = catalog.ResolveForCast("", "weather")
	if err != nil {
		t.Fatalf("ResolveForCast(weather intro): %v", err)
	}

	panels := introPanels(timeline)
	if len(panels) != 1 || panels[0] != display.PanelWeather {
		t.Fatalf("intro panels = %v, want [weather]", panels)
	}

	// A comma-separated list.
	timeline, err = catalog.ResolveForCast("bus-focus", "weather,bus.map")
	if err != nil {
		t.Fatalf("ResolveForCast(list intro): %v", err)
	}

	panels = introPanels(timeline)
	if len(panels) != 2 ||
		panels[0] != display.PanelWeather ||
		panels[1] != display.PanelBusMap {
		t.Fatalf("intro panels = %v, want [weather bus-map]", panels)
	}

	// A preset name expands to that preset's order as the intro.
	timeline, err = catalog.ResolveForCast("", "bus-focus")
	if err != nil {
		t.Fatalf("ResolveForCast(preset intro): %v", err)
	}

	panels = introPanels(timeline)
	if len(panels) != 4 || panels[0] != display.PanelBusMap {
		t.Fatalf("bus-focus intro = %v", panels)
	}

	// Loop order still resolves alongside any intro expression.
	timeline, err = catalog.ResolveForCast("interleaved", "weather")
	if err != nil {
		t.Fatalf("ResolveForCast(interleaved): %v", err)
	}

	if len(timeline.Steps) != 6 || timeline.Steps[0].Panel != display.PanelWeather {
		t.Fatalf("interleaved loop = %d steps", len(timeline.Steps))
	}

	// Unknown views reject the whole cast resolution.
	if _, err := catalog.ResolveForCast("", "mup"); err == nil {
		t.Fatal("ResolveForCast(unknown intro view) = nil error")
	}

	if _, err := catalog.ResolveForCast("", "no-such-preset"); err == nil {
		t.Fatal("ResolveForCast(unknown intro preset) = nil error")
	}

	if _, err := catalog.ResolveForCast("no-such-preset", "weather"); err == nil {
		t.Fatal("ResolveForCast(unknown order) = nil error")
	}

	// Intro length is bounded like the order.
	longIntro := "weather"
	for len(longIntro+",weather") <= 200 {
		longIntro += ",weather"
	}

	if _, err := catalog.ResolveForCast("", longIntro+",weather,eventually-more"); err == nil {
		t.Fatal("ResolveForCast(over-long intro) = nil error")
	}
}

func TestBoardServiceResolveTimelineForCast(t *testing.T) {
	catalog := DefaultTimelineCatalog()
	catalog.intro = entriesForNames([]string{"weather"})
	catalog.order = entriesForNames([]string{"bus.schedule", "bus.map", "bus.progress"})

	path := filepath.Join(t.TempDir(), "board-timeline.yml")

	if err := os.WriteFile(path, []byte("intro: [weather]\norder: [bus.schedule]\n"), 0o644); err != nil {
		t.Fatalf("write timeline: %v", err)
	}

	loaded, err := LoadTimelineCatalog(path)
	if err != nil {
		t.Fatalf("LoadTimelineCatalog: %v", err)
	}

	board := NewBoardService("", nil, nil, BoardTimelineConfig{
		Timeline:     nil,
		Catalog:      loaded,
		TimelinePath: path,
	})

	// No intro requested: only the loop, even though the catalog
	// configures a global intro.
	timeline, err := board.ResolveTimelineForCast("", "")
	if err != nil {
		t.Fatalf("ResolveTimelineForCast(empty): %v", err)
	}

	if len(timeline.Intro) != 0 || len(timeline.Steps) != 1 || timeline.Steps[0].Panel != display.PanelBus {
		t.Fatalf("cast timeline = %+v", timeline)
	}

	// Opt-in intro from a single view name.
	timeline, err = board.ResolveTimelineForCast("bus-focus", "weather")
	if err != nil {
		t.Fatalf("ResolveTimelineForCast(intro): %v", err)
	}

	if len(timeline.Intro) != 1 || timeline.Intro[0].Panel != display.PanelWeather {
		t.Fatalf("cast intro = %+v", timeline.Intro)
	}

	if len(timeline.Steps) != 4 || timeline.Steps[0].Panel != display.PanelBusMap {
		t.Fatalf("cast loop = %+v", timeline.Steps)
	}

	// A board without a catalog still resolves expressions from
	// defaults, and serves its configured timeline when nothing is
	// requested.
	defaultBoard := NewBoardService("", nil, nil, BoardTimelineConfig{
		Timeline:     nil,
		Catalog:      nil,
		TimelinePath: "",
	})

	timeline, err = defaultBoard.ResolveTimelineForCast("", "")
	if err != nil {
		t.Fatalf("default ResolveTimelineForCast: %v", err)
	}

	if timeline == nil || len(timeline.Steps) == 0 {
		t.Fatalf("default cast timeline = %+v", timeline)
	}

	timeline, err = defaultBoard.ResolveTimelineForCast("", "weather")
	if err != nil {
		t.Fatalf("default ResolveTimelineForCast(intro): %v", err)
	}

	if len(timeline.Intro) != 1 || timeline.Intro[0].Panel != display.PanelWeather {
		t.Fatalf("default cast intro = %+v", timeline.Intro)
	}

	// The default catalog route still unwraps the global intro for
	// non-cast (preview / default timeline) consumers.
	timeline, err = loaded.Resolve("")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if len(timeline.Intro) != 1 || timeline.Intro[0].Panel != display.PanelWeather {
		t.Fatalf("default resolve intro = %+v", timeline.Intro)
	}
}

func TestTimelineCatalogGroupedEntriesAndOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "board-timeline.yml")

	file := `
views:
  weather:
    duration: 2m
    refresh: 1m

order:
  - weather
  - bus: schedule
  - weather:
      duration: 30s
      refresh: 10s
  - bus:
      map:
        duration: 45s
        refresh: 15s

presets:
  morning-commute:
    - bus:
        schedule:
          duration: 3m
    - bus: map
`

	if err := os.WriteFile(path, []byte(file), 0o644); err != nil {
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

	want := []struct {
		panel    display.Panel
		duration time.Duration
		refresh  time.Duration
	}{
		{display.PanelWeather, 2 * time.Minute, time.Minute},
		{display.PanelBus, 2 * time.Minute, time.Minute},
		{display.PanelWeather, 30 * time.Second, 10 * time.Second},
		{display.PanelBusMap, 45 * time.Second, 15 * time.Second},
	}

	if len(timeline.Steps) != len(want) {
		t.Fatalf("Resolve produced %d steps, want %d: %+v", len(timeline.Steps), len(want), timeline.Steps)
	}

	for i, step := range timeline.Steps {
		if step.Panel != want[i].panel ||
			step.Duration != want[i].duration ||
			step.Refresh != want[i].refresh {
			t.Errorf("step %d = %+v, want %+v", i, step, want[i])
		}
	}

	// Per-preset duration overrides apply when an alarm references the preset.
	presetTimeline, err := catalog.Resolve("morning-commute")
	if err != nil {
		t.Fatalf("Resolve(morning-commute): %v", err)
	}

	if len(presetTimeline.Steps) != 2 {
		t.Fatalf("preset timeline = %+v", presetTimeline.Steps)
	}

	if presetTimeline.Steps[0].Panel != display.PanelBus ||
		presetTimeline.Steps[0].Duration != 3*time.Minute {
		t.Errorf("preset schedule step = %+v, want 3m", presetTimeline.Steps[0])
	}

	if presetTimeline.Steps[1].Panel != display.PanelBusMap {
		t.Errorf("preset map step = %+v", presetTimeline.Steps[1])
	}
}

func TestTimelineCatalogEntryExpressionOverrides(t *testing.T) {
	catalog := DefaultTimelineCatalog()

	timeline, err := catalog.ResolveForCast("weather:2m:30s,bus.map:45s:15s,weather:3m", "")
	if err != nil {
		t.Fatalf("ResolveForCast: %v", err)
	}

	if len(timeline.Steps) != 3 {
		t.Fatalf("timeline = %+v", timeline.Steps)
	}

	if timeline.Steps[0].Duration != 2*time.Minute || timeline.Steps[0].Refresh != 30*time.Second {
		t.Errorf("weather step = %+v, want 2m/30s", timeline.Steps[0])
	}

	if timeline.Steps[1].Panel != display.PanelBusMap ||
		timeline.Steps[1].Duration != 45*time.Second ||
		timeline.Steps[1].Refresh != 15*time.Second {
		t.Errorf("bus.map step = %+v, want 45s/15s", timeline.Steps[1])
	}

	// Duration-only override: refresh falls back to the per-view default.
	if timeline.Steps[2].Duration != 3*time.Minute || timeline.Steps[2].Refresh != time.Minute {
		t.Errorf("weather(3m) step = %+v, want 3m/1m", timeline.Steps[2])
	}

	// A single word that names a preset still expands to that preset.
	presetTimeline, err := catalog.ResolveForCast("bus-focus", "")
	if err != nil || len(presetTimeline.Steps) != 4 {
		t.Fatalf("ResolveForCast(bus-focus): %v, steps = %+v", err, presetTimeline.Steps)
	}

	// A single plain view name (not a preset) resolves to a one-view order.
	single, err := catalog.ResolveForCast("weather", "")
	if err != nil || len(single.Steps) != 1 || single.Steps[0].Panel != display.PanelWeather {
		t.Fatalf("ResolveForCast(weather): %v, steps = %+v", err, single.Steps)
	}

	bad := []string{
		"weather:0s",
		"weather:2m:5s",
		"weather:2m:30s:12m",
		"mup:2m",
		"weather:abc",
		"weather:1s:0s",
	}

	for _, expr := range bad {
		if _, err := catalog.ResolveForCast(expr, ""); err == nil {
			t.Errorf("ResolveForCast(%q) = nil error, want error", expr)
		}
	}
}

func TestTimelineCatalogIntroExpressionOverrides(t *testing.T) {
	catalog := DefaultTimelineCatalog()

	timeline, err := catalog.ResolveForCast("bus.map", "weather:30s:10s,bus.progress")
	if err != nil {
		t.Fatalf("ResolveForCast: %v", err)
	}

	if len(timeline.Intro) != 2 {
		t.Fatalf("intro = %+v", timeline.Intro)
	}

	if timeline.Intro[0].Panel != display.PanelWeather ||
		timeline.Intro[0].Duration != 30*time.Second ||
		timeline.Intro[0].Refresh != 10*time.Second {
		t.Errorf("intro weather step = %+v, want 30s/10s", timeline.Intro[0])
	}

	if timeline.Intro[1].Panel != display.PanelBusProgress {
		t.Errorf("intro progress step = %+v", timeline.Intro[1])
	}
}
