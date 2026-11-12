package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/internal/hue"
	"github.com/openhue/openhue-go"
)

type HueStatus struct {
	Available bool   `json:"available"`
	Connected bool   `json:"connected"`
	BridgeIP  string `json:"bridgeIP,omitempty"`
	Error     string `json:"error,omitempty"`
}

type HueService struct {
	mu sync.RWMutex

	apiKey string
	client *hue.Client

	available bool
	lastError string
}

func NewHueService(apiKey string) *HueService {
	return &HueService{
		apiKey: apiKey,
	}
}

func (s *HueService) Connect(ctx context.Context) error {
	client, err := hue.NewClient(ctx, s.apiKey)
	if err != nil {
		s.mu.Lock()
		s.available = false
		s.lastError = err.Error()
		s.mu.Unlock()

		return fmt.Errorf("connecting to Hue: %w", err)
	}

	s.mu.Lock()
	s.client = client
	s.available = true
	s.lastError = ""
	s.mu.Unlock()

	return nil
}

func (s *HueService) Status() HueStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := HueStatus{
		Available: s.available,
		Connected: s.client != nil,
		Error:     s.lastError,
	}

	if s.client != nil {
		status.BridgeIP = s.client.IP()
	}

	return status
}

func (s *HueService) SetBrightness(ctx context.Context, brightness int) error {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("no Hue bridge connected")
	}

	return client.SetBrightness(ctx, brightness)
}

func (s *HueService) GetLights(ctx context.Context) (map[string]openhue.LightGet, error) {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("no Hue bridge connected")
	}

	return client.GetLights(ctx)
}

func (s *HueService) TurnOn(ctx context.Context) error {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("no Hue bridge connected")
	}

	return client.TurnOn(ctx)
}

func (s *HueService) TurnOff(ctx context.Context) error {
	s.mu.RLock()
	client := s.client
	s.mu.RUnlock()

	if client == nil {
		return fmt.Errorf("no Hue bridge connected")
	}

	return client.TurnOff(ctx)
}

func (s *HueService) RunSunrise(ctx context.Context, totalDuration time.Duration) error {
	if totalDuration <= 0 {
		return fmt.Errorf("sunrise duration must be greater than zero")
	}

	stepDelay := totalDuration / 10
	if stepDelay <= 0 {
		return fmt.Errorf("sunrise duration is too short")
	}

	if err := s.TurnOff(ctx); err != nil {
		return fmt.Errorf("starting sunrise: %w", err)
	}

	for brightness := 10; brightness <= 100; brightness += 10 {
		timer := time.NewTimer(stepDelay)

		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()

		case <-timer.C:
		}

		if err := s.SetBrightness(ctx, brightness); err != nil {
			return fmt.Errorf(
				"setting sunrise brightness to %d%%: %w",
				brightness,
				err,
			)
		}
	}

	return nil
}
