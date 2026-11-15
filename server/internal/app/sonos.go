package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/sonos"
)

type SonosService struct {
	mu sync.Mutex

	speakers          []sonos.Speaker
	client            *sonos.Client
	selectedSpeakerID string
	available         bool
	lastError         string
}

type SonosStatus struct {
	Available         bool   `json:"available"`
	Connected         bool   `json:"connected"`
	SelectedSpeakerID string `json:"selectedSpeakerId,omitempty"`
	Error             string `json:"error,omitempty"`
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
	if err != nil {
		s.mu.Lock()
		s.lastError = err.Error()
		s.mu.Unlock()

		return nil, fmt.Errorf("discovering speakers: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

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
	speakerID string,
) error {
	s.mu.Lock()

	var selected *sonos.Speaker

	for i := range s.speakers {
		if s.speakers[i].ID == speakerID {
			speaker := s.speakers[i]
			selected = &speaker
			break
		}
	}

	s.mu.Unlock()

	if selected == nil {
		return fmt.Errorf("Sonos speaker %q not found", speakerID)
	}

	client, err := sonos.NewClient(ctx, selected.IP)
	if err != nil {
		return fmt.Errorf("connecting to %s: %w", selected.Name, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.client = client
	s.selectedSpeakerID = selected.ID
	s.lastError = ""

	return nil
}

func (s *SonosService) SelectedSpeaker() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.selectedSpeakerID
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
		Available:         s.available,
		Connected:         s.client != nil,
		SelectedSpeakerID: s.selectedSpeakerID,
		Error:             s.lastError,
	}
}
