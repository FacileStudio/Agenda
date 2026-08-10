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
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/tronc/middleware"
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

	// CDNProxies and CDNHeader are set when TRUSTED_PROXIES names a CDN.
	// They let httpx recover the visitor when the proxy in front replaced
	// the forwarded chain instead of extending it, which is Traefik's
	// default and leaves only the CDN's own header carrying the visitor.
	CDNProxies []netip.Prefix
	CDNHeader  string

	// TrustedProxies are the peers whose X-Forwarded-For httpx believes.
	// Unset means middleware.DefaultTrustedProxies: loopback and the
	// private ranges, which is the shape every app has behind Traefik.
	// TRUSTED_PROXIES=none trusts nothing and keys everything on the
	// connection address.
	TrustedProxies []netip.Prefix
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

	trustedProxies, err := TrustedProxies()
	if err != nil {
		return Core{}, err
	}
	cdnProxies, cdnHeader := CDN()

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
		TrustedProxies:     trustedProxies,
		CDNProxies:         cdnProxies,
		CDNHeader:          cdnHeader,
	}, nil
}

// TrustedProxySets are the names TRUSTED_PROXIES accepts alongside literal
// CIDR blocks, so a deployment says what it is fronted by instead of pasting
// two dozen ranges into its environment.
//
// "cloudflare" is opt-in and belongs here rather than in the default: an app
// that is not behind Cloudflare gains nothing from it, and a name in a config
// file is a statement about the deployment that the next person can read.
// Keeping the ranges in the library rather than in each app's environment is
// the whole point of a chassis — one bump refreshes every consumer, instead of
// a dozen copies of a list drifting apart.
var TrustedProxySets = map[string][]netip.Prefix{
	"private":    middleware.DefaultTrustedProxies,
	"cloudflare": middleware.CloudflareProxies,
}

// CDN reports the edge ranges and visitor header implied by TRUSTED_PROXIES.
//
// Naming a CDN is what opts into reading its header, because the two are the
// same statement: "this app is served through Cloudflare" is both why the edge
// is trusted and why Cf-Connecting-Ip means anything.
func CDN() ([]netip.Prefix, string) {
	for _, entry := range List("TRUSTED_PROXIES") {
		if strings.EqualFold(strings.TrimSpace(entry), "cloudflare") {
			return middleware.CloudflareProxies, "Cf-Connecting-Ip"
		}
	}
	return nil, ""
}

// TrustedProxies reads TRUSTED_PROXIES as a comma-separated list of CIDR
// blocks, bare addresses, and the names in TrustedProxySets.
//
//	TRUSTED_PROXIES=private,cloudflare
//	TRUSTED_PROXIES=10.0.0.0/8,192.168.1.7
//	TRUSTED_PROXIES=none
//
// Unset returns middleware.DefaultTrustedProxies, which is "private". The
// literal "none" returns a non-nil empty slice, which httpx honours as
// "believe no proxy" — spelling it out is deliberate, because an empty string
// is what a half-written deployment config produces and that must not silently
// mean the strictest setting an operator never chose.
func TrustedProxies() ([]netip.Prefix, error) {
	raw := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES"))
	if raw == "" {
		return middleware.DefaultTrustedProxies, nil
	}
	if strings.EqualFold(raw, "none") {
		return []netip.Prefix{}, nil
	}

	var prefixes []netip.Prefix
	var literals []string
	for _, entry := range List("TRUSTED_PROXIES") {
		if set, ok := TrustedProxySets[strings.ToLower(strings.TrimSpace(entry))]; ok {
			prefixes = append(prefixes, set...)
			continue
		}
		literals = append(literals, entry)
	}

	parsed, err := middleware.ParseTrustedProxies(literals)
	if err != nil {
		return nil, fmt.Errorf("env: TRUSTED_PROXIES takes CIDR blocks, IP addresses, or the names private and cloudflare: %w", err)
	}
	return append(prefixes, parsed...), nil
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
