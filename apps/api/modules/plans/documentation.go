package plans

import documentation "github.com/FacileStudio/Agenda/apps/api/internal/documentation"

var (
	unauthenticated = documentation.Error{Status: 401, Code: "unauthenticated", Description: "Authorization header is missing or invalid."}
	internalError   = documentation.Error{Status: 500, Code: "internal", Description: "Unexpected server error."}
	notFound        = documentation.Error{Status: 404, Code: "not_found", Description: "No plan with that ID, or it belongs to someone else."}
	planParam       = []documentation.Field{{Name: "planID", Type: "string", Description: "Plan ID."}}
)

// Documentation describes the plan routes for the API reference.
var Documentation = documentation.Module{
	Name: "plans",
	Description: "Draft schedules a model proposed, such as Bulle planning Opus tasks. A plan holds blocks of " +
		"time with a reason each. Proposing one, through the MCP tool agenda_propose_plan, changes no calendar: " +
		"the person applies all blocks, some, or none. A proposed plan expires after 7 days. Titles and reasons " +
		"are plain text written by a model: show them as text, never interpret them. Mutations by cookie need " +
		"the X-Facile-CSRF header.",
	Routes: []documentation.Route{
		{
			Method:  "GET",
			Path:    "/plans",
			Summary: "List plans",
			Description: "The caller's plans that have not expired, newest first. `state` keeps proposed, applied or " +
				"dismissed; `limit` is 1 to 50, default 20.",
			Auth: "session cookie or bearer token",
			QueryParams: []documentation.Field{
				{Name: "state", Type: "string", Description: "proposed, applied or dismissed. Absent means all."},
				{Name: "limit", Type: "integer", Description: "Plans to return, 1 to 50. Default 20."},
			},
			ResponseBody: []PlanResponse{},
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "Unknown state."},
				unauthenticated, internalError,
			},
		},
		{
			Method:       "GET",
			Path:         "/plans/{planID}",
			Summary:      "Return one plan",
			Auth:         "session cookie or bearer token",
			PathParams:   planParam,
			ResponseBody: PlanResponse{},
			Errors:       []documentation.Error{unauthenticated, notFound, internalError},
		},
		{
			Method:  "POST",
			Path:    "/plans/{planID}/apply",
			Summary: "Apply a plan",
			Description: "Creates one event per chosen block through the events service, in one transaction, and " +
				"marks the plan applied. `block_ids` picks a subset; an empty or absent body applies every block. " +
				"Each event's description holds the task link and the reason. A block that now overlaps an event " +
				"is refused with 409 and nothing is created, unless `force` is true.",
			Auth:         "session cookie or bearer token",
			PathParams:   planParam,
			RequestBody:  ApplyRequest{},
			ResponseBody: ApplyResponse{},
			Errors: []documentation.Error{
				{Status: 400, Code: "invalid_argument", Description: "A block id is not in the plan."},
				unauthenticated,
				{Status: 403, Code: "permission_denied", Description: "A block's calendar is no longer writable by the caller."},
				notFound,
				{Status: 409, Code: "already_exists", Description: "A block overlaps an event now. Retry with force, or leave it out."},
				{Status: 412, Code: "failed_precondition", Description: "The plan is already applied or dismissed, expired, or a block starts in the past."},
				internalError,
			},
		},
		{
			Method:       "POST",
			Path:         "/plans/{planID}/dismiss",
			Summary:      "Dismiss a plan",
			Description:  "Closes a proposed plan without touching the calendar. Dismissing a dismissed plan succeeds.",
			Auth:         "session cookie or bearer token",
			PathParams:   planParam,
			ResponseBody: PlanResponse{},
			Errors: []documentation.Error{
				unauthenticated, notFound,
				{Status: 412, Code: "failed_precondition", Description: "The plan is already applied."},
				internalError,
			},
		},
	},
}
