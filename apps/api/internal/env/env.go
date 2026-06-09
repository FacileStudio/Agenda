package env

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"api/internal/crypto"
)

type OIDCConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	SuccessURL   string
}

type Config struct {
	DatabaseURL        string
	Port               string
	CORSAllowedOrigins []string
	LogLevel           string
	StorageDir         string
	OIDC               *OIDCConfig
	SSOOnly            bool
	EncryptionKey      []byte
}

func Load() (Config, error) {
	env := Config{
		DatabaseURL:        valueOrDefault("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/agenda?sslmode=disable"),
		Port:               valueOrDefault("PORT", "4000"),
		LogLevel:           valueOrDefault("LOG_LEVEL", "info"),
		StorageDir:         valueOrDefault("STORAGE_DIR", "./data"),
		CORSAllowedOrigins: csvOrDefault("DOMAINS", []string{}),
	}

	port, err := strconv.Atoi(env.Port)
	if err != nil || port < 1 || port > 65535 {
		return Config{}, fmt.Errorf("PORT must be a valid TCP port")
	}
	if err := validateOrigins(env.CORSAllowedOrigins); err != nil {
		return Config{}, err
	}
	if err := validateLogLevel(env.LogLevel); err != nil {
		return Config{}, err
	}

	env.SSOOnly = strings.ToLower(os.Getenv("SSO_ONLY")) == "true"

	if issuer := os.Getenv("OIDC_ISSUER"); issuer != "" {
		clientID := os.Getenv("OIDC_CLIENT_ID")
		clientSecret := os.Getenv("OIDC_CLIENT_SECRET")
		redirectURL := os.Getenv("OIDC_REDIRECT_URL")
		if clientID == "" || clientSecret == "" || redirectURL == "" {
			return Config{}, fmt.Errorf("OIDC_CLIENT_ID, OIDC_CLIENT_SECRET, and OIDC_REDIRECT_URL are required when OIDC_ISSUER is set")
		}
		successURL := os.Getenv("OIDC_SUCCESS_URL")
		if successURL == "" && len(env.CORSAllowedOrigins) > 0 {
			successURL = env.CORSAllowedOrigins[0]
		}
		env.OIDC = &OIDCConfig{
			Issuer:       issuer,
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			SuccessURL:   successURL,
		}
	}

	if ek := os.Getenv("ENCRYPTION_KEY"); ek != "" {
		env.EncryptionKey = crypto.DeriveKey(ek)
	}

	return env, nil
}

func valueOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func csvOrDefault(key string, fallback []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return []string{}
	}
	return out
}

func validateOrigins(origins []string) error {
	for _, origin := range origins {
		if origin == "*" {
			continue
		}
		if strings.HasPrefix(origin, "http://") || strings.HasPrefix(origin, "https://") {
			continue
		}
		return fmt.Errorf("DOMAINS contains invalid origin %q", origin)
	}

	return nil
}

func validateLogLevel(level string) error {
	switch strings.ToLower(level) {
	case "debug", "info", "warn", "error":
		return nil
	default:
		return fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, error")
	}
}
