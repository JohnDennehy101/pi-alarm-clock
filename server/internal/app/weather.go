package app

import (
	"context"

	"github.com/johndennehy101/pi-alarm-clock/server/internal/weather"
)

type WeatherService struct {
	client *weather.Client
}

func NewWeatherService(latitude, longitude float64) *WeatherService {
	client, err := weather.NewWeatherService(latitude, longitude)
	if err != nil {
		return nil
	}

	return &WeatherService{
		client: client,
	}
}

func (s *WeatherService) Forecast(ctx context.Context) (*weather.Forecast, error) {
	if s == nil || s.client == nil {
		return nil, weather.ErrNotConfigured
	}

	return s.client.Forecast(ctx)
}
