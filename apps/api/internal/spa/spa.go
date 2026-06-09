package spa

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func Handler(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := filepath.Join(dir, filepath.Clean(r.URL.Path))

		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			idx := filepath.Join(dir, "index.html")
			if _, err := os.Stat(idx); err != nil {
				http.NotFound(w, r)
				return
			}
			http.ServeFile(w, r, idx)
			return
		}

		if strings.HasPrefix(filepath.Base(path), ".") {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Cache-Control", cachePolicy(r.URL.Path))
		http.ServeFile(w, r, path)
	})
}

func cachePolicy(urlPath string) string {
	if strings.HasPrefix(urlPath, "/_app/immutable/") {
		return "public, max-age=31536000, immutable"
	}
	return "public, max-age=0, must-revalidate"
}
