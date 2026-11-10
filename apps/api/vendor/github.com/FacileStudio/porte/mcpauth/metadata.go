package mcpauth

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

// wellKnownPath is the RFC 9728 well-known suffix.
const wellKnownPath = "/.well-known/oauth-protected-resource"

// ProtectedResourceMetadata is the RFC 9728 document for an MCP endpoint, for
// auth.ProtectedResourceMetadataHandler. resource must be the URL users type
// into their client, because Claude refuses a document whose resource differs.
// issuer is Registre, the only authorization server, and scopes are what the
// endpoint asks for, normally ScopesFor(prefix).List(). offline_access is not
// a resource scope and does not belong here.
func ProtectedResourceMetadata(resource, issuer string, scopes []string) *oauthex.ProtectedResourceMetadata {
	return &oauthex.ProtectedResourceMetadata{
		Resource:                          resource,
		AuthorizationServers:              []string{issuer},
		ScopesSupported:                   scopes,
		BearerMethodsSupported:            []string{"header"},
		ResourceSigningAlgValuesSupported: []string{"RS256"},
	}
}

// MetadataURL is where RFC 9728 places resource's metadata: the well-known
// suffix inserted between the host and the path, so
// https://app.example/mcp maps to
// https://app.example/.well-known/oauth-protected-resource/mcp. It is the
// value of RequireBearerTokenOptions.ResourceMetadataURL, and its path is the
// route the metadata handler is mounted on.
func MetadataURL(resource string) (string, error) {
	parsed, err := url.Parse(resource)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("porte/mcpauth: resource %q is not an absolute URI", resource)
	}
	suffix := strings.TrimSuffix(parsed.EscapedPath(), "/")
	metadata := url.URL{Scheme: parsed.Scheme, Host: parsed.Host}
	return metadata.String() + wellKnownPath + suffix, nil
}
