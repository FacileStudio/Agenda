package oidc

import (
	"encoding/base64"
	"encoding/json"
	"time"
)

// flowCookie is the name Nuage, Courrier and the rest already use for the
// state cookie. Keeping it means no third spelling to reconcile later.
const flowCookie = "oidc_state"

// flowTTL bounds how long a login may sit half-finished.
const flowTTL = 10 * time.Minute

// flow is everything the callback needs and the IdP must not see. It rides in
// one HttpOnly cookie rather than three: state, nonce and the PKCE verifier
// have the same lifetime, the same secrecy and the same failure mode, and one
// cookie cannot arrive partially.
type flow struct {
	State    string `json:"s"`
	Nonce    string `json:"n"`
	Verifier string `json:"v"`
	CLI      bool   `json:"c,omitempty"`
	Port     string `json:"p,omitempty"`

	// CLIState is the calling CLI's own nonce, echoed back on the loopback
	// redirect. It rides in this cookie rather than through the IdP because
	// the IdP has no business seeing it and would not return it anyway.
	CLIState string `json:"cs,omitempty"`
}

func (f flow) encode() (string, error) {
	payload, err := json.Marshal(f)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeFlow(value string) (flow, bool) {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return flow{}, false
	}
	var decoded flow
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return flow{}, false
	}
	return decoded, decoded.State != "" && decoded.Verifier != "" && decoded.Nonce != ""
}
