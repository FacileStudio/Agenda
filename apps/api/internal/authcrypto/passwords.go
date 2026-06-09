package authcrypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2Memory      uint32 = 64 * 1024
	argon2Iterations  uint32 = 3
	argon2Parallelism uint8  = 2
	argon2SaltLength         = 16
	argon2KeyLength   uint32 = 32
)

func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password required")
	}

	saltBytes := make([]byte, argon2SaltLength)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), saltBytes, argon2Iterations, argon2Memory, argon2Parallelism, argon2KeyLength)
	return encodeArgon2ID(saltBytes, hash), nil
}

func VerifyPassword(password, encodedHash string) bool {
	params, saltBytes, wantHash, err := decodeArgon2ID(encodedHash)
	if err != nil || password == "" {
		return false
	}

	gotHash := argon2.IDKey([]byte(password), saltBytes, params.iterations, params.memory, params.parallelism, uint32(len(wantHash)))
	return subtle.ConstantTimeCompare(gotHash, wantHash) == 1
}

type argon2Params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

func encodeArgon2ID(saltBytes []byte, hash []byte) string {
	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory,
		argon2Iterations,
		argon2Parallelism,
		base64.RawStdEncoding.EncodeToString(saltBytes),
		base64.RawStdEncoding.EncodeToString(hash),
	)
}

func decodeArgon2ID(encoded string) (argon2Params, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 {
		return argon2Params{}, nil, nil, errors.New("invalid encoded hash format")
	}
	if parts[1] != "argon2id" || parts[2] != "v=19" {
		return argon2Params{}, nil, nil, errors.New("unsupported hash algorithm")
	}

	params, err := parseArgon2Params(parts[3])
	if err != nil {
		return argon2Params{}, nil, nil, err
	}

	saltBytes, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil || len(saltBytes) == 0 {
		return argon2Params{}, nil, nil, errors.New("invalid salt")
	}

	hashBytes, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil || len(hashBytes) == 0 {
		return argon2Params{}, nil, nil, errors.New("invalid hash")
	}

	return params, saltBytes, hashBytes, nil
}

func parseArgon2Params(value string) (argon2Params, error) {
	parts := strings.Split(value, ",")
	if len(parts) != 3 {
		return argon2Params{}, errors.New("invalid argon2 params")
	}

	var params argon2Params
	for _, part := range parts {
		key, raw, ok := strings.Cut(part, "=")
		if !ok {
			return argon2Params{}, errors.New("invalid argon2 param entry")
		}

		switch key {
		case "m":
			parsed, err := strconv.ParseUint(raw, 10, 32)
			if err != nil {
				return argon2Params{}, err
			}
			params.memory = uint32(parsed)
		case "t":
			parsed, err := strconv.ParseUint(raw, 10, 32)
			if err != nil {
				return argon2Params{}, err
			}
			params.iterations = uint32(parsed)
		case "p":
			parsed, err := strconv.ParseUint(raw, 10, 8)
			if err != nil {
				return argon2Params{}, err
			}
			params.parallelism = uint8(parsed)
		default:
			return argon2Params{}, errors.New("unknown argon2 param")
		}
	}

	if params.memory == 0 || params.iterations == 0 || params.parallelism == 0 {
		return argon2Params{}, errors.New("missing argon2 params")
	}

	return params, nil
}
