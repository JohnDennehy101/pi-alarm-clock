package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/chromecast"
)

const (
	chromecastDefaultDisplayDurationMinutes = 60

	// loadRetries bounds how often a board load is attempted; the first
	// load after a TV wakes from standby often exceeds the cast
	// protocol's short launch timeout, but succeeds on the follow-up.
	loadRetries = 2

	loadRetryDelay = 5 * time.Second
)

type ChromecastStatus struct {
	Available        bool   `json:"available"`
	Connected        bool   `json:"connected"`
	SelectedDeviceID string `json:"selectedDeviceId,omitempty"`
	Error            string `json:"error,omitempty"`
}

type ChromecastService struct {
	mu sync.Mutex

	devices          []chromecast.Device
	selectedDeviceID string
	connected        bool
	available        bool
	lastError        string
	persistPath      string

	boardProvider BoardProvider
}

type persistedCastSelection struct {
	DeviceID string `json:"deviceId"`
	Name     string `json:"name,omitempty"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
}

func NewChromecastService(persistPath string) *ChromecastService {
	return &ChromecastService{
		persistPath: persistPath,
	}
}

func (s *ChromecastService) setBoardProvider(provider BoardProvider) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.boardProvider = provider
}

func (s *ChromecastService) Discover(
	ctx context.Context,
) ([]chromecast.Device, error) {
	devices, err := chromecast.DiscoverDevices(ctx)
	if err != nil {
		s.mu.Lock()
		s.lastError = err.Error()
		s.mu.Unlock()

		return nil, fmt.Errorf("discovering Chromecast devices: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.devices = devices
	s.available = true
	s.lastError = ""

	return append([]chromecast.Device(nil), devices...), nil
}

func (s *ChromecastService) Devices() []chromecast.Device {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]chromecast.Device(nil), s.devices...)
}

func (s *ChromecastService) Connect(
	ctx context.Context,
	deviceID string,
) error {
	device, err := s.device(ctx, deviceID)
	if err != nil {
		return err
	}

	if err := s.verifyConnection(ctx, device); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.selectedDeviceID = device.ID
	s.connected = true
	s.lastError = ""

	s.persistSelection(device)

	return nil
}

func (s *ChromecastService) verifyConnection(
	ctx context.Context,
	device chromecast.Device,
) error {
	client, err := chromecast.Connect(ctx, device)
	if err != nil {
		s.mu.Lock()
		s.lastError = err.Error()
		s.mu.Unlock()

		return err
	}

	defer client.Close()

	return nil
}

func (s *ChromecastService) SelectedDevice() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.selectedDeviceID
}

func (s *ChromecastService) Status() ChromecastStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	return ChromecastStatus{
		Available:        s.available,
		Connected:        s.connected,
		SelectedDeviceID: s.selectedDeviceID,
		Error:            s.lastError,
	}
}

func (s *ChromecastService) CastBoard(
	ctx context.Context,
	deviceID string,
	windowMinutes int,
) error {
	urls, err := s.boardURLs(ctx, time.Now().Truncate(time.Minute), windowMinutes)
	if err != nil {
		return err
	}

	client, err := s.connectBoardRetry(ctx, deviceID, urls[0])
	if err != nil {
		return err
	}

	return client.Close()
}

func (s *ChromecastService) RunBoardCast(
	ctx context.Context,
	deviceID string,
	windowMinutes int,
) error {
	if windowMinutes <= 0 {
		windowMinutes = chromecastDefaultDisplayDurationMinutes
	}

	start := time.Now().Truncate(time.Minute)

	urls, err := s.boardURLs(ctx, start, windowMinutes)
	if err != nil {
		return err
	}

	client, err := s.connectBoardRetry(ctx, deviceID, urls[0])
	if err != nil {
		return err
	}

	defer client.Close()

	for {
		timer := time.NewTimer(time.Until(time.Now().Add(time.Minute).Truncate(time.Minute)))

		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()

		case <-timer.C:
		}

		minute := int(time.Since(start) / time.Minute)
		if minute >= len(urls) {
			if err := client.Stop(); err != nil {
				log.Printf("Chromecast: stopping board after display window failed: %v", err)
			}

			log.Printf(
				"Chromecast: display window ended after %d minutes",
				windowMinutes,
			)

			return nil
		}

		if err := client.LoadImage(ctx, castURL(urls[minute])); err != nil {
			log.Printf("Chromecast: refreshing board failed: %v", err)

			client.Close()

			reconnected, reconnectErr := s.connectBoard(ctx, deviceID)
			if reconnectErr != nil {
				log.Printf("Chromecast: reconnecting failed: %v", reconnectErr)
				continue
			}

			client = reconnected
		}
	}
}

func (s *ChromecastService) boardURLs(
	ctx context.Context,
	start time.Time,
	windowMinutes int,
) ([]string, error) {
	s.mu.Lock()
	provider := s.boardProvider
	s.mu.Unlock()

	if provider == nil {
		return nil, fmt.Errorf("board provider is not configured")
	}

	if windowMinutes < 1 {
		windowMinutes = 1
	}

	return provider.BoardURLs(ctx, start, windowMinutes)
}

func (s *ChromecastService) connectBoardRetry(
	ctx context.Context,
	deviceID string,
	imageURL string,
) (*chromecast.Client, error) {
	var err error

	for attempt := 1; attempt <= loadRetries; attempt++ {
		client, connectErr := s.connectBoard(ctx, deviceID)
		if connectErr != nil {
			return nil, connectErr
		}

		err = client.LoadImage(ctx, castURL(imageURL))
		if err == nil {
			return client, nil
		}

		client.Close()

		if attempt < loadRetries {
			log.Printf(
				"Chromecast: board load failed (attempt %d/%d): %v",
				attempt,
				loadRetries,
				err,
			)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()

			case <-time.After(loadRetryDelay):
			}
		}
	}

	return nil, err
}

func (s *ChromecastService) StopBoard(
	ctx context.Context,
	deviceID string,
) error {
	device, err := s.device(ctx, deviceID)
	if err != nil {
		return err
	}

	client, err := chromecast.Connect(ctx, device)
	if err != nil {
		return err
	}

	defer client.Close()

	return client.Stop()
}

func (s *ChromecastService) connectBoard(
	ctx context.Context,
	deviceID string,
) (*chromecast.Client, error) {
	device, err := s.device(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	return chromecast.Connect(ctx, device)
}

func castURL(imageURL string) string {
	separator := "?"
	if strings.Contains(imageURL, "?") {
		separator = "&"
	}

	return imageURL + separator + "t=" +
		strconv.FormatInt(time.Now().UnixNano(), 10)
}

func (s *ChromecastService) device(
	ctx context.Context,
	deviceID string,
) (chromecast.Device, error) {
	s.mu.Lock()
	devices := append([]chromecast.Device(nil), s.devices...)
	s.mu.Unlock()

	device, err := chromecast.FindDevice(devices, deviceID)
	if err == nil {
		return device, nil
	}

	devices, err = s.Discover(ctx)
	if err != nil {
		return chromecast.Device{}, fmt.Errorf(
			"Chromecast device %q not found: %w",
			deviceID,
			err,
		)
	}

	return chromecast.FindDevice(devices, deviceID)
}

func (s *ChromecastService) Restore(ctx context.Context) error {
	persisted := s.readPersistedSelection()

	if persisted == nil {
		_, err := s.Discover(ctx)
		if err != nil {
			return err
		}

		return nil
	}

	device := chromecast.Device{
		ID:   persisted.DeviceID,
		Name: persisted.Name,
		IP:   persisted.IP,
		Port: persisted.Port,
	}

	s.mu.Lock()
	s.devices = []chromecast.Device{device}
	s.mu.Unlock()

	if err := s.verifyConnection(ctx, device); err == nil {
		s.mu.Lock()
		s.selectedDeviceID = device.ID
		s.connected = true
		s.available = true
		s.lastError = ""
		s.mu.Unlock()

		if _, listErr := s.Discover(ctx); listErr != nil {
			log.Printf("Chromecast: refreshing device list failed: %v", listErr)
		}

		return nil
	}

	if _, err := s.Discover(ctx); err != nil {
		return err
	}

	return s.Connect(ctx, device.ID)
}

func (s *ChromecastService) readPersistedSelection() *persistedCastSelection {
	if s.persistPath == "" {
		return nil
	}

	data, err := os.ReadFile(s.persistPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("Chromecast: reading persisted selection: %v", err)
		}

		return nil
	}

	var persisted persistedCastSelection

	if err := json.Unmarshal(data, &persisted); err != nil {
		log.Printf("Chromecast: parsing persisted selection: %v", err)
		return nil
	}

	if persisted.DeviceID == "" || persisted.IP == "" || persisted.Port == 0 {
		return nil
	}

	return &persisted
}

func (s *ChromecastService) persistSelection(device chromecast.Device) {
	if s.persistPath == "" {
		return
	}

	data, err := json.Marshal(persistedCastSelection{
		DeviceID: device.ID,
		Name:     device.Name,
		IP:       device.IP,
		Port:     device.Port,
	})
	if err != nil {
		log.Printf("Chromecast: encoding persisted selection: %v", err)
		return
	}

	if err := os.WriteFile(s.persistPath, data, 0o600); err != nil {
		log.Printf("Chromecast: saving persisted selection: %v", err)
	}
}
