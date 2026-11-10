package mcpauth

import (
	"strconv"

	"github.com/FacileStudio/porte"
	"github.com/modelcontextprotocol/go-sdk/auth"
)

// The TokenInfo.Extra keys Verifier sets. Each holds a string, empty when the
// token carries no such claim.
const (
	// ExtraClientID is the client_id claim: the OAuth client the token was
	// issued to, the exchanging client on a delegated token.
	ExtraClientID = "client_id"

	// ExtraActor is act.sub, set only on a delegated (RFC 8693) token.
	ExtraActor = "actor"

	// ExtraSubject is the Registre sub of the person the token speaks for,
	// the subject forfait reads a personal plan under.
	ExtraSubject = "subject"
)

// BulleClientID is Bulle's client id at Registre. A token Bulle exchanged
// carries it as client_id and act.sub.
const BulleClientID = "bulle"

// ThirdParty reports whether the token belongs to a client other than Bulle.
// It is false only for a delegated token, act present, that Bulle obtained by
// exchange: Bulle has its own ai.access gate and keeps its existing path. A
// nil info is third party, so a gate never opens by accident.
func ThirdParty(info *auth.TokenInfo) bool {
	if info == nil {
		return true
	}
	return extra(info, ExtraActor) == "" || extra(info, ExtraClientID) != BulleClientID
}

// Identity rebuilds the porte identity Verifier resolved, so an app can attach
// it with porte.WithIdentity and reuse handlers written for the Authorization
// header path. The second result is false when info did not come from
// Verifier.
func Identity(info *auth.TokenInfo) (porte.Identity, bool) {
	if info == nil {
		return porte.Identity{}, false
	}
	userID, err := strconv.ParseInt(info.UserID, 10, 64)
	if err != nil || userID == 0 {
		return porte.Identity{}, false
	}
	return porte.Identity{
		UserID:  userID,
		Subject: extra(info, ExtraSubject),
		Scopes:  info.Scopes,
		Actor:   extra(info, ExtraActor),
	}, true
}

func extra(info *auth.TokenInfo, key string) string {
	value, _ := info.Extra[key].(string)
	return value
}
