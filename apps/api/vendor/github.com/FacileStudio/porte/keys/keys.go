// Package keys provides API key minting, hashing, storage, and authentication middleware.
package keys

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/FacileStudio/porte"
)

// Kind discriminates secret backend tokens from public browser tokens.
type Kind string

const (
	KindSecret Kind = "secret"
	KindPublic Kind = "public"
)

// Key is one registered API key record.
type Key struct {
	ID             int64      `json:"id"`
	App            string     `json:"app"`
	Kind           Kind       `json:"kind"`
	Prefix         string     `json:"prefix"`
	TokenHash      string     `json:"-"`
	AllowedOrigins []string   `json:"allowed_origins"`
	DailyQuota     int        `json:"daily_quota"`
	UsedToday      int64      `json:"used_today,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
}

// CreateRequest carries parameters to mint a new API key.
type CreateRequest struct {
	App            string   `json:"app"`
	Kind           Kind     `json:"kind"`
	AllowedOrigins []string `json:"allowed_origins,omitempty"`
	DailyQuota     int      `json:"daily_quota,omitempty"`
}

// CreateResponse returns the created key metadata and the raw one-time token.
type CreateResponse struct {
	Key   Key    `json:"key"`
	Token string `json:"token"`
}

// Store is the storage contract for API keys.
type Store interface {
	Create(ctx context.Context, key Key) (Key, error)
	FindByHash(ctx context.Context, tokenHash string) (Key, error)
	List(ctx context.Context, app string) ([]Key, error)
	Revoke(ctx context.Context, id int64) error
	RecordUsage(ctx context.Context, keyID int64, count int64) error
	UsageToday(ctx context.Context) (map[int64]int64, error)
}

// GenerateToken generates a raw token string, its safe prefix, and its SHA-256 hash.
func GenerateToken(servicePrefix string, app string, kind Kind) (string, string, string, error) {
	entropy := make([]byte, 32)
	if _, err := rand.Read(entropy); err != nil {
		return "", "", "", fmt.Errorf("porte/keys: generate entropy: %w", err)
	}

	randomPart := hex.EncodeToString(entropy)
	var rawToken string
	if kind == KindPublic {
		rawToken = fmt.Sprintf("%s_pub_%s_%s", servicePrefix, app, randomPart)
	} else {
		rawToken = fmt.Sprintf("%s_%s_%s", servicePrefix, app, randomPart)
	}

	prefixLen := 16
	if len(rawToken) < prefixLen {
		prefixLen = len(rawToken)
	}
	prefix := rawToken[:prefixLen]
	tokenHash := porte.HashToken(rawToken)

	return rawToken, prefix, tokenHash, nil
}
