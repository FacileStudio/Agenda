// Package apiref serves a Facile API's reference documentation.
//
// Every Facile backend keeps a hand-written route registry. apiref turns that
// registry into an OpenAPI 3.1 document and serves it behind a Scalar UI at one
// suite-wide location, so every app answers on the same path:
//
//	apiref.Mount(router, apiref.Config{
//		Title:       "Sablier API",
//		Description: "Self-hosted time tracker for small teams.",
//		Servers:     []string{"/api"},
//		Registry:    docs.Registry,
//	})
//
// That registers GET /docs and GET /docs/openapi.json. Mount it on the root
// router — beside /api and before the SPA catch-all — so the reference sits at
// /docs rather than behind the API prefix.
//
// The Scalar bundle is loaded from a pinned CDN build. That is one remote
// request on a developer-facing page; an air-gapped deployment should set
// ScriptURL to a locally served copy.
package apiref

import (
	"html/template"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/FacileStudio/tronc/httpjson"
)

// ScalarScriptURL is the pinned Scalar standalone bundle. Pinned rather than
// floating so a Scalar release cannot change every suite app's reference page
// without a commit.
const ScalarScriptURL = "https://cdn.jsdelivr.net/npm/@scalar/api-reference@1.64.0"

// BasePath is where the reference is mounted, relative to the router passed to
// Mount. Mounting on the root router therefore serves /docs.
const BasePath = "/docs"

// SpecPath is the OpenAPI document, relative to the router passed to Mount.
const SpecPath = BasePath + "/openapi.json"

// Config describes one API's reference page.
type Config struct {
	// Title names the API, e.g. "Sablier API".
	Title string
	// Version is the API version. Defaults to DefaultVersion.
	Version string
	// Description is one or two sentences shown above the route list.
	Description string
	// Servers are the base URLs every documented path is relative to, usually
	// []string{"/api"}. Omitted from the document when empty.
	Servers []string
	// Registry is the route inventory the document is generated from.
	Registry Registry
	// ScriptURL overrides the Scalar bundle. Defaults to ScalarScriptURL.
	ScriptURL string
}

func (c Config) withDefaults() Config {
	if c.Version == "" {
		c.Version = DefaultVersion
	}
	if c.ScriptURL == "" {
		c.ScriptURL = ScalarScriptURL
	}
	if c.Title == "" {
		c.Title = "API"
	}
	return c
}

// Mount registers the reference page and the OpenAPI document.
func Mount(router chi.Router, cfg Config) {
	cfg = cfg.withDefaults()
	document := OpenAPI(cfg)

	router.Get(BasePath, func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page(cfg, request.URL.Path+"/openapi.json")))
	})

	router.Get(SpecPath, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600")
		httpjson.WriteJSON(w, http.StatusOK, document)
	})
}

var pageTemplate = template.Must(template.New("apiref").Parse(`<!doctype html>
<html>
<head>
  <title>{{.Title}}</title>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <style>body { margin: 0; }</style>
</head>
<body>
  <script id="api-reference" data-url="{{.SpecURL}}"></script>
  <script src="{{.ScriptURL}}"></script>
</body>
</html>`))

func page(cfg Config, specURL string) string {
	var out strings.Builder
	_ = pageTemplate.Execute(&out, struct {
		Title     string
		SpecURL   string
		ScriptURL template.URL
	}{cfg.Title, specURL, template.URL(cfg.ScriptURL)})
	return out.String()
}
