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

func NewServices(
	ctx context.Context,
	mediaPort int,
	hueAPIKey string,
	sonosEnabled bool,
	chromecastEnabled bool,
	weatherLatitude float64,
	weatherLongitude float64,
	dataDir string,
	alarmRepository alarm.Repository,
	timetable *bus.Timetable,
	busStopNumber string,
	busRealtimeURL string,
	busRealtimeAPIKey string,
	boardScheduleSpec string,
) (*Services, error) {
	boardSchedule, err := ParseBoardSchedule(boardScheduleSpec)
	if err != nil {
		return nil, fmt.Errorf("parsing board schedule: %w", err)
	}

	var realtimeClient *bus.RealtimeClient
	if busRealtimeAPIKey != "" {
		realtimeClient = bus.NewRealtimeClient(busRealtimeURL, busRealtimeAPIKey)
	}

	services := &Services{
		Sonos:      NewSonosService(filepath.Join(dataDir, "sonos.json")),
		Hue:        NewHueService(hueAPIKey),
		Chromecast: NewChromecastService(filepath.Join(dataDir, "chromecast.json")),
		Board: NewBoardService(
			filepath.Join(dataDir, "boards"),
			timetable,
			NewBusRealtimeService(realtimeClient),
			boardSchedule,
		),
		Weather: NewWeatherService(weatherLatitude, weatherLongitude),
		Alarm:   NewAlarmService(alarmRepository),
	}

	services.Chromecast.setBusStopNumber(busStopNumber)

	services.Executor = NewAlarmExecutor(
		services.Sonos,
		services.Hue,
		services.Chromecast,
	)

	services.initMedia(mediaPort, services.Weather)
	services.initBoardURL(mediaPort)
	services.initSonos(ctx, sonosEnabled)
	services.initHue(ctx, hueAPIKey)
	services.initChromecast(ctx, chromecastEnabled)

	return services, nil
}

func (s *Services) initMedia(mediaPort int, weatherClient *WeatherService) {
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

	boardRenderer, err := display.NewBoardRenderer(forecastSource)
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
