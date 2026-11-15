package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/alarm"
	"github.com/johndennehy101/pi-alarm-clock/server/internal/hue"
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

func (s *HueService) RunWakeUpLighting(
	ctx context.Context,
	settings alarm.HueSettings,
) error {
	if !settings.Enabled {
		return nil
	}

	if len(settings.LightIDs) == 0 {
		return fmt.Errorf("no Hue lights selected")
	}

	if settings.StartBrightness < 0 || settings.StartBrightness > 100 {
		return fmt.Errorf("starting brightness must be between 0 and 100")
	}

	if settings.Brightness < 1 || settings.Brightness > 100 {
		return fmt.Errorf("target brightness must be between 1 and 100")
	}

	// Turn off the selected lights before starting.
	if err := s.SetLightsPower(ctx, settings.LightIDs, false); err != nil {
		return fmt.Errorf("turning off Hue lights: %w", err)
	}

	// Set the initial brightness.
	if err := s.SetLightsBrightness(
		ctx,
		settings.LightIDs,
		settings.StartBrightness,
	); err != nil {
		return fmt.Errorf("setting starting brightness: %w", err)
	}

	// If gradual brightness is disabled, set the target immediately.
	if !settings.GradualBrightness {
		if err := s.SetLightsBrightness(
			ctx,
			settings.LightIDs,
			settings.Brightness,
		); err != nil {
			return fmt.Errorf("setting target brightness: %w", err)
		}

		return nil
	}

	if settings.BrightnessDuration < 1 {
		return fmt.Errorf("brightness duration must be at least 1 minute")
	}

	const steps = 10

	duration := time.Duration(settings.BrightnessDuration) * time.Minute
	stepDelay := duration / steps

	for step := 1; step <= steps; step++ {
		timer := time.NewTimer(stepDelay)

		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()

		case <-timer.C:
		}

		brightness := settings.StartBrightness +
			(settings.Brightness-settings.StartBrightness)*step/steps

		if err := s.SetLightsBrightness(
			ctx,
			settings.LightIDs,
			brightness,
		); err != nil {
			return fmt.Errorf(
				"setting Hue brightness to %d%%: %w",
				brightness,
				err,
			)
		}
	}

	return nil
}

func (s *HueService) SetLightsPower(
	ctx context.Context,
	lightIDs []string,
	on bool,
) error {
	s.mu.RLock()
	client := s.client
	available := s.available
	s.mu.RUnlock()

	if !available || client == nil {
		return fmt.Errorf("Hue is not connected")
	}

	for _, lightID := range lightIDs {
		if err := client.SetLightPower(ctx, lightID, on); err != nil {
			return fmt.Errorf("setting power for Hue light %s: %w", lightID, err)
		}
	}

	return nil
}

func (s *HueService) SetLightsBrightness(
	ctx context.Context,
	lightIDs []string,
	brightness int,
) error {
	if brightness < 0 || brightness > 100 {
		return fmt.Errorf("brightness must be between 0 and 100")
	}

	s.mu.RLock()
	client := s.client
	available := s.available
	s.mu.RUnlock()

	if !available || client == nil {
		return fmt.Errorf("Hue is not connected")
	}

	for _, lightID := range lightIDs {
		if err := client.SetLightBrightness(ctx, lightID, brightness); err != nil {
			return fmt.Errorf(
				"setting brightness for Hue light %s: %w",
				lightID,
				err,
			)
		}
	}

	return nil
}
