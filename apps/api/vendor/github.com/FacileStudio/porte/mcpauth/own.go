package mcpauth

import (
	"context"
	stderrors "errors"
	"fmt"
	"net/http"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/keys"
	"github.com/FacileStudio/tronc/errors"
	"github.com/modelcontextprotocol/go-sdk/auth"
)

// ErrNotOwnSession is the refusal for a confirm page request that does not
// come from the person's own browser session. It is a 403 with the same text
// as tronc's mcp.ErrDelegated, so a page answers alike whichever it returns.
var ErrNotOwnSession = errors.Forbidden("only the person's own session can confirm actions")

// ErrNoAccount is AccountExists' refusal: the token did not come from
// Verifier, or its account is gone. The lookup's error, when there is one, is
// wrapped with it.
var ErrNoAccount = stderrors.New("porte/mcpauth: the token names no account")

var errNoIdentity = errors.Unauthorized("missing auth")

// RequireOwnSession returns the person behind a confirm page request. It
// passes only porte.Identity.BrowserSession: the session cookie carrying a
// token a browser login minted (session.Manager.IssueCookieFor, the OIDC web
// callback, local login and magic link). It refuses a bearer header of any
// kind (an opaque session, a named API token, a JWT, delegated or not), a
// named API token or a CLI, device or mobile session token sent as a cookie,
// a machine token, and a request a porte/keys API key authenticated. A cookie
// issued before porte v0.20.1 carries no browser mark, so it is refused until
// the person signs in again. A request with no porte identity is a 401; every
// other refusal is ErrNotOwnSession, decided on every condition at once, so
// the answer and its timing say nothing about which one failed.
func RequireOwnSession(r *http.Request) (porte.Identity, error) {
	identity, ok := porte.From(r.Context())
	if !ok {
		return porte.Identity{}, errNoIdentity
	}
	_, apiKey := keys.From(r.Context())
	own := identity.BrowserSession && identity.SessionID != 0 && identity.UserID != 0
	delegated := identity.Actor != "" || identity.Subject != "" || len(identity.Scopes) != 0
	bearer := r.Header.Get("Authorization") != "" || apiKey
	if !own || delegated || bearer {
		return porte.Identity{}, ErrNotOwnSession
	}
	return identity, nil
}

// AccountExists reads the account a Verifier token speaks for with lookup,
// such as an app store's Identity method, so a tool never runs for an account
// erased or deleted after the token was minted. It answers ErrNoAccount when
// info did not come from Verifier or lookup fails.
func AccountExists[T any](ctx context.Context, info *auth.TokenInfo, lookup func(context.Context, int64) (T, error)) (T, error) {
	var none T
	identity, ok := Identity(info)
	if !ok {
		return none, ErrNoAccount
	}
	account, err := lookup(ctx, identity.UserID)
	if err != nil {
		return none, fmt.Errorf("%w: %w", ErrNoAccount, err)
	}
	return account, nil
}
