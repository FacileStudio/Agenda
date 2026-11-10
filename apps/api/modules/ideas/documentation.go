package ideas

import (
	documentation "github.com/FacileStudio/Agenda/apps/api/internal/documentation"
	"github.com/FacileStudio/tronc/bulle"
)

var errorsDoc = []documentation.Error{
	{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."},
	{Status: 402, Code: "payment_required", Description: "The plan has no ai.access. The error carries an upgrade_url."},
	{Status: 404, Code: "not_found", Description: "The space is unknown or the user is not a member of it."},
}

var spaceParam = documentation.Field{Name: "space_id", Type: "string", Description: "Agenda space id. Absent means personal."}

// Documentation describes the Ideas routes for the API reference.
var Documentation = documentation.Module{
	Name:        "ideas",
	Description: "AI suggestions for Agenda, generated and run by Bulle. Mounted only when BULLE_URL is set.",
	Routes: []documentation.Route{
		{
			Method:       "GET",
			Path:         "/ideas",
			Summary:      "List ideas",
			Description:  "Returns ai_access false and no ideas when the plan lacks ai.access, with the upgrade_url.",
			Auth:         "session cookie or bearer token",
			QueryParams:  []documentation.Field{spaceParam},
			ResponseBody: bulle.IdeasPage{},
			Errors:       errorsDoc,
		},
		{
			Method:       "POST",
			Path:         "/ideas/generate",
			Summary:      "Generate ideas",
			Description:  "Starts a batch in Bulle scoped to Agenda.",
			Auth:         "session cookie or bearer token",
			QueryParams:  []documentation.Field{spaceParam},
			ResponseBody: bulle.Batch{},
			Errors:       errorsDoc,
		},
		{
			Method:       "POST",
			Path:         "/ideas/{id}/run",
			Summary:      "Run an idea",
			Description:  "Runs the idea in the background in Bulle and returns the run id to poll.",
			Auth:         "session cookie or bearer token",
			QueryParams:  []documentation.Field{spaceParam},
			ResponseBody: bulle.RunStart{},
			Errors:       errorsDoc,
		},
		{
			Method:  "POST",
			Path:    "/ideas/{id}/dismiss",
			Summary: "Dismiss an idea",
			Auth:    "session cookie or bearer token",
			Errors:  errorsDoc,
		},
		{
			Method:       "GET",
			Path:         "/ideas/runs/{run}",
			Summary:      "Read a run",
			Description:  "State, summary, output links, pending approvals and the link that continues the run in Bulle.",
			Auth:         "session cookie or bearer token",
			ResponseBody: bulle.Run{},
			Errors:       errorsDoc,
		},
		{
			Method:      "POST",
			Path:        "/ideas/runs/{run}/approvals/{call}",
			Summary:     "Answer an approval",
			Description: "Body {\"approve\": bool}. Approving needs ai.access.",
			Auth:        "session cookie or bearer token",
			Errors:      errorsDoc,
		},
	},
}
