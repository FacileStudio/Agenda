// Package env reads the configuration every Facile API shares, and offers
// typed helpers for the rest.
//
// It settles two naming splits the suite carries today. CORS origins are read
// from CORS_ALLOWED_ORIGINS, the GoSvelteBoilerplate name, falling back to the
// six names that drifted out of it so a repo can adopt tronc without its
// deployment config changing in the same breath. And APP_ENV is introduced:
// no Go app had an environment-name variable at all.
//
// APP_ENV never gates security behaviour. CORS is decided by the configured
// origin list alone, so a missing APP_ENV cannot open an app up.
package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Environment names the deployment an app is running in.
type Environment string

const (
	Development Environment = "development"
	Staging     Environment = "staging"
	Production  Environment = "production"
)

// CORSOriginKeys are read in order; the first one set wins. The names after
// the first are the drift, kept readable so adoption is a code-only change.
var CORSOriginKeys = []string{
	"CORS_ALLOWED_ORIGINS",
	"ALLOWED_ORIGINS",
	"DOMAINS",
	"DOMAIN",
	"CORS_ORIGINS",
	"TRUSTED_ORIGINS",
	"CLIENT_ORIGIN",
}

// Core is the configuration shared by every Go API. Apps keep their own
// fields in their own struct and use the helpers below to read them.
type Core struct {
	AppEnv             Environment
	Port               int
	LogLevel           string
	DatabaseURL        string
	CORSAllowedOrigins []string
	JournalURL         string
	JournalToken       string
}

// IsProduction reports whether AppEnv is production.
func (c Core) IsProduction() bool { return c.AppEnv == Production }

// LoadCore reads Core from the process environment. DATABASE_URL is required;
// everything else has a default. Use LoadCoreWithout for a service that has no
// database.
func LoadCore() (Core, error) {
	return loadCore(true)
}

// LoadCoreWithout is LoadCore for a service with no database of its own, so
// DATABASE_URL is read if present and not required. Mycelium is the case that
// prompted it: it stores its state as files, and requiring a database URL would
// have kept it off the shared configuration entirely.
func LoadCoreWithout() (Core, error) {
	return loadCore(false)
}

func loadCore(requireDatabase bool) (Core, error) {
	port, err := Int("PORT", 8080)
	if err != nil {
		return Core{}, err
	}

	databaseURL := String("DATABASE_URL", "")
	if requireDatabase && databaseURL == "" {
		return Core{}, fmt.Errorf("env: DATABASE_URL is required")
	}

	return Core{
		AppEnv:             ParseEnvironment(String("APP_ENV", string(Development))),
		Port:               port,
		LogLevel:           String("LOG_LEVEL", "info"),
		DatabaseURL:        databaseURL,
		CORSAllowedOrigins: CORSOrigins(),
		JournalURL:         String("JOURNAL_URL", ""),
		JournalToken:       String("JOURNAL_TOKEN", ""),
	}, nil
}

// ParseEnvironment maps a name onto an Environment, defaulting to development.
func ParseEnvironment(value string) Environment {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "production", "prod":
		return Production
	case "staging", "stage":
		return Staging
	default:
		return Development
	}
}

// CORSOrigins reads the first CORSOriginKeys entry that is set and splits it
// on commas. An unset list means no cross-origin caller is allowed.
func CORSOrigins() []string {
	for _, key := range CORSOriginKeys {
		if value := os.Getenv(key); strings.TrimSpace(value) != "" {
			return List(key)
		}
	}
	return nil
}

// String reads key, or returns fallback when it is unset or blank.
func String(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

// Required reads key and fails when it is unset or blank.
func Required(key string) (string, error) {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value, nil
	}
	return "", fmt.Errorf("env: %s is required", key)
}

// Int reads key as an integer.
func Int(key string, fallback int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("env: %s must be an integer, got %q", key, raw)
	}
	return value, nil
}

// Bool reads key as a boolean, accepting the strconv.ParseBool spellings.
func Bool(key string, fallback bool) (bool, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("env: %s must be a boolean, got %q", key, raw)
	}
	return value, nil
}

// Duration reads key as a Go duration, such as 30s or 5m.
func Duration(key string, fallback time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("env: %s must be a duration, got %q", key, raw)
	}
	return value, nil
}

// List reads key as a comma-separated list, dropping blank entries.
func List(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			values = append(values, trimmed)
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}
