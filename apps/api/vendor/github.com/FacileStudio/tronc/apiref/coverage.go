package apiref

import (
	"net/http"
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
)

var paramPattern = regexp.MustCompile(`\{([^}:]+):[^}]*\}`)

// Undocumented reports routes that are registered on router but absent from
// cfg.Registry, as "METHOD /path" strings, sorted.
//
// A hand-written registry drifts the moment someone adds an endpoint and
// forgets the inventory. Wiring this into a test turns that drift into a
// failing build:
//
//	func TestEveryRouteIsDocumented(t *testing.T) {
//		if missing := apiref.Undocumented(buildRouter(), refConfig()); len(missing) > 0 {
//			t.Errorf("routes missing from the API registry: %v", missing)
//		}
//	}
//
// The reference page, the health probes and any wildcard route are ignored,
// along with any extra prefixes given in ignore.
func Undocumented(router chi.Routes, cfg Config, ignore ...string) []string {
	documented := map[string]bool{}
	for _, module := range cfg.Registry.Modules {
		for _, route := range module.Routes {
			method := strings.ToUpper(route.Method)
			documented[method+" "+normalize(route.Path)] = true
			for _, server := range cfg.Servers {
				documented[method+" "+normalize(path.Join(server, route.Path))] = true
			}
		}
	}

	skip := append([]string{BasePath, "/health", "/ready", "/api/health", "/api/ready"}, ignore...)

	var missing []string
	_ = chi.Walk(router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		route = normalize(route)
		if skipRoute(route, skip) {
			return nil
		}
		method = strings.ToUpper(method)
		if !documented[method+" "+route] {
			missing = append(missing, method+" "+route)
		}
		return nil
	})
	return sortedUnique(missing)
}

func normalize(route string) string {
	route = paramPattern.ReplaceAllString(route, "{$1}")
	route = strings.TrimSuffix(route, "/")
	if route == "" {
		return "/"
	}
	return route
}

func skipRoute(route string, prefixes []string) bool {
	if strings.Contains(route, "*") {
		return true
	}
	for _, prefix := range prefixes {
		if route == prefix || strings.HasPrefix(route, prefix+"/") {
			return true
		}
	}
	return false
}

func sortedUnique(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
