package mcptest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ClientOptions shape a test client.
type ClientOptions struct {
	// Elicit answers confirmation questions. Nil means the client does not
	// advertise elicitation.
	Elicit func(context.Context, *sdk.ElicitRequest) (*sdk.ElicitResult, error)

	// Version is the protocol version to ask for. Empty means the latest.
	Version string

	// Manual turns off the client's automatic MRTR retry, so a test can
	// answer input_required results itself.
	Manual bool
}

// bearer adds an access token to every request.
type bearer string

// RoundTrip sends r with the token.
func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+string(b))
	return http.DefaultTransport.RoundTrip(r)
}

// Connect opens an SDK client session on endpoint with token, closed with
// the test.
func Connect(t testing.TB, endpoint, token string, opts ClientOptions) *sdk.ClientSession {
	t.Helper()
	client := sdk.NewClient(&sdk.Implementation{Name: "mcptest", Version: "1"}, &sdk.ClientOptions{
		ElicitationHandler: opts.Elicit,
		MultiRoundTrip:     &sdk.MultiRoundTripOptions{Disabled: opts.Manual},
	})
	transport := &sdk.StreamableClientTransport{Endpoint: endpoint, HTTPClient: &http.Client{Transport: bearer(token)}}
	session, err := client.Connect(context.Background(), transport, &sdk.ClientSessionOptions{ProtocolVersion: opts.Version})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// Call runs one tool and fails the test on a protocol error.
func Call(t testing.TB, session *sdk.ClientSession, name string, args any) *sdk.CallToolResult {
	t.Helper()
	result, err := session.CallTool(context.Background(), &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return result
}

// Decode reads a result's structured content into out, failing the test on
// an error result.
func Decode(t testing.TB, result *sdk.CallToolResult, out any) {
	t.Helper()
	if result.IsError {
		t.Fatalf("tool failed: %s", Text(result))
	}
	raw, err := json.Marshal(result.StructuredContent)
	if err == nil {
		err = json.Unmarshal(raw, out)
	}
	if err != nil {
		t.Fatal(err)
	}
}

// Text joins a result's text content.
func Text(result *sdk.CallToolResult) string {
	var parts []string
	for _, content := range result.Content {
		if text, ok := content.(*sdk.TextContent); ok {
			parts = append(parts, text.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// Answering returns an elicitation handler that answers action ("accept" or
// "decline") and counts the questions in asked.
func Answering(action string, asked *int) func(context.Context, *sdk.ElicitRequest) (*sdk.ElicitResult, error) {
	return func(context.Context, *sdk.ElicitRequest) (*sdk.ElicitResult, error) {
		*asked++
		return &sdk.ElicitResult{Action: action, Content: map[string]any{"confirm": action == "accept"}}, nil
	}
}

// reply is the status and headers of a raw request.
type reply struct {
	StatusCode int
	Header     http.Header
}

// post sends one raw JSON-RPC body and returns its status and headers.
func post(t testing.TB, endpoint, token, body string, header map[string]string) reply {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for name, value := range header {
		req.Header.Set(name, value)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return reply{StatusCode: resp.StatusCode, Header: resp.Header}
}

// toolNames lists the names of tools in order.
func toolNames(tools []*sdk.Tool) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}
	return names
}

// metadataURLOf is the RFC 9728 location of resource's metadata.
func metadataURLOf(t *testing.T, resource string) string {
	parsed, err := url.Parse(resource)
	if err != nil {
		t.Fatal(err)
	}
	return parsed.Scheme + "://" + parsed.Host + "/.well-known/oauth-protected-resource" + strings.TrimSuffix(parsed.EscapedPath(), "/")
}
