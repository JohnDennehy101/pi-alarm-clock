package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/alarm"
	"github.com/johndennehy101/pi-alarm-clock/server/internal/sonos"
)

const sonosDiscoveryAttempts = 2

type SonosService struct {
	mu sync.Mutex

	speakers          []sonos.Speaker
	client            *sonos.Client
	selectedSpeakerID string
	available         bool
	lastError         string
	persistPath       string
}

type persistedSpeakerSelection struct {
	SpeakerID string `json:"speakerId"`
	Name      string `json:"name,omitempty"`
	IP        string `json:"ip"`
}

type SonosStatus struct {
	Available         bool   `json:"available"`
	Connected         bool   `json:"connected"`
	SelectedSpeakerID string `json:"selectedSpeakerId,omitempty"`
	Error             string `json:"error,omitempty"`
}

func NewSonosService(persistPath string) *SonosService {
	return &SonosService{
		persistPath: persistPath,
	}
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

	s.persistSelection(*selected)

	return nil
}

// DiscoverWithRetries retries discovery a few times before giving up.
func (s *SonosService) DiscoverWithRetries(
	ctx context.Context,
	attempts int,
) error {
	var err error

	for attempt := 1; attempt <= attempts; attempt++ {
		if _, err = s.Discover(ctx); err == nil {
			return nil
		}

		if attempt < attempts {
			select {
			case <-ctx.Done():
				return ctx.Err()

			case <-time.After(3 * time.Second):
			}
		}
	}

	return err
}

func (s *SonosService) Restore(ctx context.Context) error {
	persisted := s.readPersistedSelection()

	if persisted == nil {
		return s.DiscoverWithRetries(ctx, sonosDiscoveryAttempts)
	}

	client, err := sonos.NewClient(ctx, persisted.IP)
	if err == nil {
		s.mu.Lock()
		s.client = client
		s.selectedSpeakerID = persisted.SpeakerID
		s.available = true
		s.lastError = ""
		s.mu.Unlock()

		if listErr := s.DiscoverWithRetries(ctx, 1); listErr != nil {
			log.Printf("Sonos: refreshing speaker list failed: %v", listErr)
		}

		return nil
	}

	if err := s.DiscoverWithRetries(ctx, sonosDiscoveryAttempts); err != nil {
		return err
	}

	return s.Connect(ctx, persisted.SpeakerID)
}

func (s *SonosService) readPersistedSelection() *persistedSpeakerSelection {
	if s.persistPath == "" {
		return nil
	}

	data, err := os.ReadFile(s.persistPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("Sonos: reading persisted selection: %v", err)
		}

		return nil
	}

	var persisted persistedSpeakerSelection

	if err := json.Unmarshal(data, &persisted); err != nil {
		log.Printf("Sonos: parsing persisted selection: %v", err)
		return nil
	}

	if persisted.SpeakerID == "" || persisted.IP == "" {
		return nil
	}

	return &persisted
}

func (s *SonosService) persistSelection(speaker sonos.Speaker) {
	if s.persistPath == "" {
		return
	}

	data, err := json.Marshal(persistedSpeakerSelection{
		SpeakerID: speaker.ID,
		Name:      speaker.Name,
		IP:        speaker.IP,
	})
	if err != nil {
		log.Printf("Sonos: encoding persisted selection: %v", err)
		return
	}

	if err := os.WriteFile(s.persistPath, data, 0o600); err != nil {
		log.Printf("Sonos: saving persisted selection: %v", err)
	}
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

func (s *SonosService) Groups(
	ctx context.Context,
) ([]sonos.Group, error) {
	speakers := s.Speakers()

	groups, err := sonos.DiscoverGroups(ctx, speakers)
	if err != nil {
		return nil, fmt.Errorf("discovering Sonos groups: %w", err)
	}

	return groups, nil
}

func (s *SonosService) groupClient(
	ctx context.Context,
	groupID string,
) (*sonos.Client, sonos.Group, error) {
	speakers, group, err := s.groupDetails(ctx, groupID)
	if err != nil {
		return nil, sonos.Group{}, err
	}

	client, err := sonos.NewGroupClient(ctx, group, speakers)
	if err != nil {
		return nil, sonos.Group{}, fmt.Errorf(
			"creating client for group %q: %w",
			groupID,
			err,
		)
	}

	return client, group, nil
}

func (s *SonosService) groupDetails(
	ctx context.Context,
	groupID string,
) ([]sonos.Speaker, sonos.Group, error) {
	speakers := s.Speakers()

	if len(speakers) == 0 {
		return nil, sonos.Group{}, fmt.Errorf(
			"no speakers discovered; discover speakers first",
		)
	}

	groups, err := sonos.DiscoverGroups(ctx, speakers)
	if err != nil {
		return nil, sonos.Group{}, fmt.Errorf(
			"discovering Sonos groups: %w",
			err,
		)
	}

	for _, group := range groups {
		if group.ID == groupID {
			return speakers, group, nil
		}
	}

	return nil, sonos.Group{}, fmt.Errorf(
		"Sonos group %q not found",
		groupID,
	)
}

func (s *SonosService) PlayGroupURI(
	ctx context.Context,
	groupID string,
	uri string,
	metadata string,
) error {
	return s.withGroupClient(ctx, groupID, func(c *sonos.Client) error {
		return c.PlayURI(ctx, uri, metadata)
	})
}

func (s *SonosService) PlayGroupRadioStation(
	ctx context.Context,
	groupID string,
	name string,
) error {
	return s.withGroupClient(ctx, groupID, func(c *sonos.Client) error {
		return c.PlayRadioStation(ctx, name)
	})
}

func (s *SonosService) GetGroupRadioStations(
	ctx context.Context,
	groupID string,
) ([]sonos.RadioStation, error) {
	var stations []sonos.RadioStation

	err := s.withGroupClient(ctx, groupID, func(c *sonos.Client) error {
		var err error
		stations, err = c.GetRadioStations(ctx)
		return err
	})

	return stations, err
}

func (s *SonosService) StopGroup(
	ctx context.Context,
	groupID string,
) error {
	return s.withGroupClient(ctx, groupID, func(c *sonos.Client) error {
		return c.Stop(ctx)
	})
}

func (s *SonosService) SetGroupVolume(
	ctx context.Context,
	groupID string,
	volume int,
) error {
	if volume < 0 || volume > 100 {
		return fmt.Errorf("volume must be between 0 and 100")
	}

	speakers, group, err := s.groupDetails(ctx, groupID)
	if err != nil {
		return err
	}

	if len(group.Members) == 0 {
		return fmt.Errorf("Sonos group %q has no members", groupID)
	}

	for _, member := range group.Members {
		speaker, err := sonos.FindSpeaker(speakers, member.SpeakerID)
		if err != nil {
			return fmt.Errorf(
				"finding group member %q: %w",
				member.Name,
				err,
			)
		}

		client, err := sonos.NewClient(ctx, speaker.IP)
		if err != nil {
			return fmt.Errorf(
				"connecting to group member %q: %w",
				member.Name,
				err,
			)
		}

		if err := client.SetVolume(ctx, volume); err != nil {
			return fmt.Errorf(
				"setting volume for group member %q: %w",
				member.Name,
				err,
			)
		}
	}

	return nil
}

func (s *SonosService) PlaySpeakerURI(
	ctx context.Context,
	speakerID string,
	uri string,
	metadata string,
) error {
	return s.withSpeakerClient(ctx, speakerID, func(c *sonos.Client) error {
		return c.PlayURI(ctx, uri, metadata)
	})
}

func (s *SonosService) PlaySpeakerRadioStation(
	ctx context.Context,
	speakerID string,
	name string,
) error {
	return s.withSpeakerClient(ctx, speakerID, func(c *sonos.Client) error {
		return c.PlayRadioStation(ctx, name)
	})
}

func (s *SonosService) SetSpeakerVolume(
	ctx context.Context,
	speakerID string,
	volume int,
) error {
	return s.withSpeakerClient(ctx, speakerID, func(c *sonos.Client) error {
		return c.SetVolume(ctx, volume)
	})
}

func (s *SonosService) SetTargetVolume(
	ctx context.Context,
	target alarm.SonosTarget,
	volume int,
) error {
	switch target.Type {
	case alarm.TargetSpeaker:
		return s.SetSpeakerVolume(ctx, target.ID, volume)

	case alarm.TargetGroup:
		return s.SetGroupVolume(ctx, target.ID, volume)

	default:
		return fmt.Errorf("unsupported Sonos target type %q", target.Type)
	}
}

func (s *SonosService) RunGradualVolume(
	ctx context.Context,
	target alarm.SonosTarget,
	startVolume int,
	targetVolume int,
	duration time.Duration,
) error {
	if startVolume < 0 || startVolume > 100 {
		return fmt.Errorf("start volume must be between 0 and 100")
	}

	if targetVolume < 0 || targetVolume > 100 {
		return fmt.Errorf("volume must be between 0 and 100")
	}

	if duration <= 0 {
		return fmt.Errorf("volume duration must be greater than zero")
	}

	const steps = 10

	stepDelay := duration / steps
	if stepDelay <= 0 {
		return fmt.Errorf("volume duration is too short")
	}

	for step := 1; step <= steps; step++ {
		timer := time.NewTimer(stepDelay)

		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()

		case <-timer.C:
		}

		volume := startVolume + (targetVolume-startVolume)*step/steps

		if err := s.SetTargetVolume(ctx, target, volume); err != nil {
			return fmt.Errorf(
				"setting Sonos volume to %d%%: %w",
				volume,
				err,
			)
		}
	}

	return nil
}

func (s *SonosService) withGroupClient(
	ctx context.Context,
	groupID string,
	fn func(*sonos.Client) error,
) error {
	client, _, err := s.groupClient(ctx, groupID)
	if err != nil {
		return err
	}

	return fn(client)
}

func (s *SonosService) withSpeakerClient(
	ctx context.Context,
	speakerID string,
	fn func(*sonos.Client) error,
) error {
	client, err := s.speakerClient(ctx, speakerID)
	if err != nil {
		return err
	}

	return fn(client)
}

func (s *SonosService) speakerClient(
	ctx context.Context,
	speakerID string,
) (*sonos.Client, error) {
	speakers := s.Speakers()

	speaker, err := sonos.FindSpeaker(speakers, speakerID)
	if err != nil {
		return nil, fmt.Errorf("finding speaker %q: %w", speakerID, err)
	}

	client, err := sonos.NewClient(ctx, speaker.IP)
	if err != nil {
		return nil, fmt.Errorf(
			"connecting to speaker %q: %w",
			speaker.Name,
			err,
		)
	}

	return client, nil
}
