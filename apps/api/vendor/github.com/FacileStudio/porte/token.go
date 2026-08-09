package porte

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
)

// tokenBytes is the entropy behind every credential porte issues: session
// tokens, CLI login codes, OIDC state and nonces. It is the 32 bytes the apps
// already use.
const tokenBytes = 32

// NewToken returns a random opaque credential, URL-safe so it survives a
// cookie, a header and a redirect without encoding.
func NewToken() (string, error) {
	buffer := make([]byte, tokenBytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

// HashToken returns the stored form of a token. SHA-256 is right here and
// argon2 is not: the input is 256 bits of entropy porte generated itself, so
// there is no dictionary to slow down, and this runs on every authenticated
// request.
//
// The encoding is hex, matching what all six apps already store, so an
// adopting app's existing session rows keep authenticating.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// SecureCompare compares two credentials without leaking their contents
// through timing. Five of the six apps compare the OIDC state with a plain
// !=; this is Plume's line, which is the one that was right.
func SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
