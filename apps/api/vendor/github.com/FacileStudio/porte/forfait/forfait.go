// Package forfait is how a Facile app enforces the plan a space pays for.
//
// Ardoise computes every entitlement and pushes it to Registre, which serves
// it inside the directory snapshot an app already syncs. An app never
// recomputes a plan: it declares the features it enforces, feeds every
// snapshot to Store.Apply, and asks Enabled, Limit or Check before creating
// something new. Reading, downloading and deleting are never gated, so a
// downgrade never locks anyone out of their own data.
//
// A Store fails open only until its first authoritative snapshot (one with a
// non-zero EntitlementsRevision) arrives: every declared feature is enabled
// and unlimited meanwhile, and a warning is logged once. That covers an app
// deployed before Registre serves entitlements. After it, an outage changes
// nothing: the Store keeps the last snapshot it applied for as long as
// Registre stays unreachable, and Export and Import carry it across restarts.
package forfait

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Kind is the shape of a feature's value.
type Kind int

const (
	// Bool is a feature that is on or off, such as "glouton.access".
	Bool Kind = iota + 1
	// Limit is an integer ceiling, such as "nuage.storage_gb", where -1
	// means unlimited.
	Limit
)

// Feature is one entitlement an app enforces. Key is the suite-wide name
// Ardoise uses, such as "courrier.mailboxes". Default is the JSON value that
// applies when a scope carries no value for the key, which is the free plan:
// `false` or `true` for a Bool, an integer or -1 for a Limit. A Default that
// is missing or malformed denies: false for a Bool, 0 for a Limit. A Key that
// does not match `^[a-z0-9_]+\.[a-z0-9_]+$` is never declared.
type Feature struct {
	Key     string
	Kind    Kind
	Default json.RawMessage
}

// Scope is what an entitlement is read for. SpaceID is the Registre space id
// a resource belongs to. An empty SpaceID is the personal scope of Subject,
// the Registre `sub` of the person who holds the resource outside any space.
//
// Both fields come from the app's own server-side state: the Registre id
// stored on the resource's space row and the subject of the authenticated
// identity, never a value the client sent. A client that could choose the
// SpaceID could pick the most generous plan it can name.
//
// An empty SpaceID always means the person, never "some space". An app passes
// the Registre id of the space a resource belongs to, and refuses, rather than
// falling back to the personal scope, a resource in one of its own spaces
// that is not linked to Registre yet: otherwise the owner's personal plan
// would silently govern a shared space.
type Scope struct {
	SpaceID string
	Subject string
}

// ErrLimit is what every refusal from Check matches with errors.Is. Apps map
// it to HTTP 402 with the suite error envelope and the LimitError's
// UpgradeURL.
var ErrLimit = errors.New("forfait: plan limit reached")

// ErrInvalidUsage is what Check returns for a negative used or adding count.
// It is a caller bug, not a plan refusal, so it does not match ErrLimit.
var ErrInvalidUsage = errors.New("forfait: negative usage count")

// LimitError is the refusal Check returns. Key is the feature, Limit the
// ceiling the scope's plan allows, and UpgradeURL the Registre page where an
// owner changes the plan, empty when the Store has no Issuer.
type LimitError struct {
	Key        string
	Limit      int64
	UpgradeURL string
}

// Error names the feature and the ceiling that refused the creation.
func (e *LimitError) Error() string {
	return fmt.Sprintf("%v: %s allows %d", ErrLimit, e.Key, e.Limit)
}

// PlanLimit names the feature that refused and where to upgrade, so tronc's
// mcp.Failure maps the refusal without app code.
func (e *LimitError) PlanLimit() (string, string) {
	return e.Key, e.UpgradeURL
}

// Is makes errors.Is(err, ErrLimit) true for every LimitError.
func (e *LimitError) Is(target error) bool {
	return target == ErrLimit
}

// UpgradeURL is the Registre page where an owner changes a space's plan:
// `<issuer>/spaces/<spaceID>`, or `<issuer>/spaces` for the personal scope
// (an empty spaceID). issuer is the app's OIDC issuer, Registre's origin.
func UpgradeURL(issuer, spaceID string) string {
	base := strings.TrimSuffix(issuer, "/") + "/spaces"
	if spaceID == "" {
		return base
	}
	return base + "/" + url.PathEscape(spaceID)
}
