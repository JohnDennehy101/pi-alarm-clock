package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort          int
	MediaPort         int
	DataDir           string
	SonosEnabled      bool
	HueAPIKey         string
	ChromecastEnabled bool
	WeatherLatitude   float64
	WeatherLongitude  float64
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	port, err := getEnvAsInt("HTTP_PORT", 8080, 1, 65535)
	if err != nil {
		return nil, err
	}

	mediaPort, err := mediaPortEnv()
	if err != nil {
		return nil, err
	}

	if mediaPort == port {
		return nil, fmt.Errorf("MEDIA_PORT (%d) must differ from HTTP_PORT (%d)", mediaPort, port)
	}

	sonosEnabled, err := getEnvAsBool("SONOS_ENABLED", true)
	if err != nil {
		return nil, err
	}

	chromecastEnabled, err := getEnvAsBool("CHROMECAST_ENABLED", true)
	if err != nil {
		return nil, err
	}

	weatherLatitude, err := getEnvAsFloat("WEATHER_LATITUDE", 0, -90, 90)
	if err != nil {
		return nil, err
	}

	weatherLongitude, err := getEnvAsFloat("WEATHER_LONGITUDE", 0, -180, 180)
	if err != nil {
		return nil, err
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}

	return &Config{
		HTTPPort:          port,
		MediaPort:         mediaPort,
		DataDir:           dataDir,
		SonosEnabled:      sonosEnabled,
		HueAPIKey:         os.Getenv("HUE_API_KEY"),
		ChromecastEnabled: chromecastEnabled,
		WeatherLatitude:   weatherLatitude,
		WeatherLongitude:  weatherLongitude,
	}, nil
}

// mediaPortEnv reads MEDIA_PORT, falling back to AUDIO_PORT — the
// deprecated pre-rename spelling — so existing deployments keep working.
func mediaPortEnv() (int, error) {
	if os.Getenv("MEDIA_PORT") != "" {
		return getEnvAsInt("MEDIA_PORT", 8081, 1, 65535)
	}

	return getEnvAsInt("AUDIO_PORT", 8081, 1, 65535)
}

func getEnvAsInt(key string, defaultVal, min, max int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultVal, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %q (must be an integer)", key, value)
	}

	if parsed < min {
		return 0, fmt.Errorf("invalid %s: %d (must be >= %d)", key, parsed, min)
	}

	if parsed > max {
		return 0, fmt.Errorf("invalid %s: %d (must be <= %d)", key, parsed, max)
	}

	return parsed, nil
}

func getEnvAsBool(key string, defaultVal bool) (bool, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultVal, nil
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("invalid %s: %w", key, err)
	}

	return parsed, nil
}

func getEnvAsFloat(key string, defaultVal, min, max float64) (float64, error) {
	value := os.Getenv(key)
	if value == "" {
		return defaultVal, nil
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %q (must be a number)", key, value)
	}

	if parsed < min || parsed > max {
		return 0, fmt.Errorf("invalid %s: %v (must be between %g and %g)", key, parsed, min, max)
	}

	return parsed, nil
}
