package app

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/bus"
	"github.com/johndennehy101/pi-alarm-clock/server/internal/display"
)

const (
	boardDeparturesShown       = 5
	boardScheduleDefault       = "weather:5,bus:15"
	boardMapWindowMinutes      = 15
	boardProgressWindowMinutes = 25
)

type BoardSegment struct {
	Panel   display.Panel
	Minutes int
}

type BoardSchedule []BoardSegment

func ParseBoardSchedule(spec string) (BoardSchedule, error) {
	if strings.TrimSpace(spec) == "" {
		spec = boardScheduleDefault
	}

	var schedule BoardSchedule

	for _, entry := range splitScheduleEntries(spec) {
		segments, err := parseScheduleEntry(entry)
		if err != nil {
			return nil, err
		}

		schedule = append(schedule, segments...)
	}

	return schedule, nil
}

func splitScheduleEntries(spec string) []string {
	var entries []string

	depth := 0
	start := 0

	for i, r := range spec {
		switch r {
		case '{':
			depth++
		case '}':
			depth--
		case ',':
			if depth == 0 {
				entries = append(entries, spec[start:i])
				start = i + 1
			}
		}
	}

	if depth != 0 {
		return []string{spec}
	}

	return append(entries, spec[start:])
}

func parseScheduleEntry(entry string) ([]BoardSegment, error) {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return nil, fmt.Errorf("invalid board schedule entry %q", entry)
	}

	if strings.Contains(entry, "{") {
		return parseBusGroupEntry(entry)
	}

	name, minutesValue, found := strings.Cut(entry, ":")
	if !found {
		return nil, fmt.Errorf("invalid board schedule entry %q", entry)
	}

	panel, err := display.ParsePanel(name)
	if err != nil {
		return nil, err
	}

	minutes, err := parsePositiveMinutes(minutesValue)
	if err != nil {
		return nil, err
	}

	return []BoardSegment{{Panel: panel, Minutes: minutes}}, nil
}

func parseBusGroupEntry(entry string) ([]BoardSegment, error) {
	open := strings.Index(entry, "{")
	if open < 0 || !strings.HasSuffix(entry, "}") {
		return nil, fmt.Errorf("invalid board schedule entry %q", entry)
	}

	if err := parseBusGroupHeader(entry[:open], entry); err != nil {
		return nil, err
	}

	views := strings.TrimSuffix(entry[open+1:], "}")
	var segments []BoardSegment

	for _, view := range splitScheduleEntries(views) {
		seg, err := parseBusGroupView(entry, view)
		if err != nil {
			return nil, err
		}

		segments = append(segments, seg)
	}

	if len(segments) == 0 {
		return nil, fmt.Errorf("empty board view group %q", entry)
	}

	return segments, nil
}

func parseBusGroupHeader(prefix, fullEntry string) error {
	name, remainder, found := strings.Cut(prefix, ":")
	if !found || strings.TrimSpace(remainder) != "" {
		return fmt.Errorf("invalid board schedule entry %q", fullEntry)
	}

	name = strings.TrimSpace(name)
	if name != "bus" {
		return fmt.Errorf("board view groups are only supported for bus, not %q", name)
	}

	return nil
}

func parseBusGroupView(entry, view string) (BoardSegment, error) {
	viewName, minutesValue, found := strings.Cut(view, ":")
	if !found {
		return BoardSegment{}, fmt.Errorf("invalid board schedule entry %q", entry)
	}

	panel, err := ParseBusView(viewName)
	if err != nil {
		return BoardSegment{}, err
	}

	minutes, err := parsePositiveMinutes(minutesValue)
	if err != nil {
		return BoardSegment{}, err
	}

	return BoardSegment{
		Panel:   panel,
		Minutes: minutes,
	}, nil
}

func parsePositiveMinutes(raw string) (int, error) {
	minutes, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || minutes < 1 {
		return 0, fmt.Errorf("invalid board schedule duration %q", raw)
	}

	return minutes, nil
}

func ParseBusView(name string) (display.Panel, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "schedule", "timetable":
		return display.PanelBus, nil
	case "map", "position":
		return display.PanelBusMap, nil
	case "progress", "track":
		return display.PanelBusProgress, nil
	default:
		return display.PanelWeather, fmt.Errorf("unknown bus board view %q", name)
	}
}

func (s BoardSchedule) PanelAt(minute int) display.Panel {
	if len(s) == 0 {
		return display.PanelWeather
	}

	total := 0
	for _, segment := range s {
		total += segment.Minutes
	}

	if total > 0 {
		minute %= total
	}

	for _, segment := range s {
		if minute < segment.Minutes {
			return segment.Panel
		}

		minute -= segment.Minutes
	}

	return s[len(s)-1].Panel
}

type BoardService struct {
	mu sync.Mutex

	renderer   *display.BoardRenderer
	boardsRoot string
	baseURL    string

	timetable    *bus.Timetable
	realtime     *BusRealtimeService
	timeline     *Timeline
	catalog      *TimelineCatalog
	timelinePath string
}

type BoardProvider interface {
	ResolveTimeline(order string) (*Timeline, error)
	ResolveTimelineForCast(order, intro string) (*Timeline, error)
	BoardURLForTimeline(context.Context, BoardRenderRequest, string) (string, error)
}

type BoardTimelineConfig struct {
	Timeline     *Timeline
	Catalog      *TimelineCatalog
	TimelinePath string
}

func NewBoardService(
	boardsRoot string,
	timetable *bus.Timetable,
	realtime *BusRealtimeService,
	cfg BoardTimelineConfig,
) *BoardService {
	timeline := cfg.Timeline
	if timeline == nil {
		timeline, _ = DefaultTimelineCatalog().Resolve("")
	}

	return &BoardService{
		boardsRoot:   boardsRoot,
		timetable:    timetable,
		realtime:     realtime,
		timeline:     timeline,
		catalog:      cfg.Catalog,
		timelinePath: cfg.TimelinePath,
	}
}

func (b *BoardService) ResolveTimeline(order string) (*Timeline, error) {
	b.mu.Lock()
	catalog := b.catalog
	timeline := b.timeline
	b.mu.Unlock()

	if strings.TrimSpace(order) == "" || catalog == nil {
		return timeline, nil
	}

	return catalog.Resolve(order)
}

func (b *BoardService) ResolveTimelineForCast(order, intro string) (*Timeline, error) {
	b.mu.Lock()
	catalog := b.catalog
	timeline := b.timeline
	b.mu.Unlock()

	order = strings.TrimSpace(order)
	intro = strings.TrimSpace(intro)

	if catalog == nil {
		if order == "" && intro == "" {
			return timeline, nil
		}

		catalog = DefaultTimelineCatalog()
	}

	return catalog.ResolveForCast(order, intro)
}

func (b *BoardService) setRenderer(renderer *display.BoardRenderer) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.renderer = renderer
}

func (b *BoardService) Configure(renderer *display.BoardRenderer, baseURL string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.renderer = renderer
	b.baseURL = baseURL
}

func (b *BoardService) setBaseURL(url string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.baseURL = url
}

func (b *BoardService) Handler() http.Handler {
	b.mu.Lock()
	renderer := b.renderer
	boardsRoot := b.boardsRoot
	b.mu.Unlock()

	return display.Handler(renderer, boardsRoot)
}

func (b *BoardService) BoardURL(
	ctx context.Context,
	sessionStart time.Time,
	renderTime time.Time,
	stopNumber string,
) (string, error) {
	return b.BoardURLForTimeline(ctx, BoardRenderRequest{
		SessionStart: sessionStart,
		RenderTime:   renderTime,
		Timeline:     nil,
	}, stopNumber)
}

type boardRenderState struct {
	renderer   *display.BoardRenderer
	baseURL    string
	boardsRoot string
	timeline   *Timeline
}

type BoardRenderRequest struct {
	SessionStart time.Time
	RenderTime   time.Time
	Timeline     *Timeline
}

func (b *BoardService) BoardURLForTimeline(
	ctx context.Context,
	req BoardRenderRequest,
	stopNumber string,
) (string, error) {
	state, err := b.loadRenderState(req.Timeline)
	if err != nil {
		return "", err
	}

	renderTime := req.RenderTime.Truncate(time.Second)
	sessionStart := req.SessionStart.Truncate(time.Minute)

	if err := ctx.Err(); err != nil {
		return "", err
	}

	if renderTime.Before(sessionStart) {
		return "", fmt.Errorf("render time is before board session start")
	}

	slot := state.timeline.At(sessionStart, renderTime)

	view, err := b.buildBoardView(ctx, stopNumber, renderTime, slot)
	if err != nil {
		return "", err
	}

	image, err := state.renderer.RenderBoard(ctx, renderTime, view)
	if err != nil {
		return "", fmt.Errorf("rendering board: %w", err)
	}

	session := sessionStart.Format("20060102T1504")
	sessionDir := filepath.Join(state.boardsRoot, session)
	name := fmt.Sprintf("frame-%d.png", renderTime.Unix())

	if err := saveBoardImage(sessionDir, name, image); err != nil {
		return "", err
	}

	pruneOldSessions(state.boardsRoot)

	return fmt.Sprintf("%s/%s/%s", state.baseURL, session, name), nil
}

func (b *BoardService) loadRenderState(timeline *Timeline) (boardRenderState, error) {
	b.mu.Lock()
	state := boardRenderState{
		renderer:   b.renderer,
		baseURL:    b.baseURL,
		boardsRoot: b.boardsRoot,
		timeline:   timeline,
	}
	if state.timeline == nil {
		state.timeline = b.timeline
	}
	b.mu.Unlock()

	if state.renderer == nil {
		return boardRenderState{}, fmt.Errorf("board renderer is not configured")
	}

	if state.baseURL == "" {
		return boardRenderState{}, fmt.Errorf("board URL is not configured")
	}

	return state, nil
}

func (b *BoardService) buildBoardView(
	ctx context.Context,
	stopNumber string,
	renderTime time.Time,
	slot BoardSlot,
) (display.BoardView, error) {
	view := display.BoardView{Panel: slot.Panel}

	switch slot.Panel {
	case display.PanelBus:
		departures, err := b.loadBusDepartures(ctx, stopNumber, renderTime)
		if err != nil {
			return display.BoardView{}, fmt.Errorf("loading bus departures: %w", err)
		}
		view.Departures = departures
	case display.PanelBusMap:
		mapBoard, err := b.loadBusMap(stopNumber, renderTime)
		if err != nil {
			return display.BoardView{}, fmt.Errorf("loading bus map: %w", err)
		}
		view.Map = mapBoard
	case display.PanelBusProgress:
		progress, err := b.loadBusProgress(stopNumber, renderTime, slot.Tick)
		if err != nil {
			return display.BoardView{}, fmt.Errorf("loading bus progress: %w", err)
		}
		view.Progress = progress
	}

	return view, nil
}

func (b *BoardService) approachingTrips(
	stopNumber string,
	renderTime time.Time,
	windowMinutes int,
) ([]bus.MapTrip, error) {
	if b.timetable == nil || stopNumber == "" {
		return nil, fmt.Errorf("bus timetable is not configured")
	}

	return b.timetable.MapTrips(
		stopNumber,
		renderTime,
		windowMinutes,
		b.realtime.Updates(),
	)
}

func (b *BoardService) loadBusProgress(
	stopNumber string,
	renderTime time.Time,
	tick int,
) (*display.BusProgress, error) {
	trips, err := b.approachingTrips(
		stopNumber,
		renderTime,
		boardProgressWindowMinutes,
	)
	if err != nil {
		return nil, err
	}

	if len(trips) == 0 {
		return &display.BusProgress{StopNumber: stopNumber}, nil
	}

	trip := trips[tick%len(trips)]

	return &display.BusProgress{
		Route:            trip.Route,
		Destination:      trip.Destination,
		StopNumber:       stopNumber,
		ProgressFraction: trip.ProgressFraction,
		PreviousFraction: trip.PreviousFraction,
		StopFractions:    trip.StopFractions,
		StopsRemaining:   trip.StopsRemaining,
		ExpectedArrival:  trip.Arrival,
		ScheduledArrival: trip.ScheduledArrival,
		IsRealtime:       trip.IsRealtime,
	}, nil
}

func (b *BoardService) loadBusMap(
	stopNumber string,
	renderTime time.Time,
) (*display.MapBoard, error) {
	if b.timetable == nil || stopNumber == "" {
		return nil, fmt.Errorf("bus timetable is not configured")
	}

	position, err := b.timetable.StopPosition(stopNumber)
	if err != nil {
		return nil, err
	}

	trips, err := b.approachingTrips(
		stopNumber,
		renderTime,
		boardMapWindowMinutes,
	)
	if err != nil {
		return nil, err
	}

	mapBoard := &display.MapBoard{
		StopNumber: stopNumber,
		StopPosition: display.MapPoint{
			Lat: position.Lat,
			Lon: position.Lon,
		},
		Trips: make([]display.MapTrip, 0, len(trips)),
	}

	for _, trip := range trips {
		mapBoard.Trips = append(mapBoard.Trips, display.MapTrip{
			Route:            trip.Route,
			Destination:      trip.Destination,
			ShapeID:          trip.ShapeID,
			Position:         display.MapPoint{Lat: trip.Lat, Lon: trip.Lon},
			Previous:         display.MapPoint{Lat: trip.PrevLat, Lon: trip.PrevLon},
			MinutesToStop:    trip.MinutesToStop,
			IsRealtime:       trip.IsRealtime,
			Arrival:          trip.Arrival,
			ScheduledArrival: trip.ScheduledArrival,
			Polyline:         mapCoordinates(trip.Polyline),
			Stops:            mapCoordinates(trip.Stops),
		})
	}

	return mapBoard, nil
}

func mapCoordinates(coords []bus.MapCoordinate) []display.MapPoint {
	points := make([]display.MapPoint, len(coords))
	for i, coord := range coords {
		points[i] = display.MapPoint{Lat: coord.Lat, Lon: coord.Lon}
	}

	return points
}

func (b *BoardService) loadBusDepartures(
	ctx context.Context,
	stopNumber string,
	now time.Time,
) ([]display.BusDeparture, error) {
	if b.timetable == nil || stopNumber == "" {
		return nil, nil
	}

	if b.realtime != nil {
		if err := b.realtime.Refresh(ctx, stopNumber); err != nil {
			log.Printf("Bus realtime refresh for stop %s failed: %v", stopNumber, err)
		}
	}

	scheduled, err := b.timetable.GetDepartures(stopNumber, now, boardDeparturesShown)
	if err != nil {
		return nil, err
	}

	enriched := b.realtime.Enrich(scheduled)

	departures := make([]display.BusDeparture, 0, len(enriched))
	for _, departure := range enriched {
		departures = append(departures, display.BusDeparture{
			Route:         departure.Route,
			Destination:   departure.Destination,
			ScheduledTime: departure.ScheduledTime,
			ExpectedTime:  departure.ExpectedTime,
			MinutesAway:   minutesAway(departure.ExpectedTime, now),
			IsRealtime:    departure.IsRealtime,
		})
	}

	return departures, nil
}

func minutesAway(at time.Time, now time.Time) int {
	remaining := at.Sub(now)
	if remaining <= 0 {
		return 0
	}

	return int(math.Ceil(remaining.Minutes()))
}
