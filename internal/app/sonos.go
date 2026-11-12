package app

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/johndennehy101/pi-alarm-clock/internal/sonos"
)

type SonosService struct {
	mu sync.Mutex

	speakers        []sonos.Speaker
	client          *sonos.Client
	selectedSpeaker string
	available       bool
	lastError       string
}

type SonosStatus struct {
	Available       bool   `json:"available"`
	Connected       bool   `json:"connected"`
	SelectedSpeaker string `json:"selectedSpeaker,omitempty"`
	Error           string `json:"error,omitempty"`
}

func NewSonosService() *SonosService {
	return &SonosService{}
}

func (s *SonosService) withClient(fn func(c *sonos.Client) error) error {
	s.mu.Lock()
	client := s.client
	s.mu.Unlock()

	if client == nil {
		return fmt.Errorf("no Sonos speaker connected")
	}

	return fn(client)
}

func (s *SonosService) Discover(
	ctx context.Context,
) ([]sonos.Speaker, error) {
	speakers, err := sonos.DiscoverSpeakers(ctx)

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil {
		s.speakers = nil
		s.client = nil
		s.selectedSpeaker = ""
		s.available = false
		s.lastError = err.Error()

		return nil, fmt.Errorf("discovering speakers: %w", err)
	}

	s.speakers = speakers
	s.available = true
	s.lastError = ""

	return append([]sonos.Speaker(nil), speakers...), nil
}

func (s *SonosService) Speakers() []sonos.Speaker {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]sonos.Speaker(nil), s.speakers...)
}

func (s *SonosService) Connect(
	ctx context.Context,
	speakerName string,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.speakers) == 0 {
		return fmt.Errorf("no speakers discovered; discover speakers first")
	}

	var selected *sonos.Speaker

	for i := range s.speakers {
		if speakerName == "" ||
			strings.EqualFold(s.speakers[i].Name, speakerName) {
			selected = &s.speakers[i]
			break
		}
	}

	if selected == nil {
		return fmt.Errorf("Sonos speaker %q not found", speakerName)
	}

	client, err := sonos.NewClient(ctx, selected.IP)
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", selected.Name, err)
	}

	s.client = client
	s.selectedSpeaker = selected.Name

	return nil
}

func (s *SonosService) SelectedSpeaker() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.selectedSpeaker
}

func (s *SonosService) PlayURI(
	ctx context.Context,
	uri string,
	metadata string,
) error {
	return s.withClient(func(c *sonos.Client) error {
		return c.PlayURI(ctx, uri, metadata)
	})
}

func (s *SonosService) PlayRadioStation(
	ctx context.Context,
	name string,
) error {
	return s.withClient(func(c *sonos.Client) error {
		return c.PlayRadioStation(ctx, name)
	})
}

func (s *SonosService) GetRadioStations(
	ctx context.Context,
) ([]sonos.RadioStation, error) {
	var stations []sonos.RadioStation
	err := s.withClient(func(c *sonos.Client) error {
		var err error
		stations, err = c.GetRadioStations(ctx)
		return err
	})
	return stations, err
}

func (s *SonosService) SetVolume(
	ctx context.Context,
	volume int,
) error {
	return s.withClient(func(c *sonos.Client) error {
		return c.SetVolume(ctx, volume)
	})
}

func (s *SonosService) Stop(ctx context.Context) error {
	return s.withClient(func(c *sonos.Client) error {
		return c.Stop(ctx)
	})
}

func (s *SonosService) Status() SonosStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	return SonosStatus{
		Available:       s.available,
		Connected:       s.client != nil,
		SelectedSpeaker: s.selectedSpeaker,
		Error:           s.lastError,
	}
}
