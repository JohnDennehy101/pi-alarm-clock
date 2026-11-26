package chromecast

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vishen/go-chromecast/application"
	pb "github.com/vishen/go-chromecast/cast/proto"
	"github.com/vishen/go-chromecast/dns"
)

const (
	// discoveryTimeout bounds how long mDNS discovery listens for cast
	// devices before giving up.
	discoveryTimeout = 5 * time.Second

	// connectionRetries limits how often the cast connection is retried
	// when the device is unreachable (each retry waits 2 seconds).
	connectionRetries = 2
)

type Device struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	IP   string `json:"ip"`
	Port int    `json:"port"`
}

func DiscoverDevices(ctx context.Context) ([]Device, error) {
	ctx, cancel := context.WithTimeout(ctx, discoveryTimeout)
	defer cancel()

	entries, err := dns.DiscoverCastDNSEntries(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("discovering Chromecast devices: %w", err)
	}

	devicesByID := make(map[string]Device)

	for entry := range entries {
		device := Device{
			ID:   entry.GetUUID(),
			Name: entry.GetName(),
			IP:   entry.GetAddr(),
			Port: entry.GetPort(),
		}

		if device.ID == "" || device.IP == "" || device.Port == 0 {
			continue
		}

		devicesByID[device.ID] = device
	}

	devices := make([]Device, 0, len(devicesByID))
	for _, device := range devicesByID {
		devices = append(devices, device)
	}

	if len(devices) == 0 {
		return nil, fmt.Errorf("no Chromecast devices found")
	}

	sort.Slice(devices, func(i, j int) bool {
		nameI := strings.ToLower(devices[i].Name)
		nameJ := strings.ToLower(devices[j].Name)

		if nameI == nameJ {
			return devices[i].ID < devices[j].ID
		}

		return nameI < nameJ
	})

	return devices, nil
}

func FindDevice(devices []Device, deviceID string) (Device, error) {
	for _, device := range devices {
		if device.ID == deviceID {
			return device, nil
		}
	}

	return Device{}, fmt.Errorf("Chromecast device %q not found", deviceID)
}

type Client struct {
	app *application.Application
}

type receiverAppState struct {
	mu      sync.Mutex
	lastApp string
}

func (s *receiverAppState) handleStatus(payload []byte) {
	var status struct {
		Status struct {
			Applications []struct {
				AppID       string `json:"appId"`
				DisplayName string `json:"displayName"`
			} `json:"applications"`
		} `json:"status"`
	}
	if err := json.Unmarshal(payload, &status); err != nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(status.Status.Applications) == 0 {
		if s.lastApp != "" {
			log.Printf("Chromecast: receiver has no running app (backdrop)")
			s.lastApp = ""
		}
		return
	}

	// Match the library: take the last application in the list.
	app := status.Status.Applications[len(status.Status.Applications)-1]

	if app.AppID != s.lastApp {
		log.Printf(
			"Chromecast: receiver app is %q (%q)",
			app.AppID,
			app.DisplayName,
		)
		s.lastApp = app.AppID
	}
}

func parseCastMessageType(msg *pb.CastMessage) (payload []byte, msgType string, ok bool) {
	if msg == nil || msg.PayloadUtf8 == nil {
		return nil, "", false
	}

	payload = []byte(*msg.PayloadUtf8)

	var header struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &header); err != nil || header.Type == "" {
		return nil, "", false
	}

	return payload, header.Type, true
}

func newReceiverEventWatcher() application.CastMessageFunc {
	state := &receiverAppState{}

	return func(msg *pb.CastMessage) {
		payload, msgType, ok := parseCastMessageType(msg)
		if !ok {
			return
		}

		switch msgType {
		case "CLOSE":
			log.Printf("Chromecast: receiver closed the app session")
		case "LOAD_FAILED", "LOAD_CANCELLED":
			log.Printf("Chromecast: receiver reported %s for a media load", msgType)
		case "RECEIVER_STATUS":
			state.handleStatus(payload)
		}
	}
}

func Connect(ctx context.Context, device Device) (*Client, error) {
	if net.ParseIP(device.IP) == nil {
		return nil, fmt.Errorf("invalid Chromecast IP address: %s", device.IP)
	}

	if device.Port <= 0 {
		return nil, fmt.Errorf("invalid Chromecast port: %d", device.Port)
	}

	app := application.NewApplication(
		application.WithCacheDisabled(true),
		application.WithConnectionRetries(connectionRetries),
	)

	app.AddMessageFunc(newReceiverEventWatcher())

	if err := app.Start(device.IP, device.Port); err != nil {
		app.Close(false)
		return nil, fmt.Errorf("connecting to Chromecast %q: %w", device.Name, err)
	}

	return &Client{app: app}, nil
}

func (c *Client) LoadImage(ctx context.Context, uri string) error {
	if uri == "" {
		return fmt.Errorf("image URI cannot be empty")
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	if err := c.app.Load(uri, 0, "", false, true, false); err != nil {
		return fmt.Errorf("loading image on Chromecast: %w", err)
	}

	return nil
}

func (c *Client) Stop() error {
	if err := c.app.Stop(); err != nil {
		return fmt.Errorf("stopping Chromecast: %w", err)
	}

	return nil
}

func (c *Client) Close() error {
	return c.app.Close(false)
}
