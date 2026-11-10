package mcpauth

import (
	"net/http"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
)

// clockSkew matches porte/oidc/jwt's leeway, so a token jwt accepted is not
// refused a second later by the SDK's own expiry check.
const clockSkew = 30 * time.Second

// Require is auth.RequireBearerToken with the RFC 6750 error codes added to
// its challenge. The SDK writes `WWW-Authenticate: Bearer resource_metadata=…,
// scope=…` on both refusals but names no error, and MCP clients, the SDK's own
// included, start step-up authorization only on `error="insufficient_scope"`.
// Require adds that code to a 403 and `error="invalid_token"` to a 401 that
// carried a token. A zero opts.ClockSkew becomes 30 s. The wrapped handler
// gets the original ResponseWriter, so its own responses are left alone.
func Require(verifier auth.TokenVerifier, opts *auth.RequireBearerTokenOptions) func(http.Handler) http.Handler {
	options := auth.RequireBearerTokenOptions{}
	if opts != nil {
		options = *opts
	}
	if options.ClockSkew == 0 {
		options.ClockSkew = clockSkew
	}
	middleware := auth.RequireBearerToken(verifier, &options)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			passed := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { next.ServeHTTP(w, r) })
			challenge := &challengeWriter{ResponseWriter: w, withToken: r.Header.Get("Authorization") != ""}
			middleware(passed).ServeHTTP(challenge, r)
		})
	}
}

// challengeWriter adds the error code to the SDK's challenge as the status is
// written.
type challengeWriter struct {
	http.ResponseWriter
	withToken bool
}

func (c *challengeWriter) WriteHeader(status int) {
	switch {
	case status == http.StatusForbidden:
		addErrorCode(c.Header(), "insufficient_scope")
	case status == http.StatusUnauthorized && c.withToken:
		addErrorCode(c.Header(), "invalid_token")
	}
	c.ResponseWriter.WriteHeader(status)
}

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
