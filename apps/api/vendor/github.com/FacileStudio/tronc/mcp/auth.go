package mcp

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

// wellKnown is the RFC 9728 well-known prefix.
const wellKnown = "/.well-known/oauth-protected-resource"

// clockSkew is the leeway on token expiry, matching porte's JWT check, so a
// token the verifier accepted is not refused a second later by the SDK.
const clockSkew = 30 * time.Second

// ProtectedResourceMetadata is the RFC 9728 document for resource. resource
// must be the URL users type into their client, because Claude refuses a
// document whose resource differs. offline_access is not a resource scope
// and does not belong in scopes.
func ProtectedResourceMetadata(resource, issuer string, scopes []string) *oauthex.ProtectedResourceMetadata {
	return &oauthex.ProtectedResourceMetadata{
		Resource:                          resource,
		AuthorizationServers:              []string{issuer},
		ScopesSupported:                   scopes,
		BearerMethodsSupported:            []string{"header"},
		ResourceSigningAlgValuesSupported: []string{"RS256"},
	}
}

// MetadataURL is where RFC 9728 places the metadata of resource: the
// well-known prefix between the host and the path, so
// https://app.example/mcp maps to
// https://app.example/.well-known/oauth-protected-resource/mcp.
func MetadataURL(resource *url.URL) string {
	base := url.URL{Scheme: resource.Scheme, Host: resource.Host}
	return base.String() + wellKnown + strings.TrimSuffix(resource.EscapedPath(), "/")
}

// requireToken is auth.RequireBearerToken with the RFC 6750 error codes in
// its challenge. The SDK names no error, and MCP clients start step-up
// authorization only on error="insufficient_scope". It adds that code to a
// 403 and error="invalid_token" to a 401 that carried a token.
func requireToken(verifier auth.TokenVerifier, opts *auth.RequireBearerTokenOptions) func(http.Handler) http.Handler {
	options := *opts
	if options.ClockSkew == 0 {
		options.ClockSkew = clockSkew
	}
	check := auth.RequireBearerToken(verifier, &options)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			passed := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { next.ServeHTTP(w, r) })
			check(passed).ServeHTTP(&challengeWriter{ResponseWriter: w, withToken: r.Header.Get("Authorization") != ""}, r)
		})
	}
}

// challengeWriter adds the error code to the SDK's challenge as the status
// is written.
type challengeWriter struct {
	http.ResponseWriter
	withToken bool
}

// WriteHeader adds the code that matches status, then writes it.
func (c *challengeWriter) WriteHeader(status int) {
	switch {
	case status == http.StatusForbidden:
		addErrorCode(c.Header(), "insufficient_scope")
	case status == http.StatusUnauthorized && c.withToken:
		addErrorCode(c.Header(), "invalid_token")
	}
	c.ResponseWriter.WriteHeader(status)
}

// addErrorCode puts error="code" first in the Bearer challenge.
func addErrorCode(header http.Header, code string) {
	param := `error="` + code + `"`
	challenges := header.Values("WWW-Authenticate")
	if len(challenges) == 0 {
		header.Set("WWW-Authenticate", "Bearer "+param)
		return
	}
	header.Set("WWW-Authenticate", strings.Replace(challenges[0], "Bearer ", "Bearer "+param+", ", 1))
	for _, rest := range challenges[1:] {
		header.Add("WWW-Authenticate", rest)
	}
}
