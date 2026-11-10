package mcp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/tronc/errors"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ConfirmInput is the key of the one input request a confirmation makes.
const ConfirmInput = "confirm"

// mrtrVersion is the first protocol version with multi round-trip requests.
// Older clients reach a stateless server without a session, so the SDK's
// fallback, a server-to-client elicitation, cannot run for them.
const mrtrVersion = "2026-07-28"

// verdict is what the person answered to a confirmation.
type verdict int

const (
	accepted verdict = iota + 1
	declined
)

// ErrConfirmation is the refusal for a confirmation state this server did
// not sign, signed for someone else, for another tool, or too long ago.
var ErrConfirmation = errors.Failed("this confirmation is no longer valid. Call the tool again to ask the user afresh")

// confirmSchema is the form the client shows: a single checkbox, ticked.
var confirmSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		ConfirmInput: map[string]any{"type": "boolean", "title": "Confirm", "default": true},
	},
	"required": []string{ConfirmInput},
}

// CanElicit reports whether the client asked for form elicitation on a
// protocol version that lets a stateless server use it. An elicitation
// capability naming no mode means form.
func CanElicit(req *sdk.CallToolRequest) bool {
	if req.ProtocolVersion() < mrtrVersion {
		return false
	}
	caps := req.ClientCapabilities()
	if caps == nil || caps.Elicitation == nil {
		return false
	}
	return caps.Elicitation.Form != nil || caps.Elicitation.URL == nil
}

// Retrying reports whether a call is the retry of an input_required
// result. A sensitive tool checks it first and hands the call to
// Confirmations.Resume.
func Retrying(req *sdk.CallToolRequest) bool {
	return req.Params.RequestState != "" || len(req.Params.InputResponses) > 0
}

// signer makes and checks the request state of a confirmation: the pending
// action, the person and an expiry, signed with HMAC-SHA256, so a client
// cannot forge a confirmation for an action the server did not ask about.
type signer struct {
	key []byte
	ttl time.Duration
}

// ask returns the input_required result for pending action id.
func (s signer) ask(userID, id, message string) *sdk.CallToolResult {
	return &sdk.CallToolResult{
		InputRequests: sdk.InputRequestMap{ConfirmInput: &sdk.ElicitParams{
			Mode: "form", Message: message, RequestedSchema: confirmSchema,
		}},
		RequestState: s.sign(userID, id, time.Now().Add(s.ttl)),
	}
}

// answer reads the retry of an ask: the pending action it was about and
// whether the person accepted.
func (s signer) answer(req *sdk.CallToolRequest, userID string) (string, verdict, error) {
	id, err := s.verify(req.Params.RequestState, userID, time.Now())
	if err != nil {
		return "", 0, err
	}
	result, ok := req.Params.InputResponses[ConfirmInput].(*sdk.ElicitResult)
	if !ok {
		return "", 0, ErrConfirmation
	}
	if result.Action != "accept" || result.Content[ConfirmInput] == false {
		return id, declined, nil
	}
	return id, accepted, nil
}

// sign makes the request state for pending action id. Ids and user ids hold
// no dots: pending ids are base64url, user ids are the app's own.
func (s signer) sign(userID, id string, expires time.Time) string {
	payload := userID + "." + id + "." + strconv.FormatInt(expires.Unix(), 10)
	return payload + "." + s.mac(payload)
}

// verify returns the pending action id a state names, if the state is this
// server's, for userID, and not expired at now.
func (s signer) verify(state, userID string, now time.Time) (string, error) {
	parts := strings.Split(state, ".")
	if len(parts) != 4 || parts[0] != userID || userID == "" {
		return "", ErrConfirmation
	}
	payload := strings.Join(parts[:3], ".")
	if !hmac.Equal([]byte(parts[3]), []byte(s.mac(payload))) {
		return "", ErrConfirmation
	}
	expires, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || now.Unix() > expires {
		return "", ErrConfirmation
	}
	return parts[1], nil
}

// mac is the hex HMAC-SHA256 of payload.
func (s signer) mac(payload string) string {
	sum := hmac.New(sha256.New, s.key)
	sum.Write([]byte(payload))
	return hex.EncodeToString(sum.Sum(nil))
}
