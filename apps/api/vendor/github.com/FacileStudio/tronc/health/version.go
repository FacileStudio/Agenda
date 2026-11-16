package health

import (
	"net/http"

	"github.com/FacileStudio/tronc/httpjson"
)

// Version handles GET /version: {"version": "<release>"}. It reads no state
// and names nothing but the release, so it is public by design.
//
// The release string belongs to the binary, so each app keeps one variable in
// package main for the linker to set (-ldflags "-X main.version=v1.2.3") and
// hands it in:
//
//	var version = "dev"
//
//	router.Get("/version", health.Version(version))
func Version(version string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		httpjson.WriteJSON(w, http.StatusOK, map[string]string{"version": version})
	}
}
