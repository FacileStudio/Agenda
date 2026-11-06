package middleware

import (
	"fmt"
	"slices"
	"strings"
)

func openerPolicy(value string) (string, error) {
	if value == "" {
		return "same-origin", nil
	}
	allowed := []string{"same-origin", "same-origin-allow-popups", "noopener-allow-popups", "unsafe-none"}
	if !slices.Contains(allowed, value) {
		return "", fmt.Errorf("tronc/middleware: Cross-Origin-Opener-Policy %q is not one of %v", value, allowed)
	}
	return value, nil
}

func permissionsPolicy(allow []string) (string, error) {
	for _, feature := range allow {
		if !tokenName.MatchString(feature) {
			return "", fmt.Errorf("tronc/middleware: Permissions-Policy feature %q is not a feature name", feature)
		}
	}
	return grantFeatures("camera=(), microphone=(), geolocation=()", allow), nil
}

// grantFeatures sets each allowed feature to (self) in policy, appending the
// ones it does not list yet.
func grantFeatures(policy string, allow []string) string {
	var parts []string
	var names []string
	for _, part := range strings.Split(policy, ",") {
		name, value, _ := strings.Cut(strings.TrimSpace(part), "=")
		if name == "" {
			continue
		}
		if slices.Contains(allow, name) {
			value = "(self)"
		}
		names = append(names, name)
		parts = append(parts, name+"="+value)
	}
	for _, feature := range allow {
		if !slices.Contains(names, feature) {
			names = append(names, feature)
			parts = append(parts, feature+"=(self)")
		}
	}
	return strings.Join(parts, ", ")
}
