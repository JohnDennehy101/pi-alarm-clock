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

const (
	timelineRefreshFloor = 10 * time.Second
)

type TimelineStep struct {
	Panel    display.Panel
	Duration time.Duration
	Refresh  time.Duration
}

type Timeline struct {
	Intro []TimelineStep
	Steps []TimelineStep
}

type BoardSlot struct {
	Panel display.Panel
	Step  int
	Tick  int
}

func (tl *Timeline) At(start, t time.Time) BoardSlot {
	elapsed := t.Sub(start)
	if elapsed < 0 {
		elapsed = 0
	}

	introTotal := tl.IntroTotal()
	if elapsed < introTotal {
		offset := elapsed

		for stepIndex, step := range tl.Intro {
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
	}

	total := tl.Total()
	if total <= 0 {
		return BoardSlot{}
	}

	offset := (elapsed - introTotal) % total
	if offset < 0 {
		offset += total
	}

	introLen := len(tl.Intro)

	for stepIndex, step := range tl.Steps {
		if offset < step.Duration {
			tickIndex := int(offset / step.Refresh)

			return BoardSlot{
				Panel: step.Panel,
				Step:  introLen + stepIndex,
				Tick:  tickIndex,
			}
		}

		offset -= step.Duration
	}

	return BoardSlot{Panel: tl.Steps[len(tl.Steps)-1].Panel}
}

func (tl *Timeline) Total() time.Duration {
	var total time.Duration

	for _, step := range tl.Steps {
		total += step.Duration
	}

	return total
}

func (tl *Timeline) IntroTotal() time.Duration {
	var total time.Duration

	for _, step := range tl.Intro {
		total += step.Duration
	}

	return total
}

func (tl *Timeline) Duration(slot BoardSlot) time.Duration {
	if slot.Step < 0 {
		return 0
	}

	introLen := len(tl.Intro)

	if slot.Step < introLen {
		return tl.Intro[slot.Step].Duration
	}

	if slot.Step >= introLen+len(tl.Steps) {
		return 0
	}

	return tl.Steps[slot.Step-introLen].Duration
}

func (tl *Timeline) NextBoundary(start, t time.Time) time.Time {
	introTotal := tl.IntroTotal()
	total := tl.Total()

	if introTotal <= 0 && total <= 0 {
		return t.Add(time.Minute)
	}

	elapsed := t.Sub(start)
	if elapsed < 0 {
		elapsed = 0
	}

	if elapsed < introTotal {
		offset := elapsed
		stepStart := time.Duration(0)

		for _, step := range tl.Intro {
			if offset >= step.Duration {
				offset -= step.Duration
				stepStart += step.Duration

				continue
			}

			nextTick := (offset/step.Refresh + 1) * step.Refresh

			if nextTick < step.Duration {
				return start.Add(stepStart + nextTick)
			}

			return start.Add(stepStart + step.Duration)
		}

		return start.Add(introTotal)
	}

	if total <= 0 {
		return t.Add(time.Minute)
	}

	loopElapsed := elapsed - introTotal

	cycleBase := loopElapsed - loopElapsed%total

	offset := loopElapsed % total

	stepStart := time.Duration(0)

	for _, step := range tl.Steps {
		if offset >= step.Duration {
			offset -= step.Duration
			stepStart += step.Duration

			continue
		}

		nextTick := (offset/step.Refresh + 1) * step.Refresh

		if nextTick < step.Duration {
			return start.Add(introTotal + cycleBase + stepStart + nextTick)
		}

		return start.Add(introTotal + cycleBase + stepStart + step.Duration)
	}

	return start.Add(introTotal + cycleBase + total)
}

func (tl *Timeline) validate() error {
	if len(tl.Steps) == 0 {
		return fmt.Errorf("board timeline is empty")
	}

	for _, steps := range [][]TimelineStep{tl.Intro, tl.Steps} {
		for i, step := range steps {
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
	}

	return nil
}

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

type TimelineCatalog struct {
	views   map[display.Panel]TimelineStep
	presets map[string][]string
	order   []string
	intro   []string
}

type timelineFile struct {
	Views   yaml.Node            `yaml:"views"`
	Order   yaml.Node            `yaml:"order"`
	Intro   yaml.Node            `yaml:"intro"`
	Presets map[string]yaml.Node `yaml:"presets"`
}

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

	intro, err := catalog.decodeIntro(&file.Intro)
	if err != nil {
		return nil, err
	}

	catalog.intro = intro

	if err := catalog.validateNames(); err != nil {
		return nil, err
	}

	return catalog, nil
}

func (c *TimelineCatalog) validateNames() error {
	if _, err := c.resolveNames(c.order); err != nil {
		return fmt.Errorf("order: %w", err)
	}

	if _, err := c.resolveNames(c.intro); err != nil {
		return fmt.Errorf("intro: %w", err)
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

func (c *TimelineCatalog) decodeOrder(node *yaml.Node) ([]string, error) {
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

func (c *TimelineCatalog) decodeIntro(node *yaml.Node) ([]string, error) {
	if node == nil || node.Kind == 0 {
		return nil, nil
	}

	if node.Kind == yaml.ScalarNode {
		if node.Tag == "!!null" || strings.TrimSpace(node.Value) == "" {
			return nil, nil
		}

		preset, err := c.preset(strings.TrimSpace(node.Value))
		if err != nil {
			return nil, fmt.Errorf("intro: %w", err)
		}

		return append([]string(nil), preset...), nil
	}

	if node.Kind == yaml.SequenceNode && len(node.Content) == 0 {
		return nil, nil
	}

	views, err := decodeViewList(node)
	if err != nil {
		return nil, fmt.Errorf("intro: %w", err)
	}

	return views, nil
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

func (c *TimelineCatalog) loadViews(views *yaml.Node) error {
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

func (c *TimelineCatalog) stepsForNames(names []string) ([]TimelineStep, error) {
	steps := make([]TimelineStep, 0, len(names))

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

		steps = append(steps, TimelineStep{
			Panel:    panel,
			Duration: settings.Duration,
			Refresh:  settings.Refresh,
		})
	}

	return steps, nil
}

func (c *TimelineCatalog) orderNames(order string) ([]string, error) {
	trimmed := strings.TrimSpace(order)

	switch {
	case trimmed == "":
		return c.order, nil
	case !strings.ContainsAny(trimmed, ",."):
		return c.preset(trimmed)
	default:
		return strings.Split(trimmed, ","), nil
	}
}

func (c *TimelineCatalog) introFromExpression(expr string) ([]string, error) {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return nil, nil
	}

	if !strings.Contains(trimmed, ",") {
		if entries, ok := c.presets[trimmed]; ok {
			return append([]string(nil), entries...), nil
		}
	}

	return normalizeViewList(strings.Split(trimmed, ","))
}

func (c *TimelineCatalog) build(orderNames, introNames []string) (*Timeline, error) {
	timeline := &Timeline{}

	introSteps, err := c.stepsForNames(introNames)
	if err != nil {
		return nil, fmt.Errorf("intro: %w", err)
	}

	timeline.Intro = introSteps

	loopSteps, err := c.stepsForNames(orderNames)
	if err != nil {
		return nil, err
	}

	timeline.Steps = loopSteps

	if err := timeline.validate(); err != nil {
		return nil, err
	}

	return timeline, nil
}

func (c *TimelineCatalog) Resolve(order string) (*Timeline, error) {
	names, err := c.orderNames(order)
	if err != nil {
		return nil, err
	}

	return c.build(names, c.intro)
}

func (c *TimelineCatalog) ResolveForCast(order, intro string) (*Timeline, error) {
	names, err := c.orderNames(order)
	if err != nil {
		return nil, err
	}

	introNames, err := c.introFromExpression(intro)
	if err != nil {
		return nil, fmt.Errorf("board intro %q: %w", intro, err)
	}

	return c.build(names, introNames)
}

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
