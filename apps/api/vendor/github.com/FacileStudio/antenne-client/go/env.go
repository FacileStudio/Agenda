package antenne

import (
	"errors"
	"os"
	"strings"
)

// ErrNotConfigured is returned by ConfigFromEnv when neither ANTENNE_URL nor
// ANTENNE_INSTANCE is set, which callers treat as event publishing turned off.
var ErrNotConfigured = errors.New("antenne: ANTENNE_URL is not set")

// ConfigFromEnv builds a Config from the environment. The instance URL is read
// from ANTENNE_URL, or from its older synonym ANTENNE_INSTANCE; ANTENNE_URL wins
// when both are set. ANTENNE_APP overrides app, ANTENNE_SECRET and
// ANTENNE_INSTANCE_ID fill the matching fields. Events are left empty for the
// caller to set.
func ConfigFromEnv(app string) (*Config, error) {
	instance := firstEnv("ANTENNE_URL", "ANTENNE_INSTANCE")
	if instance == "" {
		return nil, ErrNotConfigured
	}
	cfg := &Config{
		App:        firstNonBlank(os.Getenv("ANTENNE_APP"), app),
		Instance:   strings.TrimRight(instance, "/"),
		Secret:     strings.TrimSpace(os.Getenv("ANTENNE_SECRET")),
		InstanceID: strings.TrimSpace(os.Getenv("ANTENNE_INSTANCE_ID")),
	}
	if cfg.App == "" || cfg.Secret == "" {
		return nil, errors.New("antenne: ANTENNE_SECRET and an app name are required when ANTENNE_URL is set")
	}
	return cfg, nil
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
