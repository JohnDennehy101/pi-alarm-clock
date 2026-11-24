package app

import (
	"context"
	"fmt"
	"log"
	"path/filepath"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/alarm"
	"github.com/johndennehy101/pi-alarm-clock/server/internal/audio"
	"github.com/johndennehy101/pi-alarm-clock/server/internal/bus"
	"github.com/johndennehy101/pi-alarm-clock/server/internal/display"
)

type Services struct {
	Sonos      *SonosService
	Hue        *HueService
	Chromecast *ChromecastService
	Weather    *WeatherService
	Board      *BoardService
	Audio      *audio.AudioManager
	Alarm      *AlarmService
	Executor   *AlarmExecutor
}

type DeviceConfig struct {
	SonosEnabled      bool
	HueAPIKey         string
	ChromecastEnabled bool
}

type LocationConfig struct {
	Latitude  float64
	Longitude float64
}

type BusBoardConfig struct {
	Timetable         *bus.Timetable
	StopNumber        string
	RealtimeURL       string
	RealtimeAPIKey    string
	BoardScheduleSpec string
	BasemapProvider   display.BasemapProvider
}

type ServicesConfig struct {
	DataDir         string
	MediaPort       int
	AlarmRepository alarm.Repository
	Devices         DeviceConfig
	Location        LocationConfig
	BusBoard        BusBoardConfig
}

func NewServices(
	ctx context.Context,
	cfg ServicesConfig,
) (*Services, error) {
	var realtimeClient *bus.RealtimeClient
	if cfg.BusBoard.RealtimeAPIKey != "" {
		realtimeClient = bus.NewRealtimeClient(
			cfg.BusBoard.RealtimeURL,
			cfg.BusBoard.RealtimeAPIKey,
		)
	}

	boardTimeline, boardCatalog, err := OpenBoardTimeline(
		cfg.DataDir,
		cfg.BusBoard.BoardScheduleSpec,
	)
	if err != nil {
		log.Printf("Board timeline config invalid, using built-in default: %v", err)

		boardTimeline = nil
		boardCatalog = nil
	}

	services := &Services{
		Sonos:      NewSonosService(filepath.Join(cfg.DataDir, "sonos.json")),
		Hue:        NewHueService(cfg.Devices.HueAPIKey),
		Chromecast: NewChromecastService(filepath.Join(cfg.DataDir, "chromecast.json")),
		Board: NewBoardService(
			filepath.Join(cfg.DataDir, "boards"),
			cfg.BusBoard.Timetable,
			NewBusRealtimeService(realtimeClient),
			BoardTimelineConfig{
				Timeline:     boardTimeline,
				Catalog:      boardCatalog,
				TimelinePath: filepath.Join(cfg.DataDir, "board-timeline.yml"),
			},
		),
		Weather: NewWeatherService(cfg.Location.Latitude, cfg.Location.Longitude),
		Alarm:   NewAlarmService(cfg.AlarmRepository),
	}

	services.Chromecast.setBusStopNumber(cfg.BusBoard.StopNumber)

	services.Executor = NewAlarmExecutor(
		services.Sonos,
		services.Hue,
		services.Chromecast,
	)

	services.initMedia(cfg.MediaPort, services.Weather, cfg.BusBoard.BasemapProvider)
	services.initBoardURL(cfg.MediaPort)
	services.initSonos(ctx, cfg.Devices.SonosEnabled)
	services.initHue(ctx, cfg.Devices.HueAPIKey)
	services.initChromecast(ctx, cfg.Devices.ChromecastEnabled)

	return services, nil
}

func (s *Services) initMedia(
	mediaPort int,
	weatherClient *WeatherService,
	basemapProvider display.BasemapProvider,
) {
	audioManager, err := audio.NewAudioManager(mediaPort)
	if err != nil {
		log.Printf("Audio manager unavailable at startup: %v", err)
	} else {
		s.Audio = audioManager
	}

	var forecastSource display.WeatherService
	if weatherClient != nil {
		forecastSource = weatherClient
	}

	boardRenderer, err := display.NewBoardRenderer(forecastSource, basemapProvider)
	if err != nil {
		log.Printf("Board renderer unavailable at startup: %v", err)
		return
	}

	s.Board.setRenderer(boardRenderer)
}

func (s *Services) initBoardURL(mediaPort int) {
	ip, err := audio.GetLocalIP()
	if err != nil {
		log.Printf(
			"Board URL unavailable at startup: %v",
			err,
		)
		return
	}

	s.Board.setBaseURL(fmt.Sprintf(
		"http://%s:%d/display/boards",
		ip,
		mediaPort,
	))

	s.Chromecast.setBoardProvider(s.Board)
}

func (s *Services) initSonos(ctx context.Context, enabled bool) {
	if !enabled {
		log.Println("Sonos integration disabled")
		return
	}

	if err := s.Sonos.Restore(ctx); err != nil {
		log.Printf("Sonos unavailable at startup: %v", err)
	}
}

func (s *Services) initHue(ctx context.Context, apiKey string) {
	if apiKey == "" {
		log.Println("Hue not configured: no API key provided")
		return
	}

	if err := s.Hue.Connect(ctx); err != nil {
		log.Printf("Hue unavailable at startup: %v", err)
	}
}

func (s *Services) initChromecast(ctx context.Context, enabled bool) {
	if !enabled {
		log.Println("Chromecast integration disabled")
		return
	}

	if err := s.Chromecast.Restore(ctx); err != nil {
		log.Printf("Chromecast unavailable at startup: %v", err)
	}
}
