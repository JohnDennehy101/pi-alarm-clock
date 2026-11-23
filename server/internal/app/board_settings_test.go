package app

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/display"
)

func TestBoardServiceUpdateViewSettingsPersistsAndReloads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "board-timeline.yml")
	board := NewBoardService("", nil, nil, nil, nil, path)

	duration := int64(90)
	refresh := int64(15)

	if err := board.UpdateViewSettings("weather", &duration, &refresh); err != nil {
		t.Fatalf("UpdateViewSettings: %v", err)
	}

	settings := board.TimelineSettings()
	if settings.Views["weather"].DurationSeconds != 90 {
		t.Fatalf("in-memory weather duration = %d, want 90", settings.Views["weather"].DurationSeconds)
	}

	if settings.Views["weather"].RefreshSeconds != 15 {
		t.Fatalf("in-memory weather refresh = %d, want 15", settings.Views["weather"].RefreshSeconds)
	}

	catalog, err := LoadTimelineCatalog(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	timeline, err := catalog.Resolve("weather,bus.map")
	if err != nil {
		t.Fatalf("reload resolve: %v", err)
	}

	if timeline.Steps[0].Duration != 90*time.Second ||
		timeline.Steps[0].Refresh != 15*time.Second {
		t.Fatalf("reloaded weather step = %+v", timeline.Steps[0])
	}

	timeline, err = board.ResolveTimeline("")
	if err != nil {
		t.Fatalf("default timeline after mutation: %v", err)
	}

	if timeline.Steps[0].Panel != display.PanelWeather ||
		timeline.Steps[0].Duration != 90*time.Second {
		t.Fatalf("default timeline not swapped after mutation: %+v", timeline.Steps[0])
	}
}

func TestBoardServiceViewSettingsValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "board-timeline.yml")
	board := NewBoardService("", nil, nil, nil, nil, path)

	second := int64(1)
	nine := int64(9)
	tooLong := int64(60*60 + 1)
	thirty := int64(30)
	ninety := int64(90)

	tests := []struct {
		name      string
		view      string
		duration  *int64
		refresh   *int64
		wantError string
	}{
		{name: "no fields", view: "weather", wantError: "durationSeconds or refreshSeconds"},
		{name: "unknown view", view: "mup", duration: &second, wantError: "unknown board view"},
		{name: "duration below 1s", view: "weather", duration: &zeroInt64, wantError: "duration"},
		{name: "duration above 60m", view: "weather", duration: &tooLong, wantError: "duration"},
		{name: "refresh below floor", view: "weather", refresh: &nine, wantError: "refresh"},
		{name: "refresh exceeds duration", view: "weather", duration: &thirty, refresh: &ninety, wantError: "exceeds duration"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := board.UpdateViewSettings(test.view, test.duration, test.refresh)
			if err == nil {
				t.Fatal("UpdateViewSettings = nil error")
			}

			if !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("UpdateViewSettings error = %q, want containing %q", err, test.wantError)
			}
		})
	}
}

func TestBoardServiceOrderMutations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "board-timeline.yml")
	board := NewBoardService("", nil, nil, nil, nil, path)

	if err := board.SetTimelineOrderViews([]string{"bus", "progress"}); err != nil {
		t.Fatalf("SetTimelineOrderViews: %v", err)
	}

	settings := board.TimelineSettings()
	want := []string{"bus.schedule", "bus.progress"}
	for i := range want {
		if settings.Order[i] != want[i] {
			t.Fatalf("order = %v, want %v", settings.Order, want)
		}
	}

	timeline, err := board.ResolveTimeline("")
	if err != nil {
		t.Fatalf("ResolveTimeline: %v", err)
	}

	if timeline.Steps[0].Panel != display.PanelBus {
		t.Fatalf("default timeline starts with %s, want bus", timeline.Steps[0].Panel)
	}

	if err := board.SetTimelineOrder("bus-focus"); err != nil {
		t.Fatalf("SetTimelineOrder preset: %v", err)
	}

	settings = board.TimelineSettings()
	if settings.Order[0] != "bus.map" || settings.Order[3] != "weather" {
		t.Fatalf("bus-focus order = %v", settings.Order)
	}

	if err := board.SetTimelineOrderViews([]string{"weather", "mup"}); err == nil {
		t.Fatal("SetTimelineOrderViews(unknown view) = nil error")
	}

	if err := board.SetTimelineOrder("no-such-preset"); err == nil {
		t.Fatal("SetTimelineOrder(unknown preset) = nil error")
	}
}

func TestBoardServicePresetLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "board-timeline.yml")
	board := NewBoardService("", nil, nil, nil, nil, path)

	if err := board.CreatePreset("morning", []string{"weather", "bus.map"}); err != nil {
		t.Fatalf("CreatePreset: %v", err)
	}

	if err := board.CreatePreset("morning", []string{"weather"}); !errors.Is(err, ErrBoardTimelinePresetExists) {
		t.Fatalf("duplicate CreatePreset error = %v, want ErrBoardTimelinePresetExists", err)
	}

	preset, err := board.GetPreset("morning")
	if err != nil {
		t.Fatalf("GetPreset: %v", err)
	}

	if preset.BuiltIn {
		t.Fatalf("user preset should not be builtIn: %+v", preset)
	}

	if len(preset.Order) != 2 || preset.Order[1] != "bus.map" {
		t.Fatalf("preset order = %v", preset.Order)
	}

	catalog, err := LoadTimelineCatalog(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	if _, err := catalog.Resolve("morning"); err != nil {
		t.Fatalf("Resolve(morning) after reload: %v", err)
	}

	if err := board.DeletePreset("morning"); err != nil {
		t.Fatalf("DeletePreset: %v", err)
	}

	if _, err := board.GetPreset("morning"); !errors.Is(err, ErrBoardTimelinePresetNotFound) {
		t.Fatalf("GetPreset after delete error = %v, want ErrBoardTimelinePresetNotFound", err)
	}

	if err := board.DeletePreset("morning"); !errors.Is(err, ErrBoardTimelinePresetNotFound) {
		t.Fatalf("DeletePreset again error = %v, want ErrBoardTimelinePresetNotFound", err)
	}

	if err := board.DeletePreset("default"); !errors.Is(err, ErrBoardTimelineBuiltinPreset) {
		t.Fatalf("DeletePreset(builtin) error = %v, want ErrBoardTimelineBuiltinPreset", err)
	}
}

func TestBoardServiceUpdatePresetOverridesBuiltin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "board-timeline.yml")
	board := NewBoardService("", nil, nil, nil, nil, path)

	if err := board.UpdatePreset("interleaved", []string{"weather", "bus.progress"}); err != nil {
		t.Fatalf("UpdatePreset: %v", err)
	}

	preset, err := board.GetPreset("interleaved")
	if err != nil {
		t.Fatalf("GetPreset: %v", err)
	}

	if !preset.BuiltIn {
		t.Fatalf("overridden builtin should still report builtIn: %+v", preset)
	}

	if len(preset.Order) != 2 || preset.Order[1] != "bus.progress" {
		t.Fatalf("overridden order = %v", preset.Order)
	}

	catalog, err := LoadTimelineCatalog(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	timeline, err := catalog.Resolve("interleaved")
	if err != nil {
		t.Fatalf("Resolve(interleaved): %v", err)
	}

	if len(timeline.Steps) != 2 || timeline.Steps[1].Panel != display.PanelBusProgress {
		t.Fatalf("reloaded interleaved = %+v", timeline.Steps)
	}
}

func TestSerializeCatalogRoundTrip(t *testing.T) {
	catalog := DefaultTimelineCatalog()

	catalog.views[display.PanelWeather] = TimelineStep{
		Duration: 3 * time.Minute,
		Refresh:  45 * time.Second,
	}

	catalog.presets["commute"] = entriesForNames([]string{"weather", "bus.map", "bus.progress"})
	catalog.order = entriesForNames([]string{"weather", "bus.progress"})
	catalog.intro = entriesForNames([]string{"weather"})

	data, err := serializeCatalog(catalog)
	if err != nil {
		t.Fatalf("serializeCatalog: %v", err)
	}

	if strings.Contains(string(data), "interleaved") {
		t.Fatalf("serialized file should not persist unmodified builtin presets:\n%s", data)
	}

	if strings.Contains(string(data), "bus.map") {
		t.Fatalf("serialized file should not use dot notation:\n%s", data)
	}

	path := filepath.Join(t.TempDir(), "board-timeline.yml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	reloaded, err := LoadTimelineCatalog(path)
	if err != nil {
		t.Fatalf("LoadTimelineCatalog: %v", err)
	}

	if reloaded.views[display.PanelWeather].Duration != 3*time.Minute ||
		reloaded.views[display.PanelWeather].Refresh != 45*time.Second {
		t.Fatalf("reloaded weather settings = %+v", reloaded.views[display.PanelWeather])
	}

	if len(reloaded.presets["commute"]) != 3 {
		t.Fatalf("reloaded commute preset = %v", reloaded.presets["commute"])
	}

	if len(reloaded.intro) != 1 || reloaded.intro[0].Name != "weather" {
		t.Fatalf("reloaded intro = %v, want [weather]", reloaded.intro)
	}

	timeline, err := reloaded.Resolve("")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if len(timeline.Intro) != 1 || timeline.Intro[0].Panel != display.PanelWeather {
		t.Fatalf("reloaded timeline intro = %+v", timeline.Intro)
	}

	if len(timeline.Steps) != 2 || timeline.Steps[1].Panel != display.PanelBusProgress {
		t.Fatalf("reloaded order timeline = %+v", timeline.Steps)
	}
}

func TestNormalizeViewEntries(t *testing.T) {
	entries, err := normalizeViewEntries(entriesForNames([]string{"bus", "map", "progress"}))
	if err != nil {
		t.Fatalf("normalizeViewEntries: %v", err)
	}

	want := []string{"bus.schedule", "bus.map", "bus.progress"}
	for i := range want {
		if entries[i].Name != want[i] {
			t.Fatalf("normalized = %v, want %v", entries, want)
		}
	}

	if _, err := normalizeViewEntries(nil); err == nil {
		t.Fatal("normalizeViewEntries(empty) = nil error")
	}

	if _, err := normalizeViewEntries(entriesForNames([]string{"weather", "mup"})); err == nil {
		t.Fatal("normalizeViewEntries(unknown) = nil error")
	}

	longOrder := make([]viewEntry, 21)
	for i := range longOrder {
		longOrder[i] = viewEntry{Name: "weather"}
	}

	if _, err := normalizeViewEntries(longOrder); err == nil {
		t.Fatal("normalizeViewEntries(21 views) = nil error")
	}
}

var zeroInt64 = int64(0)

func TestBoardServiceIntroMutations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "board-timeline.yml")
	board := NewBoardService("", nil, nil, nil, nil, path)

	settings := board.TimelineSettings()
	if len(settings.Intro) != 0 {
		t.Fatalf("default intro = %v, want empty", settings.Intro)
	}

	if err := board.SetTimelineOrderViews([]string{"bus.schedule", "bus.map", "bus.progress"}); err != nil {
		t.Fatalf("SetTimelineOrderViews: %v", err)
	}

	// Optional: configuring the intro must not disturb the order.
	if err := board.SetTimelineIntro([]string{"weather"}); err != nil {
		t.Fatalf("SetTimelineIntro: %v", err)
	}

	settings = board.TimelineSettings()
	if len(settings.Intro) != 1 || settings.Intro[0] != "weather" {
		t.Fatalf("intro = %v, want [weather]", settings.Intro)
	}

	if settings.Order[0] != "bus.schedule" || len(settings.Order) != 3 {
		t.Fatalf("order after intro update = %v", settings.Order)
	}

	timeline, err := board.ResolveTimeline("")
	if err != nil {
		t.Fatalf("ResolveTimeline: %v", err)
	}

	if len(timeline.Intro) != 1 || timeline.Intro[0].Panel != display.PanelWeather {
		t.Fatalf("resolved intro = %+v", timeline.Intro)
	}

	if len(timeline.Steps) != 3 || timeline.Steps[0].Panel != display.PanelBus {
		t.Fatalf("resolved order = %+v", timeline.Steps)
	}

	catalog, err := LoadTimelineCatalog(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	if len(catalog.intro) != 1 || catalog.intro[0].Name != "weather" {
		t.Fatalf("reloaded intro = %v, want [weather]", catalog.intro)
	}

	reloaded, err := catalog.Resolve("")
	if err != nil {
		t.Fatalf("Resolve after reload: %v", err)
	}

	if len(reloaded.Intro) != 1 || reloaded.Intro[0].Panel != display.PanelWeather {
		t.Fatalf("reloaded timeline intro = %+v", reloaded.Intro)
	}

	// Clearing the intro restores pure loop behaviour.
	if err := board.SetTimelineIntro(nil); err != nil {
		t.Fatalf("SetTimelineIntro(nil): %v", err)
	}

	settings = board.TimelineSettings()
	if len(settings.Intro) != 0 {
		t.Fatalf("intro after clear = %v, want empty", settings.Intro)
	}

	if settings.Order[0] != "bus.schedule" {
		t.Fatalf("order disturbed by intro clear = %v", settings.Order)
	}

	// Invalid view names are rejected and change nothing.
	if err := board.SetTimelineIntro([]string{"mup"}); err == nil {
		t.Fatal("SetTimelineIntro(unknown view) = nil error")
	}

	if settings := board.TimelineSettings(); len(settings.Intro) != 0 {
		t.Fatalf("intro after invalid update = %v, want empty", settings.Intro)
	}

	tooManyViews := make([]string, 21)
	for i := range tooManyViews {
		tooManyViews[i] = "weather"
	}

	if err := board.SetTimelineIntro(tooManyViews); err == nil {
		t.Fatal("SetTimelineIntro(21 views) = nil error")
	}
}

func TestSerializeCatalogGroupedEntriesWithOverrides(t *testing.T) {
	catalog := DefaultTimelineCatalog()

	duration := 3 * time.Minute
	refresh := 15 * time.Second

	catalog.order = []viewEntry{
		{Name: "weather", Duration: &duration},
		{Name: "bus.schedule", Refresh: &refresh},
	}
	catalog.presets["commute"] = []viewEntry{
		{Name: "bus.map", Duration: &duration, Refresh: &refresh},
		{Name: "bus.progress"},
	}

	data, err := serializeCatalog(catalog)
	if err != nil {
		t.Fatalf("serializeCatalog: %v", err)
	}

	rendered := string(data)

	if strings.Contains(rendered, "bus.map") ||
		strings.Contains(rendered, "bus.schedule") ||
		strings.Contains(rendered, "bus.progress") {
		t.Fatalf("serialized file should use grouped entries:\n%s", rendered)
	}

	path := filepath.Join(t.TempDir(), "board-timeline.yml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	reloaded, err := LoadTimelineCatalog(path)
	if err != nil {
		t.Fatalf("LoadTimelineCatalog: %v", err)
	}

	timeline, err := reloaded.Resolve("")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	if len(timeline.Steps) != 2 ||
		timeline.Steps[0].Duration != 3*time.Minute ||
		timeline.Steps[0].Refresh != time.Minute ||
		timeline.Steps[1].Refresh != 15*time.Second {
		t.Fatalf("reloaded order timeline = %+v", timeline.Steps)
	}

	presetTimeline, err := reloaded.Resolve("commute")
	if err != nil {
		t.Fatalf("Resolve(commute): %v", err)
	}

	if len(presetTimeline.Steps) != 2 ||
		presetTimeline.Steps[0].Duration != 3*time.Minute ||
		presetTimeline.Steps[0].Refresh != 15*time.Second ||
		presetTimeline.Steps[1].Refresh != 25*time.Second {
		t.Fatalf("reloaded commute timeline = %+v", presetTimeline.Steps)
	}
}

func TestBoardServiceSetTimelineOrderWithOverrides(t *testing.T) {
	path := filepath.Join(t.TempDir(), "board-timeline.yml")
	board := NewBoardService("", nil, nil, nil, nil, path)

	// The per-entry compact grammar flows through the API into the YAML file.
	if err := board.SetTimelineOrder("weather:2m:30s,bus.map:45s:15s"); err != nil {
		t.Fatalf("SetTimelineOrder: %v", err)
	}

	settings := board.TimelineSettings()
	if len(settings.Order) != 2 ||
		settings.Order[0] != "weather:2m:30s" ||
		settings.Order[1] != "bus.map:45s:15s" {
		t.Fatalf("settings order = %v", settings.Order)
	}

	timeline, err := board.ResolveTimeline("")
	if err != nil {
		t.Fatalf("ResolveTimeline: %v", err)
	}

	if len(timeline.Steps) != 2 ||
		timeline.Steps[0].Duration != 2*time.Minute ||
		timeline.Steps[0].Refresh != 30*time.Second ||
		timeline.Steps[1].Duration != 45*time.Second ||
		timeline.Steps[1].Refresh != 15*time.Second {
		t.Fatalf("resolved timeline = %+v", timeline.Steps)
	}

	catalog, err := LoadTimelineCatalog(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}

	if strings.Contains(string(mustReadFile(t, path)), "bus.map") {
		t.Fatalf("persisted file should use grouped entries:\n%s", mustReadFile(t, path))
	}

	reloaded, err := catalog.Resolve("")
	if err != nil {
		t.Fatalf("resolve after reload: %v", err)
	}

	if len(reloaded.Steps) != 2 || reloaded.Steps[1].Duration != 45*time.Second {
		t.Fatalf("reloaded timeline = %+v", reloaded.Steps)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	return data
}
