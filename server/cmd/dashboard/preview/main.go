// Command preview renders an entire board session of PNG images
// straight to the data directory, one per minute, without contacting
// any Chromecast device. It exists solely for development testing:
// in normal operation boards are rendered and cast by the API
// server.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/app"
	"github.com/johndennehy101/pi-alarm-clock/server/internal/basemap"
	"github.com/johndennehy101/pi-alarm-clock/server/internal/bus"
	"github.com/johndennehy101/pi-alarm-clock/server/internal/config"
	"github.com/johndennehy101/pi-alarm-clock/server/internal/display"
)

func main() {
	minutes := flag.Int("minutes", 60, "number of minutes to render")
	stop := flag.String(
		"stop",
		"",
		"bus stop number (defaults to BUS_STOP_NUMBER)",
	)
	order := flag.String(
		"order",
		"",
		"board timeline to render: preset name or comma-separated view list",
	)
	flag.Parse()

	if *minutes < 1 {
		log.Fatalf("-minutes must be at least 1")
	}

	if err := run(*minutes, *stop, *order); err != nil {
		log.Fatalf("board preview failed: %v", err)
	}
}

func run(minutes int, stop string, order string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	if stop == "" {
		stop = cfg.BusStopNumber
	}

	ctx := context.Background()

	timetable, err := loadTimetable(ctx, cfg.DataDir, stop)
	if err != nil {
		log.Printf("Bus timetable unavailable: %v", err)
	}

	timeline, catalog, err := app.OpenBoardTimeline(cfg.DataDir, cfg.BoardSchedule)
	if err != nil {
		log.Printf("Board timeline config invalid, using built-in default: %v", err)

		timeline, catalog = nil, nil
	}

	if order != "" {
		if catalog == nil {
			catalog = app.DefaultTimelineCatalog()
		}

		resolved, err := catalog.Resolve(order)
		if err != nil {
			return fmt.Errorf("resolving -order: %w", err)
		}

		timeline = resolved
	}

	if stop == "" {
		log.Printf("No bus stop configured; bus panels render empty")
	}

	boardsRoot := filepath.Join(cfg.DataDir, "boards")

	board := app.NewBoardService(
		boardsRoot,
		timetable,
		app.NewBusRealtimeService(realtimeClient(cfg)),
		timeline,
		catalog,
		filepath.Join(cfg.DataDir, "board-timeline.yml"),
	)

	var basemapProvider display.BasemapProvider
	if cfg.MapboxToken != "" {
		basemapProvider = basemap.NewMapboxProvider(
			cfg.MapboxToken,
			filepath.Join(cfg.DataDir, "basemap"),
		)
	} else {
		log.Printf("No MAPBOX_TOKEN configured; map panels render without a basemap")
	}

	renderer, err := display.NewBoardRenderer(
		app.NewWeatherService(cfg.WeatherLatitude, cfg.WeatherLongitude),
		basemapProvider,
	)
	if err != nil {
		return fmt.Errorf("creating board renderer: %w", err)
	}

	// The base URL is only used in the URLs that would be handed to a
	// Chromecast; nothing fetches it here.
	board.Configure(renderer, "http://preview.invalid/display/boards")

	if timeline == nil {
		timeline, _ = app.DefaultTimelineCatalog().Resolve("")
	}

	start := time.Now().Truncate(time.Minute)
	end := start.Add(time.Duration(minutes) * time.Minute)

	fmt.Printf("Rendering %d minutes to %s\n", minutes, boardsRoot)

	renderAt := start
	for renderAt.Before(end) {
		slot := timeline.At(start, renderAt)

		_, err := board.BoardURL(ctx, start, renderAt, stop)
		if err != nil {
			log.Printf("Rendering frame at %s failed: %v", renderAt.Format("15:04:05"), err)
		} else {
			fmt.Printf(
				"  frame-%d.png  %s  panel=%s  step=%d  tick=%d\n",
				renderAt.Unix(),
				renderAt.Format("15:04:05"),
				slot.Panel,
				slot.Step,
				slot.Tick,
			)
		}

		renderAt = timeline.NextBoundary(start, renderAt)
	}

	session := start.Format("20060102T1504")

	fmt.Printf("Session written to %s\n", filepath.Join(boardsRoot, session))
	fmt.Println("No Chromecast devices were contacted.")

	return nil
}

func loadTimetable(
	ctx context.Context,
	dataDir string,
	focusStopNumber string,
) (*bus.Timetable, error) {
	datasetPath, err := bus.NewDatasetManager(
		filepath.Join(dataDir, "bus"),
	).Ensure(ctx)
	if err != nil && datasetPath == "" {
		return nil, fmt.Errorf("ensuring bus dataset: %w", err)
	}

	if err != nil {
		log.Printf("Bus timetable refresh failed; using cached dataset: %v", err)
	}

	location, err := time.LoadLocation("Europe/Dublin")
	if err != nil {
		return nil, fmt.Errorf("loading Dublin timezone: %w", err)
	}

	timetable, err := bus.OpenTimetable(
		datasetPath,
		filepath.Join(dataDir, "bus", "timetable.cache"),
		location,
		focusStopNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("loading bus timetable: %w", err)
	}

	return timetable, nil
}

func realtimeClient(cfg *config.Config) *bus.RealtimeClient {
	if cfg.BusRealtimeAPIKey == "" {
		return nil
	}

	return bus.NewRealtimeClient(cfg.BusRealtimeURL, cfg.BusRealtimeAPIKey)
}
