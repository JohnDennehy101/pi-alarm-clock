package weather

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	requestTimeout = 10 * time.Second
	cacheTTL       = 10 * time.Minute
	apiURL         = "https://api.open-meteo.com/v1/forecast"
	timeLayout     = "2006-01-02T15:04"
)

var (
	ErrNotConfigured = errors.New("weather is not configured")

	httpClient = &http.Client{
		Timeout: requestTimeout,
	}
)

type Client struct {
	latitude  float64
	longitude float64

	fetchMu sync.Mutex

	cacheMu  sync.RWMutex
	cachedAt time.Time
	forecast *Forecast
}

type Forecast struct {
	Current Current
	Hourly  []Conditions
}

type Current struct {
	Time        time.Time
	Temperature float64
	FeelsLike   float64
	WindSpeed   float64
	Code        int
}

type Conditions struct {
	Time          time.Time
	Temperature   float64
	Precipitation int
	Code          int
}

func NewWeatherService(latitude, longitude float64) (*Client, error) {
	if err := validateCoordinates(latitude, longitude); err != nil {
		return nil, err
	}

	return &Client{
		latitude:  latitude,
		longitude: longitude,
	}, nil
}

func validateCoordinates(latitude, longitude float64) error {
	if !isValidCoordinate(latitude, -90, 90) ||
		!isValidCoordinate(longitude, -180, 180) {
		return fmt.Errorf("weather: invalid coordinates")
	}

	return nil
}

func isValidCoordinate(value, min, max float64) bool {
	return !math.IsNaN(value) &&
		!math.IsInf(value, 0) &&
		value >= min &&
		value <= max
}

func (c *Client) Forecast(ctx context.Context) (*Forecast, error) {
	if c == nil {
		return nil, ErrNotConfigured
	}

	// Check cache before acquiring the fetch lock.
	c.cacheMu.RLock()
	if f := c.forecast; f != nil && time.Since(c.cachedAt) < cacheTTL {
		c.cacheMu.RUnlock()
		return f, nil
	}
	c.cacheMu.RUnlock()

	// Prevent duplicate concurrent fetches.
	c.fetchMu.Lock()
	defer c.fetchMu.Unlock()

	// Another caller may have refreshed the cache.
	c.cacheMu.RLock()
	if f := c.forecast; f != nil && time.Since(c.cachedAt) < cacheTTL {
		c.cacheMu.RUnlock()
		return f, nil
	}
	c.cacheMu.RUnlock()

	forecast, err := c.fetch(ctx)
	if err != nil {
		// Return stale data if available.
		c.cacheMu.RLock()
		if f := c.forecast; f != nil {
			c.cacheMu.RUnlock()
			return f, nil
		}
		c.cacheMu.RUnlock()

		return nil, err
	}

	c.cacheMu.Lock()
	c.forecast = forecast
	c.cachedAt = time.Now()
	c.cacheMu.Unlock()

	return forecast, nil
}

func (c *Client) fetch(ctx context.Context) (*Forecast, error) {
	u, err := url.Parse(apiURL)
	if err != nil {
		return nil, fmt.Errorf("building weather URL: %w", err)
	}

	q := u.Query()
	q.Set("latitude", fmt.Sprintf("%.4f", c.latitude))
	q.Set("longitude", fmt.Sprintf("%.4f", c.longitude))
	q.Set("current", "temperature_2m,apparent_temperature,weather_code,wind_speed_10m")
	q.Set("hourly", "temperature_2m,precipitation_probability,weather_code")
	q.Set("forecast_days", "2")
	q.Set("timezone", "auto")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		u.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("building weather request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("requesting weather: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather API status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading weather response: %w", err)
	}

	return c.parseResponse(body)
}

func (c *Client) parseResponse(body []byte) (*Forecast, error) {
	payload, err := decodeResponse(body)
	if err != nil {
		return nil, err
	}

	location := time.FixedZone(
		payload.Timezone,
		payload.UTCOffsetSeconds,
	)

	current, err := parseCurrent(payload.Current, location)
	if err != nil {
		return nil, err
	}

	hourly, err := parseHourly(payload.Hourly, location)
	if err != nil {
		return nil, err
	}

	return &Forecast{
		Current: current,
		Hourly:  hourly,
	}, nil
}

func decodeResponse(body []byte) (*openMeteoResponse, error) {
	var payload openMeteoResponse

	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decoding weather response: %w", err)
	}

	return &payload, nil
}

func parseCurrent(data currentResponse, location *time.Location) (Current, error) {
	t, err := time.ParseInLocation(timeLayout, data.Time, location)
	if err != nil {
		return Current{}, fmt.Errorf("parsing current time: %w", err)
	}

	return Current{
		Time:        t,
		Temperature: data.Temperature,
		FeelsLike:   data.ApparentTemperature,
		WindSpeed:   data.WindSpeed,
		Code:        data.Code,
	}, nil
}

func parseHourly(data hourlyResponse, location *time.Location) ([]Conditions, error) {
	if len(data.Time) == 0 {
		return nil, fmt.Errorf("weather response contained no hourly data")
	}

	hourly := make([]Conditions, 0, len(data.Time))

	for i, ts := range data.Time {
		t, err := time.ParseInLocation(timeLayout, ts, location)
		if err != nil {
			return nil, fmt.Errorf("parsing hourly time %q: %w", ts, err)
		}

		conditions := Conditions{Time: t}

		if i < len(data.Temperature) {
			conditions.Temperature = data.Temperature[i]
		}

		if i < len(data.Precipitation) {
			conditions.Precipitation = data.Precipitation[i]
		}

		if i < len(data.Code) {
			conditions.Code = data.Code[i]
		}

		hourly = append(hourly, conditions)
	}

	return hourly, nil
}

type currentResponse struct {
	Time                string  `json:"time"`
	Temperature         float64 `json:"temperature_2m"`
	ApparentTemperature float64 `json:"apparent_temperature"`
	WindSpeed           float64 `json:"wind_speed_10m"`
	Code                int     `json:"weather_code"`
}

type hourlyResponse struct {
	Time          []string  `json:"time"`
	Temperature   []float64 `json:"temperature_2m"`
	Precipitation []int     `json:"precipitation_probability"`
	Code          []int     `json:"weather_code"`
}

type openMeteoResponse struct {
	Timezone         string          `json:"timezone"`
	UTCOffsetSeconds int             `json:"utc_offset_seconds"`
	Current          currentResponse `json:"current"`
	Hourly           hourlyResponse  `json:"hourly"`
}
