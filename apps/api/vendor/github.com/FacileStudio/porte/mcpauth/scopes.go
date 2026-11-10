package mcpauth

import (
	"slices"

	"github.com/modelcontextprotocol/go-sdk/auth"
)

// Scopes are an app's two MCP scopes, `<prefix>.read` and `<prefix>.write`,
// the same names Registre's token exchange grants and porte's ScopePrefix gate
// reads.
type Scopes struct {
	Read  string
	Write string
}

// ScopesFor returns the scopes of the app named prefix, such as "courrier".
func ScopesFor(prefix string) Scopes {
	return Scopes{Read: prefix + ".read", Write: prefix + ".write"}
}

// List returns both scopes, read first, as scopes_supported wants them.
func (s Scopes) List() []string {
	return []string{s.Read, s.Write}
}

// HasScope reports whether the token grants scope. A write grant implies
// read, as on porte's Authorization header path, because Verifier adds the
// read scope to any token that holds the write one.
func HasScope(info *auth.TokenInfo, scope string) bool {
	return info != nil && slices.Contains(info.Scopes, scope)
}

// withImpliedRead returns granted plus s.Read when granted holds s.Write
// without it.
func withImpliedRead(granted []string, s Scopes) []string {
	if !slices.Contains(granted, s.Write) || slices.Contains(granted, s.Read) {
		return granted
	}
	return append(slices.Clone(granted), s.Read)
}
