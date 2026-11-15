package hue

import (
	"context"
	"fmt"
	"strings"
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

func (c *Client) SetLightPower(
	ctx context.Context,
	lightID string,
	on bool,
) error {
	if strings.TrimSpace(lightID) == "" {
		return fmt.Errorf("Hue light ID is required")
	}

	err := c.home.UpdateLight(ctx, lightID, openhue.LightPut{
		On: &openhue.On{
			On: on,
		},
	})
	if err != nil {
		return fmt.Errorf("setting power for light %s: %w", lightID, err)
	}

	return nil
}

func (c *Client) SetLightBrightness(
	ctx context.Context,
	lightID string,
	brightness int,
) error {
	if strings.TrimSpace(lightID) == "" {
		return fmt.Errorf("Hue light ID is required")
	}

	if brightness < 0 || brightness > 100 {
		return fmt.Errorf("brightness must be between 0 and 100")
	}

	if brightness == 0 {
		return c.SetLightPower(ctx, lightID, false)
	}

	err := c.home.UpdateLight(ctx, lightID, openhue.LightPut{
		On: &openhue.On{
			On: true,
		},
		Dimming: &openhue.Dimming{
			Brightness: float32(brightness),
		},
	})
	if err != nil {
		return fmt.Errorf("setting brightness for light %s: %w", lightID, err)
	}

	return nil
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
		if err := c.SetLightBrightness(ctx, light.Id, brightness); err != nil {
			return err
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
		if err := c.SetLightPower(ctx, light.Id, true); err != nil {
			return err
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
		if err := c.SetLightPower(ctx, light.Id, false); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) SetLightColourTemperature(
	ctx context.Context,
	lightID string,
	temperature int,
) error {
	if strings.TrimSpace(lightID) == "" {
		return fmt.Errorf("Hue light ID is required")
	}

	if temperature < 2000 || temperature > 6500 {
		return fmt.Errorf("colour temperature must be between 2000 and 6500 Kelvin")
	}

	mirek := 1000000 / temperature

	err := c.home.UpdateLight(ctx, lightID, openhue.LightPut{
		ColorTemperature: &openhue.ColorTemperature{
			Mirek: &mirek,
		},
	})
	if err != nil {
		return fmt.Errorf("setting colour temperature for light %s: %w", lightID, err)
	}

	return nil
}
