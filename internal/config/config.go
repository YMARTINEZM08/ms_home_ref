package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds every externally-supplied value. No field is hardcoded.
// All env var names are documented in configs/.env.example.
type Config struct {
	Port           string
	ServiceName    string
	Environment    string // dev | qa | staging | prod
	LogLevel       string // OFF | ERROR | WARN | INFO | DEBUG | TRACE
	ContentService ContentServiceConfig
	DefaultBrand   string
	Breaker        BreakerConfig
	OTEL           OTELConfig
}

type ContentServiceConfig struct {
	URL        string
	Timeout    time.Duration
	HomePageID string // page identifier appended to the content-service path (e.g. tienda/home)
}

type BreakerConfig struct {
	FailureRatio float64
	MinRequests  uint32
	OpenTimeout  time.Duration
}

type OTELConfig struct {
	Endpoint    string // host:port or http(s)://host:port
	SampleRatio float64
}

// Load reads and validates all configuration from environment variables.
// Returns a descriptive error for every invalid or missing required value.
func Load() (*Config, error) {
	csURL, err := requireEnv("CONTENT_SERVICE_URL")
	if err != nil {
		return nil, err
	}

	failureRatio := float64OrDefault("BREAKER_FAILURE_RATIO", 0.05)
	if failureRatio <= 0 || failureRatio > 1 {
		return nil, fmt.Errorf("BREAKER_FAILURE_RATIO must be between 0 and 1, got %v", failureRatio)
	}

	return &Config{
		Port:        envOrDefault("PORT", "8080"),
		ServiceName: envOrDefault("SERVICE_NAME", "ms-home-liverpool"),
		Environment: envOrDefault("ENVIRONMENT", "dev"),
		LogLevel:    envOrDefault("LOG_LEVEL", "info"),
		ContentService: ContentServiceConfig{
			URL:        csURL,
			Timeout:    time.Duration(intOrDefault("CONTENT_SERVICE_TIMEOUT_MS", 30000)) * time.Millisecond,
			HomePageID: envOrDefault("HOME_PAGE_ID", "tienda/home"),
		},
		DefaultBrand: envOrDefault("DEFAULT_BRAND", "LP"),
		Breaker: BreakerConfig{
			FailureRatio: failureRatio,
			MinRequests:  uint32(intOrDefault("BREAKER_MIN_REQUESTS", 20)),
			OpenTimeout:  time.Duration(intOrDefault("BREAKER_OPEN_TIMEOUT_S", 30)) * time.Second,
		},
		OTEL: OTELConfig{
			Endpoint:    os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
			SampleRatio: float64OrDefault("OTEL_SAMPLE_RATIO", 1.0),
		},
	}, nil
}

func requireEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required environment variable %q is not set", key)
	}
	return v, nil
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func intOrDefault(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

func float64OrDefault(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return fallback
}
