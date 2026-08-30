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

// Incomplete reports routes in cfg.Registry that lack required documentation
// fields (such as missing summary, missing request body on POST/PUT/PATCH, or
// missing response body on non-204 routes).
//
// Routes matching prefixes in ignore are skipped.
func Incomplete(cfg Config, ignore ...string) []string {
	var issues []string
	for _, module := range cfg.Registry.Modules {
		for _, route := range module.Routes {
			routePath := normalize(route.Path)
			if skipRoute(routePath, ignore) {
				continue
			}
			method := strings.ToUpper(route.Method)
			label := method + " " + routePath

			if strings.TrimSpace(route.Summary) == "" {
				issues = append(issues, label+": missing summary")
			}
			if (method == "POST" || method == "PUT" || method == "PATCH") && (route.RequestBody == nil || route.RequestBody == "") {
				issues = append(issues, label+": missing request body")
			}
			if route.Status != http.StatusNoContent && (route.ResponseBody == nil || route.ResponseBody == "") {
				issues = append(issues, label+": missing response body")
			}
		}
	}
	return sortedUnique(issues)
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
