package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort     int
	SonosEnabled bool
	HueAPIKey    string
	APIToken     string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	port, err := getEnvAsInt("HTTP_PORT", 8080, 1, 65535)
	if err != nil {
		return nil, err
	}

	sonosEnabled, err := getEnvAsBool("SONOS_ENABLED", true)
	if err != nil {
		return nil, err
	}

	apiToken, err := getRequiredEnv("API_TOKEN")
	if err != nil {
		return nil, err
	}

	return &Config{
		HTTPPort:     port,
		SonosEnabled: sonosEnabled,
		HueAPIKey:    os.Getenv("HUE_API_KEY"),
		APIToken:     apiToken,
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

func getRequiredEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("%s not set", key)
	}
	return value, nil
}
