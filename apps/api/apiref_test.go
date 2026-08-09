package main

import (
	"testing"

	"github.com/FacileStudio/tronc/apiref"
)

// TestEveryRouteIsDocumented is the drift guard: it walks the router the binary
// actually serves and fails when a route is missing from the reference
// registry, so the two cannot quietly disagree.
//
// CalDAV, the avatar file server and the legacy OIDC callback are ignored. Their
// URLs are held by external clients, stored rows and Authentik rather than by
// this API's own contract, and CalDAV speaks WebDAV verbs OpenAPI cannot express.
func TestEveryRouteIsDocumented(t *testing.T) {
	// testRouter and not an empty mounts: porte owns /auth/config,
	// /auth/logout and the OIDC flow now, so a router without it would let
	// this guard pass over the routes most likely to move.
	router := testRouter(t)

	missing := apiref.Undocumented(router, referenceConfig(), "/dav", "/.well-known", "/files", "/auth/oidc/callback")
	if len(missing) > 0 {
		t.Errorf("routes missing from the API registry: %v", missing)
	}
}
