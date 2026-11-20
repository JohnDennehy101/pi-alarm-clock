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
	flag.Parse()

	if *minutes < 1 {
		log.Fatalf("-minutes must be at least 1")
	}

	if err := run(*minutes, *stop); err != nil {
		log.Fatalf("board preview failed: %v", err)
	}
}

func run(minutes int, stop string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	if stop == "" {
		stop = cfg.BusStopNumber
	}

	schedule, err := app.ParseBoardSchedule(cfg.BoardSchedule)
	if err != nil {
		return fmt.Errorf("parsing board schedule: %w", err)
	}

	ctx := context.Background()

	timetable, err := loadTimetable(ctx, cfg.DataDir)
	if err != nil {
		log.Printf("Bus timetable unavailable: %v", err)
	}

	if stop == "" {
		log.Printf("No bus stop configured; bus panels render empty")
	}

	boardsRoot := filepath.Join(cfg.DataDir, "boards")

	board := app.NewBoardService(
		boardsRoot,
		timetable,
		app.NewBusRealtimeService(realtimeClient(cfg)),
		schedule,
	)

	renderer, err := display.NewBoardRenderer(
		app.NewWeatherService(cfg.WeatherLatitude, cfg.WeatherLongitude),
	)
	if err != nil {
		return fmt.Errorf("creating board renderer: %w", err)
	}

	// The base URL is only used in the URLs that would be handed to a
	// Chromecast; nothing fetches it here.
	board.Configure(renderer, "http://preview.invalid/display/boards")

	start := time.Now().Truncate(time.Minute)

	fmt.Printf("Rendering %d minutes to %s\n", minutes, boardsRoot)

	for minute := 0; minute < minutes; minute++ {
		renderTime := start.Add(time.Duration(minute) * time.Minute)

		_, err := board.BoardURL(
			ctx,
			start,
			renderTime,
			stop,
		)
		if err != nil {
			log.Printf("Rendering minute %d failed: %v", minute, err)
			continue
		}

		fmt.Printf(
			"  minute-%02d.png  %s  panel=%s\n",
			minute,
			renderTime.Format("15:04"),
			schedule.PanelAt(minute),
		)
	}

	session := start.Format("20060102T1504")

	fmt.Printf("Session written to %s\n", filepath.Join(boardsRoot, session))
	fmt.Println("No Chromecast devices were contacted.")

	return nil
}

func loadTimetable(ctx context.Context, dataDir string) (*bus.Timetable, error) {
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

	timetable, err := bus.NewTimetable(datasetPath, location)
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
