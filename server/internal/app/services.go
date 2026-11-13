package app

import (
	"context"
	"log"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/audio"
)

type Services struct {
	Sonos *SonosService
	Hue   *HueService
	Audio *audio.AudioManager
}

func NewServices(
	ctx context.Context,
	audioPort int,
	hueAPIKey string,
	sonosEnabled bool,
) (*Services, error) {
	services := &Services{
		Sonos: NewSonosService(),
		Hue:   NewHueService(hueAPIKey),
	}

	services.initAudio(audioPort)
	services.initSonos(ctx, sonosEnabled)
	services.initHue(ctx, hueAPIKey)

	return services, nil
}

func (s *Services) initAudio(audioPort int) {
	audioManager, err := audio.NewAudioManager(audioPort)
	if err != nil {
		log.Printf("Audio manager unavailable at startup: %v", err)
		return
	}
	s.Audio = audioManager
}

func (s *Services) initSonos(ctx context.Context, enabled bool) {
	if !enabled {
		log.Println("Sonos integration disabled")
		return
	}

	if _, err := s.Sonos.Discover(ctx); err != nil {
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
