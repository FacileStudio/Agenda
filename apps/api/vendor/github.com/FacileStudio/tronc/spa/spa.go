// Package spa serves a built single-page application from a directory, so one
// Go binary can serve both the API and the client it belongs to.
//
// Mount it last, as the catch-all, with the API already registered under /api:
//
//	router := httpx.NewRouter(httpx.Config{Logger: log})
//	router.Route("/api", func(r chi.Router) { /* ... */ })
//	router.Handle("/*", spa.Handler(spa.Config{Dir: spa.DirFromEnv()}))
package spa

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// DefaultDir is where the container image places the built client.
const DefaultDir = "./client"

// ImmutablePrefixes are URL prefixes whose contents carry a content hash in the
// filename and can therefore be cached forever.
var ImmutablePrefixes = []string{"/_app/immutable/", "/assets/", "/_nuxt/"}

// Config describes what to serve and from where.
type Config struct {
	// Dir is the directory holding the built client. Defaults to DefaultDir.
	Dir string
	// Index is the history-fallback document. Defaults to index.html.
	Index string
}

// DirFromEnv reads CLIENT_DIR, falling back to DefaultDir.
func DirFromEnv() string {
	if dir := strings.TrimSpace(os.Getenv("CLIENT_DIR")); dir != "" {
		return dir
	}
	return DefaultDir
}

// Available reports whether dir looks like a built client, so an app can skip
// mounting the handler when it was built without one.
func Available(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, "index.html"))
	return err == nil && !info.IsDir()
}

// Handler serves static files from cfg.Dir and falls back to the index document
// for anything that does not resolve, which is what client-side routing needs.
//
// The fallback deliberately does not apply to paths that carry a file
// extension: a missing bundle must 404 rather than receive index.html with a
// text/html content type, which surfaces as an unrelated MIME or syntax error
// in the browser and hides the real problem.
//
// Only GET and HEAD get the fallback. Serving index.html for a POST is how a
// machine client silently succeeds against a route that no longer exists: it
// gets 200 and HTML, and any shipper that treats 2xx as delivery throws the
// payload away. Other verbs get a 405 so the failure is loud.
//
// Containment is delegated to http.Dir, so a traversal attempt cannot escape
// the directory regardless of how the path is encoded.
func Handler(cfg Config) http.Handler {
	dir := cfg.Dir
	if dir == "" {
		dir = DefaultDir
	}
	index := cfg.Index
	if index == "" {
		index = "index.html"
	}

	root := http.Dir(dir)
	files := http.FileServer(root)

	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		upath := path.Clean("/" + request.URL.Path)

		if strings.HasPrefix(path.Base(upath), ".") {
			http.NotFound(w, request)
			return
		}

		file, err := root.Open(upath)
		if err == nil {
			info, statErr := file.Stat()
			_ = file.Close()
			if statErr == nil && !info.IsDir() {
				w.Header().Set("Cache-Control", cachePolicy(upath))
				files.ServeHTTP(w, request)
				return
			}
		}

		if path.Ext(upath) != "" {
			http.NotFound(w, request)
			return
		}

		indexFile, err := root.Open("/" + index)
		if err != nil {
			http.NotFound(w, request)
			return
		}
		info, err := indexFile.Stat()
		if err != nil {
			_ = indexFile.Close()
			http.NotFound(w, request)
			return
		}

		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, request, index, info.ModTime(), indexFile)
		_ = indexFile.Close()
	})
}

func cachePolicy(urlPath string) string {
	for _, prefix := range ImmutablePrefixes {
		if strings.HasPrefix(urlPath, prefix) {
			return "public, max-age=31536000, immutable"
		}
	}
	return "public, max-age=0, must-revalidate"
}
