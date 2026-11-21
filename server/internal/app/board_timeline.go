package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/display"
	"gopkg.in/yaml.v3"
)

// A board timeline is the flattened, looping sequence of board views
// an alarm casts: a list of steps, each with how long it owns the
// screen and how often it re-renders (rotating its items, for views
// like bus progress). The catalog in data/board-timeline.yml defines
// view settings and named order presets; each alarm selects one, and
// the timeline wraps modulo the alarm's display window.

const (
	// timelineRefreshFloor keeps image swaps above cast-load latency.
	timelineRefreshFloor = 10 * time.Second
)

type TimelineStep struct {
	Panel    display.Panel
	Duration time.Duration // total time on screen per pass
	Refresh  time.Duration // re-render / item-rotation cadence
}

type Timeline struct {
	Steps []TimelineStep
}

// BoardSlot is what the timeline says to render at a moment.
type BoardSlot struct {
	Panel display.Panel
	Step  int // index into the timeline's steps
	Tick  int // refresh tick index within the current step
}

// At reports the active step and tick for t, relative to the session
// start. The timeline loops: slots wrap modulo the total duration.
func (tl *Timeline) At(start, t time.Time) BoardSlot {
	elapsed := t.Sub(start)

	total := tl.Total()
	if total <= 0 {
		return BoardSlot{}
	}

	offset := elapsed % total
	if offset < 0 {
		offset += total
	}

	for stepIndex, step := range tl.Steps {
		if offset < step.Duration {
			tickIndex := int(offset / step.Refresh)

			return BoardSlot{
				Panel: step.Panel,
				Step:  stepIndex,
				Tick:  tickIndex,
			}
		}

		offset -= step.Duration
	}

	// Unreachable when steps are validated to sum positive.
	return BoardSlot{Panel: tl.Steps[len(tl.Steps)-1].Panel}
}

// Total is the full cycle length of the timeline.
func (tl *Timeline) Total() time.Duration {
	var total time.Duration

	for _, step := range tl.Steps {
		total += step.Duration
	}

	return total
}

// Duration returns the active step's duration, alarm-session
// relative, for the cast scheduler.
func (tl *Timeline) Duration(slot BoardSlot) time.Duration {
	if slot.Step < 0 || slot.Step >= len(tl.Steps) {
		return 0
	}

	return tl.Steps[slot.Step].Duration
}

// NextBoundary returns when the slot after t begins: the next refresh
// tick within a step, or the next step. The cast scheduler and the
// preview tool both walk timelines with it.
func (tl *Timeline) NextBoundary(start, t time.Time) time.Time {
	total := tl.Total()
	if total <= 0 {
		return t.Add(time.Minute)
	}

	elapsed := t.Sub(start)
	if elapsed < 0 {
		elapsed = 0
	}

	cycleBase := elapsed - elapsed%total

	offset := elapsed % total

	stepStart := time.Duration(0)

	for _, step := range tl.Steps {
		if offset >= step.Duration {
			offset -= step.Duration
			stepStart += step.Duration

			continue
		}

		nextTick := (offset/step.Refresh + 1) * step.Refresh

		if nextTick < step.Duration {
			return start.Add(cycleBase + stepStart + nextTick)
		}

		return start.Add(cycleBase + stepStart + step.Duration)
	}

	return start.Add(cycleBase + total)
}

func (tl *Timeline) validate() error {
	if len(tl.Steps) == 0 {
		return fmt.Errorf("board timeline is empty")
	}

	for i, step := range tl.Steps {
		if step.Duration < time.Second {
			return fmt.Errorf(
				"timeline step %d (%s) duration %s is below 1s",
				i,
				step.Panel,
				step.Duration,
			)
		}

		if step.Refresh < timelineRefreshFloor {
			return fmt.Errorf(
				"timeline step %d (%s) refresh %s is below %s",
				i,
				step.Panel,
				step.Refresh,
				timelineRefreshFloor,
			)
		}
	}

	for i, step := range tl.Steps {
		if step.Refresh > step.Duration {
			return fmt.Errorf(
				"timeline step %d (%s) refresh %s exceeds duration %s",
				i,
				step.Panel,
				step.Refresh,
				step.Duration,
			)
		}
	}

	return nil
}

// defaultViewSettings are the built-in per-view duration and refresh
// used whenever the timeline file omits them.
func defaultViewSettings(panel display.Panel) (duration time.Duration, refresh time.Duration) {
	if panel == display.PanelBusProgress {
		return 2 * time.Minute, 25 * time.Second
	}

	return 2 * time.Minute, 60 * time.Second
}

func panelForViewName(name string) (display.Panel, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "weather":
		return display.PanelWeather, nil
	case "bus", "schedule", "timetable", "bus.schedule", "bus.timetable":
		return display.PanelBus, nil
	case "map", "bus.map":
		return display.PanelBusMap, nil
	case "progress", "bus.progress", "track", "bus.track":
		return display.PanelBusProgress, nil
	default:
		return display.PanelWeather, fmt.Errorf("unknown board view %q", name)
	}
}

// TimelineCatalog is the parsed board-timeline.yml: view settings,
// named presets, and the default order to play.
type TimelineCatalog struct {
	views   map[display.Panel]TimelineStep
	presets map[string][]string
	order   []string
}

type timelineFile struct {
	Views   yaml.Node            `yaml:"views"`
	Order   yaml.Node            `yaml:"order"`
	Presets map[string]yaml.Node `yaml:"presets"`
}

// LoadTimelineCatalog reads the timeline file; a missing file is not
// an error, it means "fall back to the next resolution source".
func LoadTimelineCatalog(path string) (*TimelineCatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("reading board timeline: %w", err)
	}

	var file timelineFile

	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing board timeline: %w", err)
	}

	catalog := &TimelineCatalog{
		views:   defaultViews(),
		presets: map[string][]string{},
	}

	for name, entries := range builtinPresets {
		catalog.presets[name] = append([]string(nil), entries...)
	}

	if err := catalog.loadViews(&file.Views); err != nil {
		return nil, err
	}

	for name, node := range file.Presets {
		order, err := decodeViewList(&node)
		if err != nil {
			return nil, fmt.Errorf("preset %q: %w", name, err)
		}

		catalog.presets[name] = order
	}

	order, err := catalog.decodeOrder(&file.Order)
	if err != nil {
		return nil, err
	}

	catalog.order = order

	if err := catalog.validateNames(); err != nil {
		return nil, err
	}

	return catalog, nil
}

// validateNames fails fast at load time on typos in the default order
// or any preset, rather than when an alarm tries to use them.
func (c *TimelineCatalog) validateNames() error {
	if _, err := c.resolveNames(c.order); err != nil {
		return fmt.Errorf("order: %w", err)
	}

	names := make([]string, 0, len(c.presets))
	for name := range c.presets {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if _, err := c.resolveNames(c.presets[name]); err != nil {
			return fmt.Errorf("preset %q: %w", name, err)
		}
	}

	return nil
}

func (c *TimelineCatalog) resolveNames(viewNames []string) ([]display.Panel, error) {
	panels := make([]display.Panel, 0, len(viewNames))

	for _, name := range viewNames {
		panel, err := panelForViewName(name)
		if err != nil {
			return nil, err
		}

		panels = append(panels, panel)
	}

	return panels, nil
}

func defaultViews() map[display.Panel]TimelineStep {
	views := make(map[display.Panel]TimelineStep)

	for _, panel := range []display.Panel{
		display.PanelWeather,
		display.PanelBus,
		display.PanelBusMap,
		display.PanelBusProgress,
	} {
		duration, refresh := defaultViewSettings(panel)

		views[panel] = TimelineStep{Duration: duration, Refresh: refresh}
	}

	return views
}

var builtinPresets = map[string][]string{
	"default":     {"weather", "bus.map", "bus.schedule", "bus.progress"},
	"interleaved": {"weather", "bus.map", "weather", "bus.schedule", "weather", "bus.progress"},
	"bus-focus":   {"bus.map", "bus.schedule", "bus.progress", "weather"},
}

// decodeOrder accepts either a preset name or an explicit list.
func (c *TimelineCatalog) decodeOrder(node *yaml.Node) ([]string, error) {
	// Absent order decodes to a zero-valued node: use the default.
	if node == nil || node.Kind == 0 {
		return c.presets["default"], nil
	}

	if node.Kind == yaml.ScalarNode {
		var name string

		if err := node.Decode(&name); err != nil {
			return nil, fmt.Errorf("parsing order: %w", err)
		}

		if strings.TrimSpace(name) == "" {
			return c.presets["default"], nil
		}

		preset, err := c.preset(strings.TrimSpace(name))
		if err != nil {
			return nil, err
		}

		return preset, nil
	}

	return decodeViewList(node)
}

func (c *TimelineCatalog) preset(name string) ([]string, error) {
	if entries, ok := c.presets[name]; ok {
		return entries, nil
	}

	available := make([]string, 0, len(c.presets))
	for name := range c.presets {
		available = append(available, name)
	}
	sort.Strings(available)

	return nil, fmt.Errorf(
		"unknown board timeline preset %q (available: %s)",
		name,
		strings.Join(available, ", "),
	)
}

func decodeViewList(node *yaml.Node) ([]string, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("expected a list of view names")
	}

	var views []string

	if err := node.Decode(&views); err != nil {
		return nil, fmt.Errorf("parsing view list: %w", err)
	}

	if len(views) == 0 {
		return nil, fmt.Errorf("view list is empty")
	}

	return views, nil
}

type viewSettingsFile struct {
	Duration string `yaml:"duration"`
	Refresh  string `yaml:"refresh"`
}

// loadViews accepts two shapes: a leaf view (weather: {duration: 1m})
// or one nested level (bus: {map: {...}, progress: {...}}).
func (c *TimelineCatalog) loadViews(views *yaml.Node) error {
	// Absent sections decode to zero-valued nodes; nothing to apply.
	if views == nil || views.Kind == 0 {
		return nil
	}

	if views.Kind == yaml.ScalarNode && views.Value == "" {
		return nil
	}

	if views.Kind != yaml.MappingNode {
		return fmt.Errorf("views must be a mapping of view settings")
	}

	for i := 0; i < len(views.Content); i += 2 {
		name := views.Content[i].Value
		entry := views.Content[i+1]

		if isViewSettingsMapping(entry) {
			if err := c.applyViewSettings(name, entry); err != nil {
				return err
			}

			continue
		}

		if entry.Kind == yaml.MappingNode {
			for j := 0; j < len(entry.Content); j += 2 {
				subName := entry.Content[j].Value
				settings := entry.Content[j+1]

				if !isViewSettingsMapping(settings) {
					return fmt.Errorf(
						"view %s.%s: expected duration/refresh settings",
						name,
						subName,
					)
				}

				qualifiedName := name + "." + subName

				if err := c.applyViewSettings(qualifiedName, settings); err != nil {
					return err
				}
			}

			continue
		}

		return fmt.Errorf("view %q: expected view settings", name)
	}

	return nil
}

// isViewSettingsMapping reports whether a node looks like a view
// settings block (only duration and/or refresh keys), as opposed to a
// nesting of further view names.
func isViewSettingsMapping(node *yaml.Node) bool {
	if node.Kind != yaml.MappingNode {
		return false
	}

	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i].Value

		switch key {
		case "duration", "refresh":
		default:
			return false
		}
	}

	return true
}

func (c *TimelineCatalog) applyViewSettings(name string, node *yaml.Node) error {
	panel, err := panelForViewName(name)
	if err != nil {
		return err
	}

	var settings viewSettingsFile

	if err := node.Decode(&settings); err != nil {
		return fmt.Errorf("view %s: %w", name, err)
	}

	step, ok := c.views[panel]
	if !ok {
		duration, refresh := defaultViewSettings(panel)

		step = TimelineStep{Duration: duration, Refresh: refresh}
	}

	if settings.Duration != "" {
		duration, err := time.ParseDuration(settings.Duration)
		if err != nil || duration < time.Second {
			return fmt.Errorf(
				"view %s: invalid duration %q",
				name,
				settings.Duration,
			)
		}

		step.Duration = duration
	}

	if settings.Refresh != "" {
		refresh, err := time.ParseDuration(settings.Refresh)
		if err != nil || refresh < timelineRefreshFloor {
			return fmt.Errorf(
				"view %s: invalid refresh %q (minimum %s)",
				name,
				settings.Refresh,
				timelineRefreshFloor,
			)
		}

		step.Refresh = refresh
	}

	c.views[panel] = step

	return nil
}

// Resolve builds the timeline for a per-alarm order: "" uses the
// catalog's default order, a bare name selects a preset, and anything
// else is a comma-separated view list.
func (c *TimelineCatalog) Resolve(order string) (*Timeline, error) {
	var names []string

	trimmed := strings.TrimSpace(order)

	switch {
	case trimmed == "":
		names = c.order
	case !strings.ContainsAny(trimmed, ",."):
		preset, err := c.preset(trimmed)
		if err != nil {
			return nil, err
		}

		names = preset
	default:
		names = strings.Split(trimmed, ",")
	}

	timeline := &Timeline{}

	for _, name := range names {
		panel, err := panelForViewName(name)
		if err != nil {
			return nil, err
		}

		settings, ok := c.views[panel]
		if !ok {
			duration, refresh := defaultViewSettings(panel)

			settings = TimelineStep{Duration: duration, Refresh: refresh}
		}

		timeline.Steps = append(timeline.Steps, TimelineStep{
			Panel:    panel,
			Duration: settings.Duration,
			Refresh:  settings.Refresh,
		})
	}

	if err := timeline.validate(); err != nil {
		return nil, err
	}

	return timeline, nil
}

// OpenBoardTimeline loads the board timeline configuration: the
// yml catalog when present, otherwise the legacy BOARD_SCHEDULE
// value, otherwise the built-in default. It returns the server's
// default timeline plus the catalog (never nil) for per-alarm preset
// resolution. Callers decide how hard to fail on err.
func OpenBoardTimeline(
	dataDir string,
	legacySpec string,
) (timeline *Timeline, catalog *TimelineCatalog, err error) {
	catalog, err = LoadTimelineCatalog(
		filepath.Join(dataDir, "board-timeline.yml"),
	)
	if err != nil {
		return nil, nil, err
	}

	if catalog != nil {
		timeline, err = catalog.Resolve("")
		if err != nil {
			return nil, nil, err
		}

		return timeline, catalog, nil
	}

	// No timeline file: fall back to the legacy schedule, but keep the
	// built-in catalog available so per-alarm presets still resolve.
	if strings.TrimSpace(legacySpec) != "" {
		schedule, err := ParseBoardSchedule(legacySpec)
		if err != nil {
			return nil, nil, err
		}

		return NewTimelineFromLegacySchedule(schedule), DefaultTimelineCatalog(), nil
	}

	catalog = DefaultTimelineCatalog()

	timeline, err = catalog.Resolve("")

	return timeline, catalog, err
}

// DefaultTimelineCatalog is the built-in fallback used when no
// timeline file and no legacy schedule are configured.
func DefaultTimelineCatalog() *TimelineCatalog {
	catalog := &TimelineCatalog{
		views:   defaultViews(),
		presets: map[string][]string{},
	}

	for name, entries := range builtinPresets {
		catalog.presets[name] = append([]string(nil), entries...)
	}

	catalog.order = catalog.presets["default"]

	return catalog
}

// NewTimelineFromLegacySchedule converts a parsed BOARD_SCHEDULE
// value into a timeline, preserving the legacy refresh behaviour of
// one re-render per minute.
func NewTimelineFromLegacySchedule(schedule BoardSchedule) *Timeline {
	timeline := &Timeline{}

	for _, segment := range schedule {
		timeline.Steps = append(timeline.Steps, TimelineStep{
			Panel:    segment.Panel,
			Duration: time.Duration(segment.Minutes) * time.Minute,
			Refresh:  time.Minute,
		})
	}

	return timeline
}
