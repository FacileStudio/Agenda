package env

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/FacileStudio/Agenda/apps/api/internal/crypto"
	troncenv "github.com/FacileStudio/tronc/env"
)

type OIDCConfig struct {
	Issuer       string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	SuccessURL   string
}

type Config struct {
	troncenv.Core
	StorageDir    string
	OIDC          *OIDCConfig
	SSOOnly       bool
	EncryptionKey []byte
}

func Load() (Config, error) {
	core, err := loadCore()
	if err != nil {
		return Config{}, err
	}
	cfg := Config{
		Core:       core,
		StorageDir: troncenv.String("STORAGE_DIR", "./data"),
	}

	if cfg.SSOOnly, err = troncenv.Bool("SSO_ONLY", false); err != nil {
		return Config{}, err
	}

	if issuer := troncenv.String("OIDC_ISSUER", ""); issuer != "" {
		oidc := &OIDCConfig{
			Issuer:       issuer,
			ClientID:     troncenv.String("OIDC_CLIENT_ID", ""),
			ClientSecret: troncenv.String("OIDC_CLIENT_SECRET", ""),
			RedirectURL:  troncenv.String("OIDC_REDIRECT_URL", ""),
			SuccessURL:   troncenv.String("OIDC_SUCCESS_URL", ""),
		}
		if oidc.ClientID == "" || oidc.ClientSecret == "" || oidc.RedirectURL == "" {
			return Config{}, fmt.Errorf("OIDC_CLIENT_ID, OIDC_CLIENT_SECRET, and OIDC_REDIRECT_URL are required when OIDC_ISSUER is set")
		}
		if oidc.SuccessURL == "" {
			oidc.SuccessURL = firstAbsoluteOrigin(cfg.CORSAllowedOrigins)
		}
		cfg.OIDC = oidc
	}

	if key := troncenv.String("ENCRYPTION_KEY", ""); key != "" {
		cfg.EncryptionKey = crypto.DeriveKey(key)
	}

	return cfg, nil
}

// loadCore fills troncenv.Core without troncenv.LoadCore, which requires
// DATABASE_URL. The deployment only sets DB_USER and DB_PASSWORD, so the DSN is
// still assembled from the DB_* variables when DATABASE_URL is absent.
func loadCore() (troncenv.Core, error) {
	port, err := troncenv.Int("PORT", 4000)
	if err != nil {
		return troncenv.Core{}, err
	}
	if port < 1 || port > 65535 {
		return troncenv.Core{}, fmt.Errorf("PORT must be a valid TCP port")
	}

	databaseURL, err := resolveDatabaseURL()
	if err != nil {
		return troncenv.Core{}, err
	}

	return troncenv.Core{
		AppEnv:             troncenv.ParseEnvironment(troncenv.String("APP_ENV", string(troncenv.Development))),
		Port:               port,
		LogLevel:           troncenv.String("LOG_LEVEL", "info"),
		DatabaseURL:        databaseURL,
		CORSAllowedOrigins: troncenv.CORSOrigins(),
		JournalURL:         troncenv.String("JOURNAL_URL", ""),
		JournalToken:       troncenv.String("JOURNAL_TOKEN", ""),
	}, nil
}

func resolveDatabaseURL() (string, error) {
	if dsn := troncenv.String("DATABASE_URL", ""); dsn != "" {
		return dsn, nil
	}
	dbUser := troncenv.String("DB_USER", "")
	dbPassword := troncenv.String("DB_PASSWORD", "")
	if dbUser == "" || dbPassword == "" {
		return "", fmt.Errorf("set DATABASE_URL, or DB_USER and DB_PASSWORD, to connect to PostgreSQL")
	}
	u := &url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(dbUser, dbPassword),
		Host:     troncenv.String("DB_HOST", "db") + ":" + troncenv.String("DB_PORT", "5432"),
		Path:     "/" + troncenv.String("DB_NAME", "agenda"),
		RawQuery: "sslmode=" + troncenv.String("DB_SSLMODE", "disable"),
	}
	return u.String(), nil
}

// firstAbsoluteOrigin returns the first origin usable as a redirect target.
// The CORS list is also fed by DOMAIN, which deployments set to a bare
// hostname; redirecting to one of those would resolve against the callback path
// instead of leaving the site.
func firstAbsoluteOrigin(origins []string) string {
	for _, origin := range origins {
		if strings.HasPrefix(origin, "http://") || strings.HasPrefix(origin, "https://") {
			return origin
		}
	}
	return ""
}
