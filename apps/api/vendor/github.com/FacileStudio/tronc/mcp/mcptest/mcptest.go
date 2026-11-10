// Package mcptest is the conformance kit for servers built on tronc/mcp.
// Run checks a live endpoint the way MCP clients and the suite expect it:
// the 401 challenge naming the RFC 9728 metadata, the metadata itself, 405
// on GET, the Origin check, explicit annotations, valid schemas, a stable
// sorted tool list, the 403 write-scope challenge, and for each sensitive
// tool the whole confirmation path.
//
// Issuer and Pending are the fakes an app's tests mount the server on.
package mcptest

import (
	"encoding/json"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// listTools is a JSON-RPC tools/list request.
const listTools = `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`

// Config points Run at a running endpoint.
type Config struct {
	// URL is the resource URI the endpoint serves.
	URL string

	// Token mints a valid access token for URL with scopes, or with every
	// scope the metadata lists when none is given.
	Token func(t testing.TB, scopes ...string) string

	// Confirm lists the sensitive tools and how to set up a fresh target for
	// each run.
	Confirm []ConfirmCase
}

// ConfirmCase is one sensitive tool under test.
type ConfirmCase struct {
	// Tool is the tool name.
	Tool string

	// Setup creates a fresh target and returns the tool arguments for it
	// plus a counter of how many times the action really ran on it.
	Setup func(t testing.TB) (args map[string]any, runs func() int)
}

// metadata is the part of the RFC 9728 document Run checks.
type metadata struct {
	Resource             string   `json:"resource"`
	AuthorizationServers []string `json:"authorization_servers"`
	ScopesSupported      []string `json:"scopes_supported"`
}

// Run checks the endpoint at cfg.URL.
func Run(t *testing.T, cfg Config) {
	t.Helper()
	meta := checkChallenge(t, cfg)
	if t.Failed() || len(meta.ScopesSupported) != 2 {
		t.Fatalf("metadata must list a read and a write scope, got %v", meta.ScopesSupported)
	}
	read, write := meta.ScopesSupported[0], meta.ScopesSupported[1]
	prefix, _, _ := strings.Cut(read, ".")
	t.Run("get is not allowed", func(t *testing.T) { checkGet(t, cfg) })
	t.Run("foreign origin is refused", func(t *testing.T) { checkOrigin(t, cfg) })
	tools := listed(t, cfg)
	t.Run("tools", func(t *testing.T) { checkTools(t, tools, prefix+"_") })
	t.Run("write scope", func(t *testing.T) { checkWriteScope(t, cfg, tools, read, write) })
	for _, c := range cfg.Confirm {
		t.Run("confirm "+c.Tool, func(t *testing.T) { checkConfirm(t, cfg, c) })
	}
}

// checkChallenge checks that a call without a token gets a 401 naming the
// metadata, and that the metadata names the resource and its issuer.
func checkChallenge(t *testing.T, cfg Config) metadata {
	t.Helper()
	resp := post(t, cfg.URL, "", listTools, nil)
	challenge := resp.Header.Get("WWW-Authenticate")
	metadataURL := metadataURLOf(t, cfg.URL)
	if resp.StatusCode != http.StatusUnauthorized || !strings.Contains(challenge, `resource_metadata="`+metadataURL+`"`) {
		t.Errorf("no token: want 401 with resource_metadata=%q, got %d %q", metadataURL, resp.StatusCode, challenge)
	}
	var meta metadata
	doc, err := http.Get(metadataURL)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = doc.Body.Close() }()
	if doc.StatusCode != http.StatusOK || json.NewDecoder(doc.Body).Decode(&meta) != nil {
		t.Fatalf("metadata: want a JSON document at %s, got %d", metadataURL, doc.StatusCode)
	}
	if meta.Resource != cfg.URL || len(meta.AuthorizationServers) == 0 {
		t.Errorf("metadata must name resource %s and an authorization server, got %+v", cfg.URL, meta)
	}
	return meta
}

// checkGet checks that the stateless endpoint refuses GET.
func checkGet(t *testing.T, cfg Config) {
	req, _ := http.NewRequest(http.MethodGet, cfg.URL, nil)
	req.Header.Set("Authorization", "Bearer "+cfg.Token(t))
	req.Header.Set("Accept", "text/event-stream")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("want 405 on GET, got %d", resp.StatusCode)
	}
}

// checkOrigin checks the DNS rebinding guard and that the endpoint's own
// origin passes it.
func checkOrigin(t *testing.T, cfg Config) {
	if resp := post(t, cfg.URL, cfg.Token(t), listTools, map[string]string{"Origin": "https://evil.example"}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("want 403 for a foreign origin, got %d", resp.StatusCode)
	}
	own, _ := url.Parse(cfg.URL)
	if resp := post(t, cfg.URL, cfg.Token(t), listTools, map[string]string{"Origin": own.Scheme + "://" + own.Host}); resp.StatusCode != http.StatusOK {
		t.Errorf("the endpoint's own origin must pass, got %d", resp.StatusCode)
	}
}

// listed lists the tools twice and checks the lists are the same, sorted by
// name, and cacheable only privately.
func listed(t *testing.T, cfg Config) []*sdk.Tool {
	t.Helper()
	session := Connect(t, cfg.URL, cfg.Token(t), ClientOptions{})
	first, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	again, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	names := toolNames(first.Tools)
	if !slices.Equal(names, toolNames(again.Tools)) || !slices.IsSorted(names) || len(names) == 0 {
		t.Errorf("tools/list must be non-empty, stable and sorted by name, got %v then %v", names, toolNames(again.Tools))
	}
	if first.CacheScope != "private" {
		t.Errorf("tools/list must be cacheable privately only, got cacheScope %q", first.CacheScope)
	}
	return first.Tools
}

// checkTools checks names, descriptions, annotations and schemas.
func checkTools(t *testing.T, tools []*sdk.Tool, prefix string) {
	for _, tool := range tools {
		a := tool.Annotations
		switch {
		case !strings.HasPrefix(tool.Name, prefix):
			t.Errorf("%s: name must start with %s", tool.Name, prefix)
		case tool.Description == "" || tool.Title == "":
			t.Errorf("%s: needs a title and a description", tool.Name)
		case a == nil || a.DestructiveHint == nil || a.OpenWorldHint == nil || a.Title == "":
			t.Errorf("%s: annotations must set every hint and a title, got %+v", tool.Name, a)
		case a.ReadOnlyHint && *a.DestructiveHint:
			t.Errorf("%s: a read-only tool cannot be destructive", tool.Name)
		}
		checkSchema(t, tool.Name+" input", tool.InputSchema)
		if tool.OutputSchema != nil {
			checkSchema(t, tool.Name+" output", tool.OutputSchema)
		}
	}
}

// checkSchema checks a tool schema is an object schema that resolves.
func checkSchema(t *testing.T, label string, schema any) {
	raw, err := json.Marshal(schema)
	var parsed jsonschema.Schema
	if err == nil {
		err = json.Unmarshal(raw, &parsed)
	}
	if err == nil {
		_, err = parsed.Resolve(nil)
	}
	if err != nil || parsed.Type != "object" {
		t.Errorf("%s: want a valid object schema, got %s (%v)", label, raw, err)
	}
}

// checkWriteScope checks a read-only token gets a 403 step-up challenge on a
// write tool and can still list tools.
func checkWriteScope(t *testing.T, cfg Config, tools []*sdk.Tool, read, write string) {
	index := slices.IndexFunc(tools, func(tool *sdk.Tool) bool { return !tool.Annotations.ReadOnlyHint })
	if index < 0 {
		t.Skip("no write tool")
	}
	token := cfg.Token(t, read)
	call := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"` + tools[index].Name + `","arguments":{}}}`
	resp := post(t, cfg.URL, token, call, nil)
	challenge := resp.Header.Get("WWW-Authenticate")
	if resp.StatusCode != http.StatusForbidden || !strings.Contains(challenge, `error="insufficient_scope"`) || !strings.Contains(challenge, `scope="`+write+`"`) {
		t.Errorf("read-only token on %s: want 403 insufficient_scope for %s, got %d %q", tools[index].Name, write, resp.StatusCode, challenge)
	}
	if resp := post(t, cfg.URL, token, listTools, nil); resp.StatusCode != http.StatusOK {
		t.Errorf("a read-only token must list tools, got %d", resp.StatusCode)
	}
}
