package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/display"
	"gopkg.in/yaml.v3"
)

const (
	timelineMaxDuration = 60 * time.Minute
	timelineMaxOrder    = 20
)

var (
	ErrBoardTimelinePresetExists   = errors.New("board timeline preset already exists")
	ErrBoardTimelinePresetNotFound = errors.New("board timeline preset not found")
	ErrBoardTimelineBuiltinPreset  = errors.New("built-in board timeline preset cannot be deleted")
)

var presetNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)

type BoardViewSettings struct {
	DurationSeconds int64 `json:"durationSeconds"`
	RefreshSeconds  int64 `json:"refreshSeconds"`
}

type BoardPreset struct {
	Name    string   `json:"name"`
	Order   []string `json:"order"`
	BuiltIn bool     `json:"builtIn"`
}

type BoardTimelineSettings struct {
	Views          map[string]BoardViewSettings `json:"views"`
	Presets        []BoardPreset                `json:"presets"`
	Order          []string                     `json:"order"`
	Intro          []string                     `json:"intro"`
	AvailableViews []string                     `json:"availableViews"`
}

func availableViewNames() []string {
	return []string{"weather", "bus", "schedule", "bus.map", "bus.progress"}
}

func canonicalViewName(panel display.Panel) string {
	switch panel {
	case display.PanelBus:
		return "bus.schedule"
	case display.PanelBusMap:
		return "bus.map"
	case display.PanelBusProgress:
		return "bus.progress"
	default:
		return "weather"
	}
}

func normalizePresetName(name string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))

	if !presetNamePattern.MatchString(normalized) {
		return "", fmt.Errorf(
			"invalid preset name %q: expected lower-case letters, digits and hyphens, 1-32 characters starting with a letter or digit",
			name,
		)
	}

	return normalized, nil
}

func entriesFromStrings(values []string) ([]viewEntry, error) {
	entries := make([]viewEntry, 0, len(values))

	for _, value := range values {
		entry, err := parseViewExpression(value)
		if err != nil {
			return nil, err
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func normalizeViewEntries(entries []viewEntry) ([]viewEntry, error) {
	if len(entries) < 1 || len(entries) > timelineMaxOrder {
		return nil, fmt.Errorf("order must contain between 1 and %d views", timelineMaxOrder)
	}

	normalized := make([]viewEntry, 0, len(entries))

	for _, entry := range entries {
		canonical, err := entry.Canonical()
		if err != nil {
			return nil, fmt.Errorf(
				"%w (available: %s)",
				err,
				strings.Join(availableViewNames(), ", "),
			)
		}

		normalized = append(normalized, canonical)
	}

	return normalized, nil
}

func isBuiltinPreset(name string) bool {
	_, ok := builtinPresets[name]
	return ok
}

func equalEntries(a, b []viewEntry) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		ca, err := a[i].Canonical()
		if err != nil {
			return false
		}

		cb, err := b[i].Canonical()
		if err != nil {
			return false
		}

		if ca.String() != cb.String() {
			return false
		}
	}

	return true
}

func cloneEntries(entries []viewEntry) []viewEntry {
	if entries == nil {
		return nil
	}

	cloned := make([]viewEntry, len(entries))

	for i, entry := range entries {
		cloned[i] = viewEntry{Name: entry.Name, Duration: entry.Duration, Refresh: entry.Refresh}
	}

	return cloned
}

func (c *TimelineCatalog) clone() *TimelineCatalog {
	if c == nil {
		return DefaultTimelineCatalog()
	}

	cloned := &TimelineCatalog{
		views:   make(map[display.Panel]TimelineStep, len(c.views)),
		presets: make(map[string][]viewEntry, len(c.presets)),
		order:   cloneEntries(c.order),
		intro:   cloneEntries(c.intro),
	}

	for panel, step := range c.views {
		cloned.views[panel] = step
	}

	for name, entries := range c.presets {
		cloned.presets[name] = cloneEntries(entries)
	}

	return cloned
}

func durationString(d time.Duration) string {
	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", d/time.Minute)
	}

	return fmt.Sprintf("%ds", d/time.Second)
}

func (c *TimelineCatalog) setViewSettings(
	name string,
	duration *time.Duration,
	refresh *time.Duration,
) error {
	if duration == nil && refresh == nil {
		return fmt.Errorf("durationSeconds or refreshSeconds is required")
	}

	panel, step, err := c.lookupViewStep(name)
	if err != nil {
		return err
	}

	if err := applyViewDuration(&step, name, duration); err != nil {
		return err
	}

	if err := applyViewRefresh(&step, name, refresh); err != nil {
		return err
	}

	if step.Refresh > step.Duration {
		return fmt.Errorf(
			"view %s: refresh %s exceeds duration %s",
			name,
			step.Refresh,
			step.Duration,
		)
	}

	c.views[panel] = step

	return nil
}

func (c *TimelineCatalog) lookupViewStep(name string) (display.Panel, TimelineStep, error) {
	panel, err := panelForViewName(name)
	if err != nil {
		return 0, TimelineStep{}, fmt.Errorf(
			"%w (available: %s)",
			err,
			strings.Join(availableViewNames(), ", "),
		)
	}

	step, ok := c.views[panel]
	if !ok {
		defaultDuration, defaultRefresh := defaultViewSettings(panel)
		step = TimelineStep{Duration: defaultDuration, Refresh: defaultRefresh}
	}

	return panel, step, nil
}

func applyViewDuration(step *TimelineStep, name string, duration *time.Duration) error {
	if duration == nil {
		return nil
	}

	if *duration < time.Second || *duration > timelineMaxDuration {
		return fmt.Errorf(
			"view %s: duration must be between 1s and %s",
			name,
			timelineMaxDuration,
		)
	}

	step.Duration = *duration

	return nil
}

func applyViewRefresh(step *TimelineStep, name string, refresh *time.Duration) error {
	if refresh == nil {
		return nil
	}

	if *refresh < timelineRefreshFloor {
		return fmt.Errorf(
			"view %s: refresh must be at least %s",
			name,
			timelineRefreshFloor,
		)
	}

	step.Refresh = *refresh

	return nil
}

func (c *TimelineCatalog) setOrder(entries []viewEntry) error {
	normalized, err := normalizeViewEntries(entries)
	if err != nil {
		return err
	}

	c.order = normalized

	return nil
}

func (c *TimelineCatalog) setIntro(entries []viewEntry) error {
	if len(entries) > timelineMaxOrder {
		return fmt.Errorf("intro must contain at most %d views", timelineMaxOrder)
	}

	normalized := make([]viewEntry, 0, len(entries))

	for _, entry := range entries {
		canonical, err := entry.Canonical()
		if err != nil {
			return fmt.Errorf(
				"%w (available: %s)",
				err,
				strings.Join(availableViewNames(), ", "),
			)
		}

		normalized = append(normalized, canonical)
	}

	c.intro = normalized

	return nil
}

func (c *TimelineCatalog) orderFromExpression(expr string) ([]viewEntry, error) {
	trimmed := strings.TrimSpace(expr)
	if trimmed == "" {
		return nil, fmt.Errorf("order is required")
	}

	return c.orderEntries(trimmed)
}

func (c *TimelineCatalog) putPreset(name string, entries []viewEntry) error {
	normalizedName, err := normalizePresetName(name)
	if err != nil {
		return err
	}

	normalizedOrder, err := normalizeViewEntries(entries)
	if err != nil {
		return err
	}

	c.presets[normalizedName] = normalizedOrder

	return nil
}

func (c *TimelineCatalog) deletePreset(name string) error {
	normalizedName, err := normalizePresetName(name)
	if err != nil {
		return err
	}

	if _, ok := c.presets[normalizedName]; !ok {
		return ErrBoardTimelinePresetNotFound
	}

	if isBuiltinPreset(normalizedName) {
		return ErrBoardTimelineBuiltinPreset
	}

	delete(c.presets, normalizedName)

	return nil
}

func entryStrings(entries []viewEntry) []string {
	out := make([]string, 0, len(entries))

	for _, entry := range entries {
		rendered := entry.String()

		if canonical, err := entry.Canonical(); err == nil {
			rendered = canonical.String()
		}

		out = append(out, rendered)
	}

	return out
}

func (c *TimelineCatalog) settings() BoardTimelineSettings {
	settings := BoardTimelineSettings{
		Views:          map[string]BoardViewSettings{},
		Presets:        []BoardPreset{},
		Order:          entryStrings(c.order),
		Intro:          entryStrings(c.intro),
		AvailableViews: availableViewNames(),
	}

	for panel, step := range c.views {
		settings.Views[canonicalViewName(panel)] = BoardViewSettings{
			DurationSeconds: int64(step.Duration / time.Second),
			RefreshSeconds:  int64(step.Refresh / time.Second),
		}
	}

	names := make([]string, 0, len(c.presets))
	for name := range c.presets {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		settings.Presets = append(settings.Presets, BoardPreset{
			Name:    name,
			Order:   entryStrings(c.presets[name]),
			BuiltIn: isBuiltinPreset(name),
		})
	}

	return settings
}

func yamlScalar(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func viewSettingsNode(step TimelineStep) *yaml.Node {
	node := &yaml.Node{
		Kind:  yaml.MappingNode,
		Tag:   "!!map",
		Style: yaml.FlowStyle,
	}

	node.Content = append(
		node.Content,
		yamlScalar("duration"),
		yamlScalar(durationString(step.Duration)),
		yamlScalar("refresh"),
		yamlScalar(durationString(step.Refresh)),
	)

	return node
}

func entrySettingsNode(entry viewEntry) *yaml.Node {
	node := &yaml.Node{
		Kind:  yaml.MappingNode,
		Tag:   "!!map",
		Style: yaml.FlowStyle,
	}

	if entry.Duration != nil {
		node.Content = append(
			node.Content,
			yamlScalar("duration"),
			yamlScalar(durationString(*entry.Duration)),
		)
	}

	if entry.Refresh != nil {
		node.Content = append(
			node.Content,
			yamlScalar("refresh"),
			yamlScalar(durationString(*entry.Refresh)),
		)
	}

	return node
}

func viewPairNode(name string, value *yaml.Node) *yaml.Node {
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	node.Content = append(node.Content, yamlScalar(name), value)

	return node
}

func entryNode(entry viewEntry) (*yaml.Node, error) {
	canonical, err := entry.Canonical()
	if err != nil {
		return nil, err
	}

	prefix, leaf, grouped := strings.Cut(canonical.Name, ".")

	if !canonical.HasOverrides() {
		if grouped {
			return viewPairNode(prefix, yamlScalar(leaf)), nil
		}

		return yamlScalar(canonical.Name), nil
	}

	settings := entrySettingsNode(canonical)

	if grouped {
		inner := viewPairNode(leaf, settings)

		return viewPairNode(prefix, inner), nil
	}

	return viewPairNode(canonical.Name, settings), nil
}

func viewsNode(c *TimelineCatalog) *yaml.Node {
	views := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	groups := map[string]*yaml.Node{}

	for _, name := range sortedViewNames(c) {
		appendViewNode(views, groups, c, name)
	}

	return views
}

func sortedViewNames(c *TimelineCatalog) []string {
	names := make([]string, 0, len(c.views))
	for panel := range c.views {
		names = append(names, canonicalViewName(panel))
	}

	sort.Strings(names)

	return names
}

func appendViewNode(
	views *yaml.Node,
	groups map[string]*yaml.Node,
	c *TimelineCatalog,
	name string,
) {
	panel, err := panelForViewName(name)
	if err != nil {
		return
	}

	step, ok := c.views[panel]
	if !ok {
		return
	}

	defaultDuration, defaultRefresh := defaultViewSettings(panel)
	if step.Duration == defaultDuration && step.Refresh == defaultRefresh {
		return
	}

	prefix, leaf, grouped := strings.Cut(name, ".")
	if !grouped {
		views.Content = append(views.Content, yamlScalar(name), viewSettingsNode(step))
		return
	}

	group, exists := groups[prefix]
	if !exists {
		group = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		groups[prefix] = group
		views.Content = append(views.Content, yamlScalar(prefix), group)
	}

	group.Content = append(group.Content, yamlScalar(leaf), viewSettingsNode(step))
}

func entriesSeq(entries []viewEntry) (*yaml.Node, error) {
	seq := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}

	for _, entry := range entries {
		node, err := entryNode(entry)
		if err != nil {
			return nil, err
		}

		seq.Content = append(seq.Content, node)
	}

	return seq, nil
}

func serializeCatalog(c *TimelineCatalog) ([]byte, error) {
	if c == nil {
		c = DefaultTimelineCatalog()
	}

	root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}

	if views := viewsNode(c); len(views.Content) > 0 {
		root.Content = append(root.Content, yamlScalar("views"), views)
	}

	if err := appendEntriesSection(root, "intro", c.intro, true); err != nil {
		return nil, err
	}

	if err := appendEntriesSection(root, "order", c.order, false); err != nil {
		return nil, err
	}

	if err := appendPresetsSection(root, c); err != nil {
		return nil, err
	}

	return yaml.Marshal(root)
}

func appendEntriesSection(
	root *yaml.Node,
	key string,
	entries []viewEntry,
	skipIfEmpty bool,
) error {
	if skipIfEmpty && len(entries) == 0 {
		return nil
	}

	seq, err := entriesSeq(entries)
	if err != nil {
		return err
	}

	root.Content = append(root.Content, yamlScalar(key), seq)

	return nil
}

func appendPresetsSection(root *yaml.Node, c *TimelineCatalog) error {
	names := customPresetNames(c)
	if len(names) == 0 {
		return nil
	}

	presets := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, name := range names {
		seq, err := entriesSeq(c.presets[name])
		if err != nil {
			return err
		}

		presets.Content = append(presets.Content, yamlScalar(name), seq)
	}

	root.Content = append(root.Content, yamlScalar("presets"), presets)

	return nil
}

func customPresetNames(c *TimelineCatalog) []string {
	names := make([]string, 0, len(c.presets))
	for name := range c.presets {
		if isBuiltinPreset(name) && equalEntries(c.presets[name], entriesForNames(builtinPresets[name])) {
			continue
		}

		names = append(names, name)
	}

	sort.Strings(names)

	return names
}

func (b *BoardService) persistCatalog(catalog *TimelineCatalog) error {
	if b.timelinePath == "" {
		return fmt.Errorf("board timeline file path is not configured")
	}

	data, err := serializeCatalog(catalog)
	if err != nil {
		return err
	}

	dir := filepath.Dir(b.timelinePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating board timeline directory: %w", err)
	}

	tmpPath := b.timelinePath + ".tmp"

	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("writing board timeline: %w", err)
	}

	if err := os.Rename(tmpPath, b.timelinePath); err != nil {
		_ = os.Remove(tmpPath)

		return fmt.Errorf("replacing board timeline: %w", err)
	}

	return nil
}

func (b *BoardService) persistAndSwapLocked(catalog *TimelineCatalog) error {
	timeline, err := catalog.Resolve("")
	if err != nil {
		return err
	}

	if err := b.persistCatalog(catalog); err != nil {
		return err
	}

	b.catalog = catalog
	b.timeline = timeline

	return nil
}

func (b *BoardService) TimelineSettings() BoardTimelineSettings {
	b.mu.Lock()
	catalog := b.catalog
	b.mu.Unlock()

	if catalog == nil {
		catalog = DefaultTimelineCatalog()
	}

	return catalog.settings()
}

func (b *BoardService) ListPresets() []BoardPreset {
	return b.TimelineSettings().Presets
}

func (b *BoardService) GetPreset(name string) (BoardPreset, error) {
	normalizedName, err := normalizePresetName(name)
	if err != nil {
		return BoardPreset{}, err
	}

	b.mu.Lock()
	catalog := b.catalog
	b.mu.Unlock()

	if catalog == nil {
		catalog = DefaultTimelineCatalog()
	}

	order, ok := catalog.presets[normalizedName]
	if !ok {
		return BoardPreset{}, ErrBoardTimelinePresetNotFound
	}

	return BoardPreset{
		Name:    normalizedName,
		Order:   entryStrings(order),
		BuiltIn: isBuiltinPreset(normalizedName),
	}, nil
}

func (b *BoardService) CreatePreset(name string, order []string) error {
	return b.mutatePreset(name, order, func(exists bool) error {
		if exists {
			return ErrBoardTimelinePresetExists
		}

		return nil
	})
}

func (b *BoardService) UpdatePreset(name string, order []string) error {
	return b.mutatePreset(name, order, func(exists bool) error {
		if !exists {
			return ErrBoardTimelinePresetNotFound
		}

		return nil
	})
}

func (b *BoardService) mutatePreset(
	name string,
	order []string,
	checkExistence func(exists bool) error,
) error {
	normalizedName, err := normalizePresetName(name)
	if err != nil {
		return err
	}

	entries, err := entriesFromStrings(order)
	if err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	catalog := b.currentCatalogLocked()
	_, exists := catalog.presets[normalizedName]
	if err := checkExistence(exists); err != nil {
		return err
	}

	cloned := catalog.clone()
	if err := cloned.putPreset(normalizedName, entries); err != nil {
		return err
	}

	return b.persistAndSwapLocked(cloned)
}

func (b *BoardService) currentCatalogLocked() *TimelineCatalog {
	if b.catalog == nil {
		return DefaultTimelineCatalog()
	}

	return b.catalog
}

func (b *BoardService) DeletePreset(name string) error {
	normalizedName, err := normalizePresetName(name)
	if err != nil {
		return err
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	catalog := b.catalog
	if catalog == nil {
		catalog = DefaultTimelineCatalog()
	}

	cloned := catalog.clone()
	if err := cloned.deletePreset(normalizedName); err != nil {
		return err
	}

	return b.persistAndSwapLocked(cloned)
}

func (b *BoardService) UpdateViewSettings(
	name string,
	durationSeconds *int64,
	refreshSeconds *int64,
) error {
	var duration, refresh *time.Duration

	if durationSeconds != nil {
		d := time.Duration(*durationSeconds) * time.Second
		duration = &d
	}

	if refreshSeconds != nil {
		r := time.Duration(*refreshSeconds) * time.Second
		refresh = &r
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	catalog := b.catalog
	if catalog == nil {
		catalog = DefaultTimelineCatalog()
	}

	cloned := catalog.clone()
	if err := cloned.setViewSettings(name, duration, refresh); err != nil {
		return err
	}

	return b.persistAndSwapLocked(cloned)
}

func (b *BoardService) SetTimelineOrder(expr string) error {
	return b.updateCatalogLocked(func(cloned *TimelineCatalog) error {
		entries, err := cloned.orderFromExpression(expr)
		if err != nil {
			return err
		}

		return cloned.setOrder(entries)
	})
}

func (b *BoardService) SetTimelineOrderViews(views []string) error {
	return b.updateCatalogLocked(func(cloned *TimelineCatalog) error {
		entries, err := entriesFromStrings(views)
		if err != nil {
			return err
		}

		return cloned.setOrder(entries)
	})
}

func (b *BoardService) SetTimelineIntro(views []string) error {
	return b.updateCatalogLocked(func(cloned *TimelineCatalog) error {
		entries, err := entriesFromStrings(views)
		if err != nil {
			return err
		}

		return cloned.setIntro(entries)
	})
}

func (b *BoardService) updateCatalogLocked(apply func(cloned *TimelineCatalog) error) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	cloned := b.currentCatalogLocked().clone()
	if err := apply(cloned); err != nil {
		return err
	}

	return b.persistAndSwapLocked(cloned)
}
