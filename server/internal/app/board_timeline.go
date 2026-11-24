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

type viewEntry struct {
	Name     string
	Duration *time.Duration
	Refresh  *time.Duration
}

func (e viewEntry) HasOverrides() bool {
	return e.Duration != nil || e.Refresh != nil
}

func (e viewEntry) Canonical() (viewEntry, error) {
	panel, err := panelForViewName(e.Name)
	if err != nil {
		return viewEntry{}, err
	}

	return viewEntry{
		Name:     canonicalViewName(panel),
		Duration: e.Duration,
		Refresh:  e.Refresh,
	}, nil
}

func (e viewEntry) String() string {
	rendered := e.Name

	if e.Duration != nil {
		rendered += ":" + durationString(*e.Duration)
	}

	if e.Refresh != nil {
		rendered += ":" + durationString(*e.Refresh)
	}

	return rendered
}

func parseViewExpression(value string) (viewEntry, error) {
	name, parts, err := splitViewExpression(value)
	if err != nil {
		return viewEntry{}, err
	}

	return parseViewEntryParts(name, parts)
}

func splitViewExpression(value string) (string, []string, error) {
	trimmed := strings.TrimSpace(value)
	parts := strings.Split(trimmed, ":")

	if len(parts) > 3 {
		return "", nil, fmt.Errorf(
			"invalid view expression %q: expected view[:duration[:refresh]]",
			value,
		)
	}

	name := strings.TrimSpace(parts[0])
	if name == "" {
		return "", nil, fmt.Errorf("invalid view expression %q: view name is empty", value)
	}

	return name, parts, nil
}

type durationFieldSpec struct {
	field  string
	raw    string
	target **time.Duration
}

func parseViewEntryParts(name string, parts []string) (viewEntry, error) {
	entry := viewEntry{Name: name}
	specs := buildDurationSpecs(parts, &entry)

	for _, spec := range specs {
		if err := assignDurationField(name, spec.field, spec.raw, spec.target); err != nil {
			return viewEntry{}, err
		}
	}

	return entry, nil
}

func buildDurationSpecs(parts []string, entry *viewEntry) []durationFieldSpec {
	specs := make([]durationFieldSpec, 0, 2)

	if len(parts) >= 2 {
		specs = append(specs, durationFieldSpec{
			field:  "duration",
			raw:    parts[1],
			target: &entry.Duration,
		})
	}

	if len(parts) == 3 {
		specs = append(specs, durationFieldSpec{
			field:  "refresh",
			raw:    parts[2],
			target: &entry.Refresh,
		})
	}

	return specs
}

func assignDurationField(
	viewName string,
	fieldName string,
	raw string,
	target **time.Duration,
) error {
	parsed, err := parseEntryDuration(viewName, fieldName, raw)
	if err != nil {
		return err
	}

	*target = &parsed

	return nil
}

func parseEntryDuration(name, field string, value string) (time.Duration, error) {
	trimmed := strings.TrimSpace(value)

	if trimmed == "" {
		return 0, fmt.Errorf("view %s: missing %s", name, field)
	}

	parsed, err := time.ParseDuration(trimmed)
	if err != nil || parsed < time.Second {
		return 0, fmt.Errorf(
			"view %s: invalid %s %q (expected 1s or more, e.g. 30s, 2m)",
			name,
			field,
			value,
		)
	}

	if field == "refresh" && parsed < timelineRefreshFloor {
		return 0, fmt.Errorf(
			"view %s: invalid refresh %q (minimum %s)",
			name,
			value,
			timelineRefreshFloor,
		)
	}

	return parsed, nil
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
	elapsed := max(t.Sub(start), 0)

	introTotal := tl.IntroTotal()
	if elapsed < introTotal {
		return slotAtOffset(tl.Intro, elapsed, 0, BoardSlot{})
	}

	total := tl.Total()
	if total <= 0 {
		return BoardSlot{}
	}

	offset := (elapsed - introTotal) % total
	if offset < 0 {
		offset += total
	}

	fallback := BoardSlot{}
	if len(tl.Steps) > 0 {
		fallback = BoardSlot{Panel: tl.Steps[len(tl.Steps)-1].Panel}
	}

	return slotAtOffset(tl.Steps, offset, len(tl.Intro), fallback)
}

func slotAtOffset(
	steps []TimelineStep,
	offset time.Duration,
	stepOffset int,
	fallback BoardSlot,
) BoardSlot {
	remaining := offset

	for idx, step := range steps {
		if remaining < step.Duration {
			return BoardSlot{
				Panel: step.Panel,
				Step:  stepOffset + idx,
				Tick:  int(remaining / step.Refresh),
			}
		}

		remaining -= step.Duration
	}

	return fallback
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

	elapsed := max(t.Sub(start), 0)

	if elapsed < introTotal {
		delta := nextBoundaryInSteps(tl.Intro, elapsed, introTotal)
		return start.Add(delta)
	}

	if total <= 0 {
		return t.Add(time.Minute)
	}

	loopElapsed := elapsed - introTotal
	cycleBase := loopElapsed - loopElapsed%total
	offset := loopElapsed % total
	delta := nextBoundaryInSteps(tl.Steps, offset, total)

	return start.Add(introTotal + cycleBase + delta)
}

func nextBoundaryInSteps(
	steps []TimelineStep,
	offset time.Duration,
	sectionTotal time.Duration,
) time.Duration {
	remaining := offset
	stepStart := time.Duration(0)

	for _, step := range steps {
		if remaining >= step.Duration {
			remaining -= step.Duration
			stepStart += step.Duration

			continue
		}

		nextTick := (remaining/step.Refresh + 1) * step.Refresh
		if nextTick < step.Duration {
			return stepStart + nextTick
		}

		return stepStart + step.Duration
	}

	return sectionTotal
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
	presets map[string][]viewEntry
	order   []viewEntry
	intro   []viewEntry
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

	catalog := newCatalogWithDefaults()
	if err := catalog.populateFromFile(&file); err != nil {
		return nil, err
	}

	if err := catalog.validateNames(); err != nil {
		return nil, err
	}

	return catalog, nil
}

func newCatalogWithDefaults() *TimelineCatalog {
	catalog := &TimelineCatalog{
		views:   defaultViews(),
		presets: map[string][]viewEntry{},
	}

	for name, entries := range builtinPresets {
		catalog.presets[name] = entriesForNames(entries)
	}

	return catalog
}

func (c *TimelineCatalog) populateFromFile(file *timelineFile) error {
	if err := c.loadViews(&file.Views); err != nil {
		return err
	}

	if err := c.loadPresets(file.Presets); err != nil {
		return err
	}

	order, err := c.decodeOrder(&file.Order)
	if err != nil {
		return err
	}
	c.order = order

	intro, err := c.decodeIntro(&file.Intro)
	if err != nil {
		return err
	}
	c.intro = intro

	return nil
}

func (c *TimelineCatalog) loadPresets(rawPresets map[string]yaml.Node) error {
	for name, node := range rawPresets {
		order, err := decodeViewList(&node)
		if err != nil {
			return fmt.Errorf("preset %q: %w", name, err)
		}

		c.presets[name] = order
	}

	return nil
}

func (c *TimelineCatalog) validateNames() error {
	if err := c.validateEntries(c.order); err != nil {
		return fmt.Errorf("order: %w", err)
	}

	if err := c.validateEntries(c.intro); err != nil {
		return fmt.Errorf("intro: %w", err)
	}

	names := make([]string, 0, len(c.presets))
	for name := range c.presets {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		if err := c.validateEntries(c.presets[name]); err != nil {
			return fmt.Errorf("preset %q: %w", name, err)
		}
	}

	return nil
}

func (c *TimelineCatalog) validateEntries(entries []viewEntry) error {
	for _, entry := range entries {
		if err := c.validateEntrySettings(entry); err != nil {
			return err
		}

		if _, err := panelForViewName(entry.Name); err != nil {
			return err
		}
	}

	return nil
}

func (c *TimelineCatalog) validateEntrySettings(entry viewEntry) error {
	if entry.Duration != nil && *entry.Duration < time.Second {
		return fmt.Errorf("view %s: duration is below 1s", entry.Name)
	}

	if entry.Refresh != nil && *entry.Refresh < timelineRefreshFloor {
		return fmt.Errorf(
			"view %s: refresh is below %s",
			entry.Name,
			timelineRefreshFloor,
		)
	}

	return nil
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

func entriesForNames(names []string) []viewEntry {
	entries := make([]viewEntry, 0, len(names))

	for _, name := range names {
		entries = append(entries, viewEntry{Name: name})
	}

	return entries
}

func (c *TimelineCatalog) decodeOrder(node *yaml.Node) ([]viewEntry, error) {
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

func (c *TimelineCatalog) decodeIntro(node *yaml.Node) ([]viewEntry, error) {
	if node == nil || node.Kind == 0 {
		return nil, nil
	}

	if node.Kind == yaml.ScalarNode {
		return c.decodeIntroPreset(node)
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

func (c *TimelineCatalog) decodeIntroPreset(node *yaml.Node) ([]viewEntry, error) {
	if node.Tag == "!!null" || strings.TrimSpace(node.Value) == "" {
		return nil, nil
	}

	preset, err := c.preset(strings.TrimSpace(node.Value))
	if err != nil {
		return nil, fmt.Errorf("intro: %w", err)
	}

	return append([]viewEntry(nil), preset...), nil
}

func (c *TimelineCatalog) preset(name string) ([]viewEntry, error) {
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

func decodeViewList(node *yaml.Node) ([]viewEntry, error) {
	if node.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("expected a list of view entries")
	}

	entries := make([]viewEntry, 0, len(node.Content))

	for i, item := range node.Content {
		entry, err := decodeViewEntryNode(item)
		if err != nil {
			return nil, fmt.Errorf("view %d: %w", i+1, err)
		}

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("view list is empty")
	}

	return entries, nil
}

func decodeViewEntryNode(item *yaml.Node) (viewEntry, error) {
	switch item.Kind {
	case yaml.ScalarNode:
		if item.Tag == "!!null" {
			return viewEntry{}, fmt.Errorf("expected a view name")
		}

		name := strings.TrimSpace(item.Value)
		if name == "" {
			return viewEntry{}, fmt.Errorf("expected a view name")
		}

		return viewEntry{Name: name}, nil
	case yaml.MappingNode:
		return decodeViewEntryMapping(item)
	default:
		return viewEntry{}, fmt.Errorf("expected a view name, a view with duration/refresh settings, or a group: view entry")
	}
}

func decodeViewEntryMapping(item *yaml.Node) (viewEntry, error) {
	if len(item.Content) != 2 {
		return viewEntry{}, fmt.Errorf("expected a single view entry, got %d", len(item.Content)/2)
	}

	name := strings.TrimSpace(item.Content[0].Value)
	if name == "" {
		return viewEntry{}, fmt.Errorf("view name is empty")
	}

	value := item.Content[1]
	switch value.Kind {
	case yaml.ScalarNode:
		return decodeGroupedViewScalar(name, value)
	case yaml.MappingNode:
		if isViewSettingsMapping(value) {
			return viewEntryFromSettings(name, value)
		}

		return decodeGroupedViewSettings(name, value)
	default:
		return viewEntry{}, fmt.Errorf(
			"view %q: expected a view name, group view, or duration/refresh settings",
			name,
		)
	}
}

func decodeGroupedViewScalar(prefix string, value *yaml.Node) (viewEntry, error) {
	if value.Tag == "!!null" || strings.TrimSpace(value.Value) == "" {
		return viewEntry{}, fmt.Errorf("view %q: expected a view name", prefix)
	}

	// Grouped name: "bus: schedule" resolves to "bus.schedule".
	return viewEntry{Name: prefix + "." + strings.TrimSpace(value.Value)}, nil
}

func decodeGroupedViewSettings(prefix string, value *yaml.Node) (viewEntry, error) {
	// Grouped view with overrides: "bus: schedule: {duration: 2m}".
	if len(value.Content) != 2 {
		return viewEntry{}, fmt.Errorf(
			"view %q: expected a single entry, got %d",
			prefix,
			len(value.Content)/2,
		)
	}

	viewName := strings.TrimSpace(value.Content[0].Value)
	settings := value.Content[1]

	if settings.Kind != yaml.MappingNode || !isViewSettingsMapping(settings) {
		return viewEntry{}, fmt.Errorf(
			"view %s.%s: expected duration/refresh settings",
			prefix,
			viewName,
		)
	}

	return viewEntryFromSettings(prefix+"."+viewName, settings)
}

func viewEntryFromSettings(name string, node *yaml.Node) (viewEntry, error) {
	entry := viewEntry{Name: name}

	for i := 0; i < len(node.Content); i += 2 {
		key := strings.TrimSpace(node.Content[i].Value)
		value := strings.TrimSpace(node.Content[i+1].Value)

		if err := applyViewEntrySetting(name, &entry, key, value); err != nil {
			return viewEntry{}, err
		}
	}

	if !entry.HasOverrides() {
		return viewEntry{}, fmt.Errorf(
			"view %q: expected duration and/or refresh settings",
			name,
		)
	}

	return entry, nil
}

func applyViewEntrySetting(
	viewName string,
	entry *viewEntry,
	key string,
	value string,
) error {
	switch key {
	case "duration":
		return assignDurationField(viewName, "duration", value, &entry.Duration)
	case "refresh":
		return assignDurationField(viewName, "refresh", value, &entry.Refresh)
	default:
		return nil
	}
}

type viewSettingsFile struct {
	Duration string `yaml:"duration"`
	Refresh  string `yaml:"refresh"`
}

func (c *TimelineCatalog) loadViews(views *yaml.Node) error {
	if err := validateViewsRootNode(views); err != nil {
		return err
	}

	if views == nil {
		return nil
	}

	if views.Kind == 0 {
		return nil
	}

	if views.Kind == yaml.ScalarNode && views.Value == "" {
		return nil
	}

	for i := 0; i < len(views.Content); i += 2 {
		if err := c.loadViewEntry(views.Content[i].Value, views.Content[i+1]); err != nil {
			return err
		}
	}

	return nil
}

func validateViewsRootNode(views *yaml.Node) error {
	if views == nil || views.Kind == 0 {
		return nil
	}

	if views.Kind == yaml.ScalarNode && views.Value == "" {
		return nil
	}

	if views.Kind != yaml.MappingNode {
		return fmt.Errorf("views must be a mapping of view settings")
	}

	return nil
}

func (c *TimelineCatalog) loadViewEntry(name string, entry *yaml.Node) error {
	if isViewSettingsMapping(entry) {
		return c.applyViewSettings(name, entry)
	}

	if entry.Kind == yaml.MappingNode {
		return c.loadViewGroupMapping(name, entry)
	}

	return fmt.Errorf("view %q: expected view settings", name)
}

func (c *TimelineCatalog) loadViewGroupMapping(groupName string, groupNode *yaml.Node) error {
	for j := 0; j < len(groupNode.Content); j += 2 {
		subName := groupNode.Content[j].Value
		settings := groupNode.Content[j+1]

		if !isViewSettingsMapping(settings) {
			return fmt.Errorf(
				"view %s.%s: expected duration/refresh settings",
				groupName,
				subName,
			)
		}

		if err := c.applyViewSettings(groupName+"."+subName, settings); err != nil {
			return err
		}
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

	step := c.viewStepOrDefault(panel)
	if err := applyStepOverrides(name, &step, settings); err != nil {
		return err
	}

	c.views[panel] = step

	return nil
}

func (c *TimelineCatalog) viewStepOrDefault(panel display.Panel) TimelineStep {
	if step, ok := c.views[panel]; ok {
		return step
	}

	duration, refresh := defaultViewSettings(panel)

	return TimelineStep{Duration: duration, Refresh: refresh}
}

func applyStepOverrides(name string, step *TimelineStep, settings viewSettingsFile) error {
	if err := applyDurationOverride(name, step, settings.Duration); err != nil {
		return err
	}

	return applyRefreshOverride(name, step, settings.Refresh)
}

func applyDurationOverride(name string, step *TimelineStep, raw string) error {
	if raw == "" {
		return nil
	}

	duration, err := parseMinDuration(raw, time.Second, func() error {
		return fmt.Errorf("view %s: invalid duration %q", name, raw)
	})
	if err != nil {
		return err
	}

	step.Duration = duration

	return nil
}

func applyRefreshOverride(name string, step *TimelineStep, raw string) error {
	if raw == "" {
		return nil
	}

	refresh, err := parseMinDuration(raw, timelineRefreshFloor, func() error {
		return fmt.Errorf(
			"view %s: invalid refresh %q (minimum %s)",
			name,
			raw,
			timelineRefreshFloor,
		)
	})
	if err != nil {
		return err
	}

	step.Refresh = refresh

	return nil
}

func parseMinDuration(
	raw string,
	minDuration time.Duration,
	onInvalid func() error,
) (time.Duration, error) {
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed < minDuration {
		return 0, onInvalid()
	}

	return parsed, nil
}

func (c *TimelineCatalog) stepsForEntries(entries []viewEntry) ([]TimelineStep, error) {
	steps := make([]TimelineStep, 0, len(entries))

	for _, entry := range entries {
		panel, err := panelForViewName(entry.Name)
		if err != nil {
			return nil, err
		}

		settings, ok := c.views[panel]
		if !ok {
			duration, refresh := defaultViewSettings(panel)

			settings = TimelineStep{Duration: duration, Refresh: refresh}
		}

		if entry.Duration != nil {
			settings.Duration = *entry.Duration
		}

		if entry.Refresh != nil {
			settings.Refresh = *entry.Refresh
		}

		steps = append(steps, TimelineStep{
			Panel:    panel,
			Duration: settings.Duration,
			Refresh:  settings.Refresh,
		})
	}

	return steps, nil
}

// orderEntries resolves a single-line order expression (from the alarm
// board_timeline column or the API) into view entries. A single word that
// names a preset expands to that preset; otherwise the expression is a
// comma-separated list of view[:duration[:refresh]] references.
func (c *TimelineCatalog) orderEntries(order string) ([]viewEntry, error) {
	trimmed := strings.TrimSpace(order)

	switch {
	case trimmed == "":
		return c.order, nil
	case !strings.ContainsAny(trimmed, ",:"):
		if entries, ok := c.presets[trimmed]; ok {
			return append([]viewEntry(nil), entries...), nil
		}

		return []viewEntry{{Name: trimmed}}, nil
	default:
		return parseEntryList(trimmed)
	}
}

func parseEntryList(expr string) ([]viewEntry, error) {
	parts := strings.Split(expr, ",")
	entries := make([]viewEntry, 0, len(parts))

	for _, part := range parts {
		entry, err := parseViewExpression(part)
		if err != nil {
			return nil, err
		}

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("view list is empty")
	}

	return entries, nil
}

func (c *TimelineCatalog) introFromExpression(expr string) ([]viewEntry, error) {
	trimmed := strings.TrimSpace(expr)

	switch {
	case trimmed == "":
		return nil, nil
	case !strings.ContainsAny(trimmed, ",:"):
		if entries, ok := c.presets[trimmed]; ok {
			return append([]viewEntry(nil), entries...), nil
		}

		return []viewEntry{{Name: trimmed}}, nil
	default:
		return parseEntryList(trimmed)
	}
}

func (c *TimelineCatalog) build(orderEntries, introEntries []viewEntry) (*Timeline, error) {
	timeline := &Timeline{}

	introSteps, err := c.stepsForEntries(introEntries)
	if err != nil {
		return nil, fmt.Errorf("intro: %w", err)
	}

	timeline.Intro = introSteps

	loopSteps, err := c.stepsForEntries(orderEntries)
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
	entries, err := c.orderEntries(order)
	if err != nil {
		return nil, err
	}

	return c.build(entries, c.intro)
}

func (c *TimelineCatalog) ResolveForCast(order, intro string) (*Timeline, error) {
	entries, err := c.orderEntries(order)
	if err != nil {
		return nil, err
	}

	introEntries, err := c.introFromExpression(intro)
	if err != nil {
		return nil, fmt.Errorf("board intro %q: %w", intro, err)
	}

	return c.build(entries, introEntries)
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
		return resolveLoadedCatalog(catalog)
	}

	if strings.TrimSpace(legacySpec) != "" {
		return openLegacyTimeline(legacySpec)
	}

	catalog = DefaultTimelineCatalog()
	timeline, err = catalog.Resolve("")

	return timeline, catalog, err
}

func resolveLoadedCatalog(
	catalog *TimelineCatalog,
) (*Timeline, *TimelineCatalog, error) {
	timeline, err := catalog.Resolve("")
	if err != nil {
		return nil, nil, err
	}

	return timeline, catalog, nil
}

func openLegacyTimeline(
	legacySpec string,
) (*Timeline, *TimelineCatalog, error) {
	schedule, err := ParseBoardSchedule(legacySpec)
	if err != nil {
		return nil, nil, err
	}

	return NewTimelineFromLegacySchedule(schedule), DefaultTimelineCatalog(), nil
}

func DefaultTimelineCatalog() *TimelineCatalog {
	catalog := &TimelineCatalog{
		views:   defaultViews(),
		presets: map[string][]viewEntry{},
	}

	for name, entries := range builtinPresets {
		catalog.presets[name] = entriesForNames(entries)
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
