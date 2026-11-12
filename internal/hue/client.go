package hue

import (
	"context"
	"fmt"
	"time"

	"github.com/openhue/openhue-go"
)

type Client struct {
	home *openhue.Home
	ip   string
}

func NewClient(ctx context.Context, apiKey string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Hue API key is required")
	}

	discovery, err := openhue.NewBridgeDiscovery(
		openhue.WithTimeout(3 * time.Second),
	).Discover()
	if err != nil {
		return nil, fmt.Errorf("discovering Hue bridge: %w", err)
	}

	home, err := openhue.NewHome(
		discovery.IpAddress,
		apiKey,
		openhue.WithRequestTimeout(10*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("creating Hue client: %w", err)
	}

	return &Client{
		home: home,
		ip:   discovery.IpAddress,
	}, nil
}

func (c *Client) IP() string {
	return c.ip
}

func (c *Client) GetLights(ctx context.Context) (map[string]openhue.LightGet, error) {
	lights, err := c.home.GetLights(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting Hue lights: %w", err)
	}

	return lights, nil
}

func (c *Client) SetBrightness(ctx context.Context, brightness int) error {
	if brightness < 1 || brightness > 100 {
		return fmt.Errorf("brightness must be between 1 and 100")
	}

	lights, err := c.GetLights(ctx)
	if err != nil {
		return err
	}

	for _, light := range lights {
		err := c.home.UpdateLight(ctx, light.Id, openhue.LightPut{
			On: &openhue.On{
				On: true,
			},
			Dimming: &openhue.Dimming{
				Brightness: float32(brightness),
			},
		})
		if err != nil {
			return fmt.Errorf("setting brightness for light %s: %w", light.Id, err)
		}
	}

	return nil
}

func (c *Client) TurnOn(ctx context.Context) error {
	lights, err := c.GetLights(ctx)
	if err != nil {
		return err
	}

	for _, light := range lights {
		err := c.home.UpdateLight(ctx, light.Id, openhue.LightPut{
			On: &openhue.On{
				On: true,
			},
		})
		if err != nil {
			return fmt.Errorf("turning on light %s: %w", light.Id, err)
		}
	}

	return nil
}

func (c *Client) TurnOff(ctx context.Context) error {
	lights, err := c.GetLights(ctx)
	if err != nil {
		return err
	}

	for _, light := range lights {
		err := c.home.UpdateLight(ctx, light.Id, openhue.LightPut{
			On: &openhue.On{
				On: false,
			},
		})
		if err != nil {
			return fmt.Errorf("turning off light %s: %w", light.Id, err)
		}
	}

	return nil
}
