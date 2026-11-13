package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort     int
	AudioPort    int
	DataDir      string
	SonosEnabled bool
	HueAPIKey    string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	port, err := getEnvAsInt("HTTP_PORT", 8080, 1, 65535)
	if err != nil {
		return nil, err
	}

	audioPort, err := getEnvAsInt("AUDIO_PORT", 8081, 1, 65535)
	if err != nil {
		return nil, err
	}

	if audioPort == port {
		return nil, fmt.Errorf("AUDIO_PORT (%d) must differ from HTTP_PORT (%d)", audioPort, port)
	}

	sonosEnabled, err := getEnvAsBool("SONOS_ENABLED", true)
	if err != nil {
		return nil, err
	}

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "data"
	}

	return &Config{
		HTTPPort:     port,
		AudioPort:    audioPort,
		DataDir:      dataDir,
		SonosEnabled: sonosEnabled,
		HueAPIKey:    os.Getenv("HUE_API_KEY"),
	}, nil
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
