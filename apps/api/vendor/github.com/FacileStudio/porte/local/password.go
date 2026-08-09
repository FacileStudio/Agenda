package local

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

// The argon2id parameters the suite already runs: 64 MiB, three passes, two
// lanes. They are not tuned here — they are copied from the apps so that every
// password hash already in a Facile database keeps verifying after the move,
// which is what makes adopting porte a code change and not a password reset.
const (
	argon2Memory      uint32 = 64 * 1024
	argon2Iterations  uint32 = 3
	argon2Parallelism uint8  = 2
	argon2SaltLength  int    = 16
	argon2KeyLength   uint32 = 32
)

// HashPassword returns a PHC-encoded argon2id hash: the encoding carries the
// parameters, so raising the cost later verifies old hashes at their own
// settings instead of locking everyone out.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argon2SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey([]byte(password), salt, argon2Iterations, argon2Memory, argon2Parallelism, argon2KeyLength)
	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argon2Memory, argon2Iterations, argon2Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	), nil
}

// VerifyPassword reports whether password matches the encoded hash, comparing
// in constant time. A malformed or empty encoding is false for every input,
// which is what makes an account with no password — one created through SSO —
// impossible to sign into with one.
func VerifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false
	}

	var memory, iterations uint32
	var parallelism uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism); err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	got := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

var (
	dummyOnce sync.Once
	dummyHash string
)

// EqualizeTiming runs a verification against a fixed throwaway hash so that a
// login attempt for an unknown address costs about what one for a known
// address costs.
//
// Without it the login form answers "no such account" in a millisecond and
// "wrong password" in sixty, which is the same oracle as different error
// messages, only harder to notice in review.
func EqualizeTiming(password string) {
	dummyOnce.Do(func() {
		if hash, err := HashPassword("timing-equalizer-not-a-secret"); err == nil {
			dummyHash = hash
		}
	})
	if dummyHash != "" {
		VerifyPassword(password, dummyHash)
	}
}
