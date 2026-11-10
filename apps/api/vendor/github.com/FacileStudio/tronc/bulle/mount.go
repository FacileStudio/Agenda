package bulle

import (
	"context"
	stderrors "errors"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"
)

var appName = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,31}$`)

// Options configures [MountIdeas]. App, URL and TokenSource are required.
type Options struct {
	// App is the app's name in Bulle's idea enum, such as agenda or opus.
	App         string
	URL         string
	TokenSource TokenSource
	Forget      func(ctx context.Context)
	HTTPClient  *http.Client
	// Space returns the request's active space: the Registre shared space
	// id, or "" for the personal space. Nil means always personal.
	Space func(r *http.Request) string
	// AIAccess reports whether the request's plan includes ai.access. When
	// it says no, the routes that spend AI answer 402 without calling Bulle
	// and the list answers ai_access false. Nil leaves the check to Bulle.
	AIAccess func(r *http.Request) bool
	// UpgradeURL goes with that 402, such as Registre's /spaces page.
	UpgradeURL string
}

// IdeasPage is GET /ideas: the plan state and, with AI access, the ideas.
type IdeasPage struct {
	AIAccess   bool   `json:"ai_access"`
	UpgradeURL string `json:"upgrade_url,omitempty"`
	IdeaList
}

type ideas struct {
	client *Client
	opts   Options
}

// MountIdeas serves the app's own frontend under /ideas on router, which is
// the router the app's API lives on (so the routes are /api/ideas...):
//
//	GET  /ideas                               the page: ai_access, ideas, generating
//	POST /ideas/generate                      a new batch (402 without AI access)
//	POST /ideas/{id}/run                      run an idea in the background (402 without AI access)
//	POST /ideas/{id}/dismiss                  hide an idea
//	GET  /ideas/runs/{run}                    a run's state, outputs and approvals
//	POST /ideas/runs/{run}/approvals/{call}   {"approve": bool}; approving needs AI access
//
// Mount it behind the app's authentication, so TokenSource finds the user:
//
//	r.Group(func(r chi.Router) { r.Use(requireAuth); err = bulle.MountIdeas(r, opts) })
func MountIdeas(router chi.Router, opts Options) error {
	if !appName.MatchString(opts.App) {
		return stderrors.New("bulle: App must be the app's name in Bulle, such as agenda")
	}
	client, err := New(Config{URL: opts.URL, TokenSource: opts.TokenSource, Forget: opts.Forget, HTTPClient: opts.HTTPClient})
	if err != nil {
		return err
	}
	h := &ideas{client: client, opts: opts}
	router.Route("/ideas", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/generate", h.generate)
		r.Post("/{id}/run", h.run)
		r.Post("/{id}/dismiss", h.dismiss)
		r.Get("/runs/{run}", h.status)
		r.Post("/runs/{run}/approvals/{call}", h.decide)
	})
	return nil
}

func (h *ideas) space(r *http.Request) string {
	if h.opts.Space == nil {
		return ""
	}
	return h.opts.Space(r)
}

func (h *ideas) aiAccess(r *http.Request) bool {
	return h.opts.AIAccess == nil || h.opts.AIAccess(r)
}

func (h *ideas) requireAI(w http.ResponseWriter, r *http.Request) bool {
	if h.aiAccess(r) {
		return true
	}
	plan := &PlanRequired{Message: "Ideas need a plan with AI access.", UpgradeURL: h.opts.UpgradeURL, Key: "ai.access"}
	writeError(w, plan)
	return false
}
