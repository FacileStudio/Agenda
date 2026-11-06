package middleware

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

var tokenName = regexp.MustCompile(`^[a-z][a-z-]*$`)

type cspDirective struct {
	name    string
	sources []string
}

// defaultCSP is the policy an app gets before it extends anything, in the
// order the header lists it. style-src carries 'unsafe-inline' on purpose:
// Svelte transitions and SvelteKit's own markup write inline styles, and a
// style cannot run code, so the trade is worth it. script-src does not.
//
// It is a function, not a variable, so no importing package can weaken the
// suite's policy by mutating a shared slice.
func defaultCSP() []cspDirective {
	return []cspDirective{
		{"default-src", []string{"'self'"}},
		{"base-uri", []string{"'self'"}},
		{"object-src", []string{"'none'"}},
		{"frame-ancestors", []string{"'none'"}},
		{"form-action", []string{"'self'"}},
		{"img-src", []string{"'self'", "data:", "blob:", "https:"}},
		{"connect-src", []string{"'self'"}},
		{"script-src", []string{"'self'"}},
		{"style-src", []string{"'self'", "'unsafe-inline'"}},
	}
}

// BuildCSP renders the default policy extended by extra, plus report-uri when
// set. Sources extend a directive, replace a 'none' default, and a directive
// the default lacks is appended in name order. It refuses a directive name or
// source that would inject a directive or split the header.
func BuildCSP(extra map[string][]string, reportURI string) (string, error) {
	directives, err := buildDirectives(extra, reportURI)
	if err != nil {
		return "", err
	}
	return renderPolicy(directives), nil
}

func buildDirectives(extra map[string][]string, reportURI string) ([]cspDirective, error) {
	if err := validateCSP(extra, reportURI); err != nil {
		return nil, err
	}
	directives := extendDirectives(defaultCSP(), extra, true)
	if reportURI != "" {
		directives = append(directives, cspDirective{"report-uri", []string{reportURI}})
	}
	return directives, nil
}

// framingPolicy picks out of a full policy the minimal one enforced even in
// report-only mode, so framing, plugins and base-tag hijacks are blocked from
// the first deploy while the full policy is still being tuned.
func framingPolicy(directives []cspDirective) []cspDirective {
	names := []string{"frame-ancestors", "object-src", "base-uri"}
	picked := make([]cspDirective, 0, len(names))
	for _, name := range names {
		if index := slices.IndexFunc(directives, func(d cspDirective) bool { return d.name == name }); index >= 0 {
			picked = append(picked, directives[index])
		}
	}
	return picked
}

func validateCSP(extra map[string][]string, reportURI string) error {
	for name, sources := range extra {
		if !tokenName.MatchString(name) {
			return fmt.Errorf("tronc/middleware: CSP directive %q is not a directive name", name)
		}
		for _, source := range sources {
			if !validSource(source) {
				return fmt.Errorf("tronc/middleware: CSP source %q would corrupt the policy", source)
			}
		}
	}
	if reportURI != "" && !validSource(reportURI) {
		return fmt.Errorf("tronc/middleware: CSP report-uri %q would corrupt the policy", reportURI)
	}
	return nil
}

func validSource(source string) bool {
	return source != "" && !strings.ContainsAny(source, ";, \t\r\n") && !strings.ContainsFunc(source, unicode.IsControl)
}

// extendDirectives merges extra into directives. A directive extra names but
// directives lacks is appended in name order only when addNew is set, which
// keeps the minimal framing policy from growing a script-src that would
// enforce what the full policy only reports.
func extendDirectives(directives []cspDirective, extra map[string][]string, addNew bool) []cspDirective {
	merged := make([]cspDirective, 0, len(directives)+len(extra))
	known := make(map[string]bool, len(directives))
	for _, directive := range directives {
		known[directive.name] = true
		merged = append(merged, cspDirective{directive.name, mergeSources(directive.sources, extra[directive.name])})
	}
	if !addNew {
		return merged
	}
	added := make([]string, 0, len(extra))
	for name := range extra {
		if !known[name] {
			added = append(added, name)
		}
	}
	slices.Sort(added)
	for _, name := range added {
		merged = append(merged, cspDirective{name, mergeSources(nil, extra[name])})
	}
	return merged
}

func mergeSources(defaults, extra []string) []string {
	if len(extra) == 0 {
		return defaults
	}
	if len(defaults) == 1 && defaults[0] == "'none'" {
		defaults = nil
	}
	merged := slices.Clone(defaults)
	for _, source := range extra {
		if !slices.Contains(merged, source) {
			merged = append(merged, source)
		}
	}
	return merged
}
