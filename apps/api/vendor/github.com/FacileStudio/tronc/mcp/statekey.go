package mcp

import "crypto/sha256"

// StateKey derives the key that signs an app's confirmation state from one
// of its secrets: SHA-256 over "<app>-mcp-state:" followed by secret. Every
// app derived exactly this before it moved here, so questions signed before
// an app adopts it still verify.
func StateKey(app, secret string) []byte {
	sum := sha256.Sum256([]byte(app + "-mcp-state:" + secret))
	return sum[:]
}
