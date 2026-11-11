package mcp

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

// Digest fingerprints the fields of a target for Kind.Fingerprint: the hex
// of the first 16 bytes of SHA-256 over the fields joined by NUL. The five
// apps computed exactly this before it moved here, so the digests of open
// actions stay valid across the move.
func Digest(fields ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(fields, "\x00")))
	return hex.EncodeToString(sum[:16])
}

// TargetIDs reads an action whose user id and target are both decimal ids,
// or returns refusal when either is not.
func TargetIDs(a Action, refusal error) (userID, targetID int64, err error) {
	userID, errUser := strconv.ParseInt(a.UserID, 10, 64)
	targetID, errTarget := strconv.ParseInt(a.Target, 10, 64)
	if errUser != nil || errTarget != nil {
		return 0, 0, refusal
	}
	return userID, targetID, nil
}
