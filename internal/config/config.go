// Package config loads service configuration from environment variables.
// Twelve-Factor: no hardcoded URLs, secrets, or timeouts (skill Rule 6).
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration. Extend per migration phase.
type Config struct {
	Env      string // dev | qa | staging | prod
	Port     string
	LogLevel string // debug | info | warn | error
	Version  string // build/revision id (canary identification)

	// DefaultBrand is used when the inbound request omits x-brand-id.
	DefaultBrand string

	// PersonalizationEnabled is the environment gate. Effective personalization
	// is (this AND the CMS feature flag) — see HomeService flag merge (rule #3).
	PersonalizationEnabled bool

	ContentService ContentServiceConfig
	GroupBy        GroupByConfig
	Jewel          JewelConfig
	Salesforce     SalesforceConfig
	ATG            ATGConfig
	SearchFacade   SearchFacadeConfig
	Auth           AuthConfig
	Tracing        TracingConfig
}

// TracingConfig controls OpenTelemetry tracing. Enabled when an OTLP endpoint is
// configured (standard OTEL_EXPORTER_OTLP_ENDPOINT); otherwise propagation-only.
type TracingConfig struct {
	Enabled     bool
	ServiceName string
}

// AuthConfig configures authentication. Mode is chosen by which URL is set:
// OpaqueExchangeURL (digital_bff-style cookie exchange) > JWKSURL (local JWT) > dev
// (x-profile-id header).
type AuthConfig struct {
	JWKSURL      string
	Issuer       string
	Audience     string
	ProfileClaim string // JWT mode: claim holding the profile id (default "prn")
	Timeout      time.Duration

	// OpaqueExchangeURL is the Auth service base for /v2/auth/exchange-token; when set,
	// ms_home exchanges the session cookie for claims (matches digital_bff).
	OpaqueExchangeURL string
	CookieName        string // session cookie name (default "SessionId")
}

// SearchFacadeConfig targets the Search Facade. Optional: empty URL disables
// banner_products (multi-product details).
type SearchFacadeConfig struct {
	BaseURL string
	Timeout time.Duration
}

// ATGConfig targets the ATG cart-header endpoint. Optional: empty URL disables
// favorite-store resolution and the continue-buying shortcut.
type ATGConfig struct {
	CartHeaderURL string
	Timeout       time.Duration
}

// SalesforceConfig targets the Salesforce actions endpoint. Optional: empty URL
// disables the Salesforce-backed strategies.
type SalesforceConfig struct {
	URL     string
	Timeout time.Duration
}

// GroupByConfig targets the GroupBy services. Each URL is optional; an empty URL
// means the corresponding strategy is not registered.
type GroupByConfig struct {
	SearchURL          string
	RecommendationsURL string
	Timeout            time.Duration
}

// JewelConfig targets the Jewel recommendation service. Optional.
type JewelConfig struct {
	URL     string
	Timeout time.Duration
}

// ContentServiceConfig targets the existing Content Service proxy (SHARED_CONTENT_SERVICE_*).
// Per migration decision, ms_home keeps calling this proxy rather than Contentstack directly.
type ContentServiceConfig struct {
	BaseURL string
	Timeout time.Duration
}

// Load reads configuration from the environment, applying safe defaults for
// local DX. Returns an error only for invalid values, not missing optionals.
func Load() (Config, error) {
	cfg := Config{
		Env:                    getEnv("ENV", "dev"),
		Port:                   getEnv("PORT", "8080"),
		LogLevel:               getEnv("LOG_LEVEL", "info"),
		Version:                getEnv("BUILD_VERSION", "dev"),
		DefaultBrand:           getEnv("DEFAULT_BRAND", "LP"),
		PersonalizationEnabled: getBool("PERSONALIZATION_ENABLED", false),
		ContentService: ContentServiceConfig{
			BaseURL: getEnv("SHARED_CONTENT_SERVICE_URL", ""),
			Timeout: getDuration("SHARED_CONTENT_SERVICE_TIMEOUT", 5*time.Second),
		},
		GroupBy: GroupByConfig{
			SearchURL:          getEnv("SHARED_GROUPBY_SEARCH_URL", ""),
			RecommendationsURL: getEnv("SHARED_GROUPBY_RECOMMENDATIONS_URL", ""),
			Timeout:            getDuration("SHARED_GROUPBY_TIMEOUT", 5*time.Second),
		},
		Jewel: JewelConfig{
			URL:     getEnv("SHARED_JEWEL_URL", ""),
			Timeout: getDuration("SHARED_JEWEL_TIMEOUT", 5*time.Second),
		},
		Salesforce: SalesforceConfig{
			URL:     getEnv("SALESFORCE_MODULE_HTTP", ""),
			Timeout: getDuration("SALESFORCE_MODULE_TIMEOUT", 5*time.Second),
		},
		ATG: ATGConfig{
			CartHeaderURL: getEnv("SHARED_ATG_CART_HEADER_URL", ""),
			Timeout:       getDuration("SHARED_ATG_TIMEOUT", 5*time.Second),
		},
		SearchFacade: SearchFacadeConfig{
			BaseURL: getEnv("SHARED_SEARCH_FACADE_URL", ""),
			Timeout: getDuration("SHARED_SEARCH_FACADE_TIMEOUT", 5*time.Second),
		},
		Auth: AuthConfig{
			JWKSURL:           getEnv("AUTH_JWKS_URL", ""),
			Issuer:            getEnv("AUTH_ISSUER", ""),
			Audience:          getEnv("AUTH_AUDIENCE", ""),
			ProfileClaim:      getEnv("AUTH_PROFILE_CLAIM", "prn"),
			Timeout:           getDuration("AUTH_TIMEOUT", 5*time.Second),
			OpaqueExchangeURL: getEnv("AUTH_OPAQUE_EXCHANGE_URL", ""),
			CookieName:        getEnv("AUTH_COOKIE_NAME", "SessionId"),
		},
		Tracing: TracingConfig{
			Enabled:     getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "") != "",
			ServiceName: getEnv("OTEL_SERVICE_NAME", "ms_home"),
		},
	}

	if cfg.ContentService.BaseURL == "" {
		return Config{}, fmt.Errorf("config: SHARED_CONTENT_SERVICE_URL is required")
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
