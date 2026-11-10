package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// rpcCall is the part of a JSON-RPC message the write gate reads.
type rpcCall struct {
	Method string `json:"method"`
	Params struct {
		Name string `json:"name"`
	} `json:"params"`
}

// writeGate answers 403 insufficient_scope when a token without the write
// scope calls a write tool, so the client can ask the person for more access
// instead of failing inside the tool. It reads the body rather than the
// Mcp-Name header, because the body is what the transport acts on.
func writeGate(writes map[string]bool, scope, metadataURL string) func(http.Handler) http.Handler {
	challenge := fmt.Sprintf(`Bearer error="insufficient_scope", scope=%q, resource_metadata=%q`, scope, metadataURL)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || grants(r, scope) {
				next.ServeHTTP(w, r)
				return
			}
			body, ok := rereadable(w, r)
			if !ok {
				return
			}
			if callsAny(body, writes) {
				w.Header().Set("WWW-Authenticate", challenge)
				http.Error(w, "Forbidden: this tool needs the "+scope+" scope", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// grants reports whether the request's verified token holds scope.
func grants(r *http.Request, scope string) bool {
	info := auth.TokenInfoFromContext(r.Context())
	return info != nil && slices.Contains(info.Scopes, scope)
}

// rereadable reads the request body, capped at the transport's limit, and
// puts it back for the next handler. It answers 413 and false when the body
// is too large.
func rereadable(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, sdk.DefaultMaxRequestBodyBytes))
	if err != nil {
		http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
		return nil, false
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, true
}

// callsAny reports whether a JSON-RPC message, or any message of a batch,
// calls one of the named tools.
func callsAny(body []byte, names map[string]bool) bool {
	var calls []rpcCall
	if err := json.Unmarshal(body, &calls); err != nil {
		var call rpcCall
		if json.Unmarshal(body, &call) != nil {
			return false
		}
		calls = []rpcCall{call}
	}
	return slices.ContainsFunc(calls, func(call rpcCall) bool {
		return call.Method == "tools/call" && names[call.Params.Name]
	})
}

// originGuard refuses a request whose Origin names a site that is not
// allowed, which stops a web page from driving the endpoint through DNS
// rebinding. Requests without Origin, from native clients, pass.
func originGuard(allowed []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin := r.Header.Get("Origin"); origin != "" && !slices.Contains(allowed, origin) {
				http.Error(w, "Forbidden: origin not allowed", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
