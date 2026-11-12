package sonos

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/huin/goupnp"
	"github.com/huin/goupnp/dcps/av1"
)

const (
	sonosDeviceType = "urn:schemas-upnp-org:device:ZonePlayer:1"

	deviceDescriptionPath = "/xml/device_description.xml"

	instanceID uint32 = 0
	channel           = "Master"
)

type Speaker struct {
	Name string
	IP   string
}

type Client struct {
	avTransport *av1.AVTransport1
	rendering   *av1.RenderingControl1
	contentDir  *av1.ContentDirectory1
}

func DiscoverSpeakers(ctx context.Context) ([]Speaker, error) {
	devices, err := goupnp.DiscoverDevicesCtx(ctx, sonosDeviceType)

	if err != nil {
		return nil, fmt.Errorf("discovering Sonos speakers: %w", err)
	}

	var speakers []Speaker

	for _, device := range devices {

		if device.Err != nil {
			continue
		}

		if device.Root == nil {
			continue
		}

		if device.Location == nil {
			continue
		}

		host := device.Location.Hostname()
		ip := net.ParseIP(host)

		if ip == nil {
			continue
		}

		speakers = append(speakers, Speaker{
			Name: device.Root.Device.FriendlyName,
			IP:   ip.String(),
		})
	}

	if len(speakers) == 0 {
		return nil, fmt.Errorf("no Sonos speakers found")
	}

	return speakers, nil
}

func NewClient(ctx context.Context, speakerIP string) (*Client, error) {
	deviceURL, err := parseDeviceURL(speakerIP)
	if err != nil {
		return nil, err
	}

	rootDevice, err := goupnp.DeviceByURLCtx(ctx, deviceURL)
	if err != nil {
		return nil, fmt.Errorf("connecting to Sonos speaker: %w", err)
	}

	avTransport, err := initServiceClient(av1.NewAVTransport1ClientsFromRootDevice, rootDevice, deviceURL)
	if err != nil {
		return nil, fmt.Errorf("finding AVTransport service: %w", err)
	}

	contentDir, err := initServiceClient(av1.NewContentDirectory1ClientsFromRootDevice, rootDevice, deviceURL)
	if err != nil {
		return nil, fmt.Errorf("finding ContentDirectory service: %w", err)
	}

	rendering, err := initServiceClient(av1.NewRenderingControl1ClientsFromRootDevice, rootDevice, deviceURL)
	if err != nil {
		return nil, fmt.Errorf("finding RenderingControl service: %w", err)
	}

	return &Client{
		avTransport: avTransport,
		rendering:   rendering,
		contentDir:  contentDir,
	}, nil
}

func initServiceClient[T any](
	fetcher func(*goupnp.RootDevice, *url.URL) ([]T, error),
	rootDevice *goupnp.RootDevice,
	deviceURL *url.URL,
) (T, error) {
	clients, err := fetcher(rootDevice, deviceURL)
	if err != nil {
		var zero T
		return zero, err
	}

	if len(clients) == 0 {
		var zero T
		return zero, fmt.Errorf("speaker does not expose service")
	}

	return clients[0], nil
}

func parseDeviceURL(speakerIP string) (*url.URL, error) {
	if net.ParseIP(speakerIP) == nil {
		return nil, fmt.Errorf("invalid speaker IP address: %s", speakerIP)
	}

	return url.Parse(
		fmt.Sprintf(
			"http://%s%s",
			net.JoinHostPort(speakerIP, "1400"),
			deviceDescriptionPath,
		),
	)
}

func NewDiscoveredClient(
	ctx context.Context,
	speakerName string,
) (*Client, error) {
	speakers, err := DiscoverSpeakers(ctx)
	if err != nil {
		return nil, err
	}

	for _, speaker := range speakers {
		if speakerName == "" ||
			strings.EqualFold(speaker.Name, speakerName) {
			return NewClient(ctx, speaker.IP)
		}
	}

	return nil, fmt.Errorf("Sonos speaker %q not found", speakerName)
}

func (c *Client) PlayURI(
	ctx context.Context,
	uri string,
	metadata string,
) error {
	if uri == "" {
		return fmt.Errorf("audio URI cannot be empty")
	}

	if err := c.avTransport.SetAVTransportURICtx(
		ctx,
		instanceID,
		uri,
		metadata,
	); err != nil {
		return fmt.Errorf("setting playback URI: %w", err)
	}

	if err := c.avTransport.PlayCtx(
		ctx,
		instanceID,
		"1",
	); err != nil {
		return fmt.Errorf("starting playback: %w", err)
	}

	return nil
}

func (c *Client) SetVolume(
	ctx context.Context,
	volume int,
) error {
	if volume < 0 || volume > 100 {
		return fmt.Errorf("volume must be between 0 and 100")
	}

	return c.rendering.SetVolumeCtx(
		ctx,
		instanceID,
		channel,
		uint16(volume),
	)
}

func (c *Client) Stop(ctx context.Context) error {
	const instanceID uint32 = 0

	if err := c.avTransport.StopCtx(ctx, instanceID); err != nil {
		return fmt.Errorf("stopping playback: %w", err)
	}

	return nil
}
