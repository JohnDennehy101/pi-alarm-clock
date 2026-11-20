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
	boardDeparturesShown = 5
	boardScheduleDefault = "weather:5,bus:15"
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

	for _, entry := range strings.Split(spec, ",") {
		name, minutesValue, found := strings.Cut(entry, ":")
		if !found {
			return nil, fmt.Errorf("invalid board schedule entry %q", entry)
		}

		panel, err := display.ParsePanel(name)
		if err != nil {
			return nil, err
		}

		minutes, err := strconv.Atoi(strings.TrimSpace(minutesValue))
		if err != nil || minutes < 1 {
			return nil, fmt.Errorf("invalid board schedule duration %q", minutesValue)
		}

		schedule = append(schedule, BoardSegment{Panel: panel, Minutes: minutes})
	}

	return schedule, nil
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

	timetable *bus.Timetable
	realtime  *BusRealtimeService
	schedule  BoardSchedule
}

type BoardProvider interface {
	BoardURL(
		ctx context.Context,
		sessionStart time.Time,
		renderTime time.Time,
		stopNumber string,
	) (string, error)
}

func NewBoardService(
	boardsRoot string,
	timetable *bus.Timetable,
	realtime *BusRealtimeService,
	schedule BoardSchedule,
) *BoardService {
	return &BoardService{
		boardsRoot: boardsRoot,
		timetable:  timetable,
		realtime:   realtime,
		schedule:   schedule,
	}
}

func (b *BoardService) setRenderer(renderer *display.BoardRenderer) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.renderer = renderer
}

// Configure installs a renderer and base URL so BoardURL can render
// boards outside of the HTTP service lifecycle, e.g. the
// cmd/dashboard/preview tool.
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
	b.mu.Lock()
	renderer := b.renderer
	baseURL := b.baseURL
	boardsRoot := b.boardsRoot
	b.mu.Unlock()

	if renderer == nil {
		return "", fmt.Errorf("board renderer is not configured")
	}

	if baseURL == "" {
		return "", fmt.Errorf("board URL is not configured")
	}

	renderTime = renderTime.Truncate(time.Minute)
	sessionStart = sessionStart.Truncate(time.Minute)

	if err := ctx.Err(); err != nil {
		return "", err
	}

	minute := int(renderTime.Sub(sessionStart) / time.Minute)
	if minute < 0 {
		return "", fmt.Errorf("render time is before board session start")
	}

	panel := b.schedule.PanelAt(minute)

	var departures []display.BusDeparture

	if panel == display.PanelBus {
		var err error

		departures, err = b.loadBusDepartures(ctx, stopNumber, renderTime)
		if err != nil {
			return "", fmt.Errorf("loading bus departures: %w", err)
		}
	}

	image, err := renderer.RenderBoard(ctx, renderTime, panel, departures)
	if err != nil {
		return "", fmt.Errorf("rendering board: %w", err)
	}

	session := sessionStart.Format("20060102T1504")
	sessionDir := filepath.Join(boardsRoot, session)

	if err := saveBoardImage(sessionDir, minute, image); err != nil {
		return "", err
	}

	pruneOldSessions(boardsRoot)

	name := fmt.Sprintf("minute-%02d.png", minute)

	return fmt.Sprintf("%s/%s/%s", baseURL, session, name), nil
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
