package mcpauth

import (
	"encoding/json"
	"net/http"

	"github.com/FacileStudio/porte/forfait"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/modelcontextprotocol/go-sdk/auth"
)

// AIAccess is the entitlement a third-party MCP client needs: the same paid
// feature that turns Bulle's assistant on.
const AIAccess = "ai.access"

const planRefusal = "Third-party AI clients need a paid plan. Upgrade your plan to connect this client."

// AIAccessFeature declares AIAccess for forfait.NewStore. It defaults to off,
// the free plan.
func AIAccessFeature() forfait.Feature {
	return forfait.Feature{Key: AIAccess, Kind: forfait.Bool, Default: json.RawMessage("false")}
}

// Entitlements is the part of *forfait.Store the plan gate reads.
type Entitlements interface {
	Enabled(scope forfait.Scope, key string) bool
}

// CheckPlan answers nil when the token may use the MCP endpoint, and the
// suite's payment_required error, with an upgrade_url to Registre's plans
// page, when a third-party client acts for a person whose personal plan lacks
// ai.access. Bulle's exchanged tokens always pass. A nil plans refuses every
// third-party token. The plan is read under the token's Registre subject.
func CheckPlan(info *auth.TokenInfo, plans Entitlements, issuer string) error {
	if !ThirdParty(info) {
		return nil
	}
	if plans != nil && plans.Enabled(forfait.Scope{Subject: extra(info, ExtraSubject)}, AIAccess) {
		return nil
	}
	return errors.PaymentRequired(planRefusal).WithUpgradeURL(forfait.UpgradeURL(issuer, ""))
}

// PlanGate is middleware that runs CheckPlan after Require and answers its
// refusal as HTTP 402 with the suite error envelope. A request Require did not
// authenticate is a 401.
func PlanGate(plans Entitlements, issuer string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			info := auth.TokenInfoFromContext(r.Context())
			if info == nil {
				httpjson.WriteError(w, errors.Unauthorized("not authenticated"))
				return
			}
			if err := CheckPlan(info, plans, issuer); err != nil {
				httpjson.WriteError(w, err)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
