package pg

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"strings"

	"github.com/FacileStudio/porte"
)

const (
	sealedPrefix = "pv1:"
	sealInfo     = "porte/pg token sealing v1"
	sealAAD      = "porte/pg identity token"
	secretSize   = 32
)

// ErrTokenUnreadable reports a sealed value no configured key opens, or one
// that is malformed. The store never returns it from a read: it treats the
// token as absent instead.
var ErrTokenUnreadable = stderrors.New("porte/pg: sealed token is unreadable")

// TokenSealer encrypts the OIDC tokens porte_identities keeps, with
// AES-256-GCM under a key derived from an app secret.
//
// A sealed value is "pv1:" followed by the base64url of a random 12-byte nonce
// and the ciphertext. Anything without the prefix is plaintext, which is what
// lets an app turn sealing on over existing rows without a migration.
//
// The first key seals; every key opens. Rotation is therefore: put the new
// secret first, keep the old one after it, run [IdentityStore.ResealIdentities],
// then drop the old one.
type TokenSealer struct {
	aeads []cipher.AEAD
}

// NewTokenSealer builds a sealer from 32-byte app secrets, newest first. Each
// secret goes through HKDF-SHA256 with a porte-specific info string, so the
// same secret used elsewhere in the app never yields the same key here.
func NewTokenSealer(secrets ...[]byte) (*TokenSealer, error) {
	if len(secrets) == 0 {
		return nil, stderrors.New("porte/pg: a token sealer needs at least one secret")
	}
	sealer := &TokenSealer{}
	for _, secret := range secrets {
		if len(secret) != secretSize {
			return nil, fmt.Errorf("porte/pg: a token secret must be %d bytes, got %d", secretSize, len(secret))
		}
		key, err := hkdf.Key(sha256.New, secret, nil, sealInfo, secretSize)
		if err != nil {
			return nil, fmt.Errorf("porte/pg: derive token key: %w", err)
		}
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, fmt.Errorf("porte/pg: token cipher: %w", err)
		}
		aead, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("porte/pg: token cipher: %w", err)
		}
		sealer.aeads = append(sealer.aeads, aead)
	}
	return sealer, nil
}

// ParseTokenSealer builds a sealer from the value of PORTE_TOKEN_KEY: one or
// more comma-separated secrets, newest first, each 64 hex characters or
// standard base64 of 32 bytes. An empty value returns a nil sealer and no
// error, which leaves sealing off.
func ParseTokenSealer(spec string) (*TokenSealer, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, nil
	}
	var secrets [][]byte
	for _, part := range strings.Split(spec, ",") {
		secret, err := decodeSecret(strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, secret)
	}
	return NewTokenSealer(secrets...)
}

func decodeSecret(text string) ([]byte, error) {
	if secret, err := hex.DecodeString(text); err == nil && len(secret) == secretSize {
		return secret, nil
	}
	secret, err := base64.StdEncoding.DecodeString(text)
	if err != nil || len(secret) != secretSize {
		return nil, fmt.Errorf("porte/pg: a token secret must be %d bytes as hex or base64", secretSize)
	}
	return secret, nil
}

// Seal encrypts plain under the first key. The empty string stays empty, so a
// missing token is not turned into a value.
func (s *TokenSealer) Seal(plain string) (string, error) {
	if s == nil || plain == "" {
		return plain, nil
	}
	aead := s.aeads[0]
	nonce := make([]byte, aead.NonceSize(), aead.NonceSize()+len(plain)+aead.Overhead())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("porte/pg: seal token: %w", err)
	}
	sealed := aead.Seal(nonce, nonce, []byte(plain), []byte(sealAAD))
	return sealedPrefix + base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Open returns the plaintext of stored. A value without the "pv1:" prefix is
// plaintext and comes back unchanged. A sealed value no key opens returns
// [ErrTokenUnreadable]. GCM verifies its tag in constant time.
func (s *TokenSealer) Open(stored string) (string, error) {
	plain, _, err := s.open(stored)
	return plain, err
}

func (s *TokenSealer) open(stored string) (plain string, current bool, err error) {
	body, sealed := strings.CutPrefix(stored, sealedPrefix)
	if !sealed {
		return stored, false, nil
	}
	if s == nil {
		return "", false, ErrTokenUnreadable
	}
	raw, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return "", false, ErrTokenUnreadable
	}
	for i, aead := range s.aeads {
		if len(raw) < aead.NonceSize() {
			return "", false, ErrTokenUnreadable
		}
		nonce, text := raw[:aead.NonceSize()], raw[aead.NonceSize():]
		if opened, err := aead.Open(nil, nonce, text, []byte(sealAAD)); err == nil {
			return string(opened), i == 0, nil
		}
	}
	return "", false, ErrTokenUnreadable
}

func (s *TokenSealer) sealTokens(tokens porte.TokenSet) (access, refresh string, err error) {
	if access, err = s.Seal(tokens.AccessToken); err != nil {
		return "", "", err
	}
	if refresh, err = s.Seal(tokens.RefreshToken); err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (s *TokenSealer) openTokens(tokens porte.TokenSet) porte.TokenSet {
	tokens.AccessToken, _ = s.Open(tokens.AccessToken)
	tokens.RefreshToken, _ = s.Open(tokens.RefreshToken)
	return tokens
}
