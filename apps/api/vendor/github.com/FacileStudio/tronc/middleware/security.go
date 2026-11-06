package middleware

import (
	"log/slog"
	"net/http"
	"net/netip"
	"slices"
	"strings"
)

// Header names SecurityHeaders writes.
const (
	HeaderHSTS              = "Strict-Transport-Security"
	HeaderCSP               = "Content-Security-Policy"
	HeaderCSPReportOnly     = "Content-Security-Policy-Report-Only"
	HeaderContentTypeOpts   = "X-Content-Type-Options"
	HeaderFrameOptions      = "X-Frame-Options"
	HeaderReferrerPolicy    = "Referrer-Policy"
	HeaderPermissionsPolicy = "Permissions-Policy"
	HeaderCOOP              = "Cross-Origin-Opener-Policy"
)

// MaxCSPHeaderBytes is the CSP length past which SecurityHeaders warns at
// startup. Proxies cap response headers: nginx's default proxy_buffer_size
// holds all of them in 4 KB or 8 KB, and an over-long header there fails the
// request with a 502 rather than dropping the policy. Each inline script hash
// costs 54 bytes with its separator, so the default policy has room for about 70.
const MaxCSPHeaderBytes = 4096

// HSTSValue is the Strict-Transport-Security value sent on HTTPS requests.
const HSTSValue = "max-age=31536000; includeSubDomains"

// SecurityConfig shapes the security headers. The zero value is the suite
// default, with the full CSP in report-only mode.
type SecurityConfig struct {
	// CSPEnforce sends the full policy as Content-Security-Policy. False,
	// where every app starts, sends it as Content-Security-Policy-Report-Only
	// and enforces only its frame-ancestors, object-src and base-uri, in a
	// Content-Security-Policy of their own.
	CSPEnforce bool
	// CSPSources adds sources to a directive, keyed by directive name. See
	// BuildCSP for how they combine with the default policy.
	CSPSources map[string][]string
	// CSPReportURI, when set, appends a report-uri directive. Without it, a
	// report-only violation shows in the browser console and nowhere else.
	CSPReportURI string
	// Allow grants each listed Permissions-Policy feature to the page's own
	// origin, e.g. "camera" and "microphone" for a video app. camera,
	// microphone and geolocation are denied unless listed.
	Allow []string
	// CrossOriginOpenerPolicy defaults to same-origin, which cuts the link
	// to a cross-origin popup: an OAuth or payment popup that reports back
	// through window.opener needs same-origin-allow-popups.
	CrossOriginOpenerPolicy string
	// HSTSExcludeSubdomains drops includeSubDomains from HSTS, for an app on
	// a host whose subdomains are not all served over HTTPS.
	HSTSExcludeSubdomains bool
	// Trusted are the peers whose X-Forwarded-Proto is believed for the
	// HSTS decision. nil trusts none; httpx fills it with the router's
	// trusted proxies.
	Trusted []netip.Prefix
}

type securityHeaders struct {
	static  [][2]string
	hsts    string
	trusted []netip.Prefix
}

// SecurityHeaders sets the suite's security headers on every response.
//
// Each header is set before the handler runs and only when nothing upstream
// set it, so a handler that calls Header().Set on one of them wins. Use Set,
// not Add: an added second CSP is enforced alongside the first, not instead.
//
// X-Frame-Options: DENY is sent unless CSPSources opens frame-ancestors to an
// origin, so browsers that ignore frame-ancestors are covered too.
//
// It panics at construction on a source, directive or feature that would
// corrupt the header, the same stance CORS takes on a credential leak, and
// warns through slog.Default when a CSP header exceeds MaxCSPHeaderBytes.
func SecurityHeaders(cfg SecurityConfig) func(http.Handler) http.Handler {
	headers := must(newSecurityHeaders(cfg))
	for _, pair := range headers.static {
		if (pair[0] == HeaderCSP || pair[0] == HeaderCSPReportOnly) && len(pair[1]) > MaxCSPHeaderBytes {
			slog.Warn("tronc/middleware: CSP header exceeds the size proxies reliably pass",
				slog.String("header", pair[0]), slog.Int("bytes", len(pair[1])), slog.Int("limit", MaxCSPHeaderBytes))
		}
	}
	return headers.middleware
}

// must turns a construction error into the startup panic both middlewares
// promise, so a malformed policy never reaches a response.
func must[T any](value T, err error) T {
	if err != nil {
		panic(err.Error())
	}
	return value
}

func newSecurityHeaders(cfg SecurityConfig) (*securityHeaders, error) {
	coop, err := openerPolicy(cfg.CrossOriginOpenerPolicy)
	if err != nil {
		return nil, err
	}
	directives, err := buildDirectives(cfg.CSPSources, cfg.CSPReportURI)
	if err != nil {
		return nil, err
	}
	permissions, err := permissionsPolicy(cfg.Allow)
	if err != nil {
		return nil, err
	}

	static := append(cspHeaders(directives, cfg),
		[2]string{HeaderContentTypeOpts, "nosniff"},
		[2]string{HeaderReferrerPolicy, "strict-origin-when-cross-origin"},
		[2]string{HeaderPermissionsPolicy, permissions},
		[2]string{HeaderCOOP, coop},
	)
	hsts := HSTSValue
	if cfg.HSTSExcludeSubdomains {
		hsts = "max-age=31536000"
	}
	return &securityHeaders{static: static, hsts: hsts, trusted: cfg.Trusted}, nil
}

// cspHeaders are the full policy, enforced or reported, the framing policy
// enforced beside a reported one, and X-Frame-Options while framing is shut.
func cspHeaders(directives []cspDirective, cfg SecurityConfig) [][2]string {
	headers := [][2]string{{HeaderCSP, renderPolicy(directives)}}
	if !cfg.CSPEnforce {
		headers = [][2]string{
			{HeaderCSPReportOnly, renderPolicy(directives)},
			{HeaderCSP, renderPolicy(framingPolicy(directives))},
		}
	}
	if !framingOpened(cfg.CSPSources["frame-ancestors"]) {
		headers = append(headers, [2]string{HeaderFrameOptions, "DENY"})
	}
	return headers
}

// framingOpened reports whether frame-ancestors sources let any origin frame
// the page. An explicit 'none' keeps it shut, and X-Frame-Options with it.
func framingOpened(sources []string) bool {
	return slices.ContainsFunc(sources, func(source string) bool { return source != "'none'" })
}

func (s *securityHeaders) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		header := w.Header()
		for _, pair := range s.static {
			setIfAbsent(header, pair[0], pair[1])
		}
		if IsHTTPS(request, s.trusted) {
			setIfAbsent(header, HeaderHSTS, s.hsts)
		}
		next.ServeHTTP(w, request)
	})
}

func setIfAbsent(header http.Header, key, value string) {
	if _, present := header[key]; !present {
		header.Set(key, value)
	}
}

// IsHTTPS reports whether the visitor reached this server over HTTPS: the
// connection itself is TLS, or a trusted peer says so in X-Forwarded-Proto.
// Only the rightmost value counts, the one the nearest proxy wrote.
func IsHTTPS(request *http.Request, trusted []netip.Prefix) bool {
	if request.TLS != nil {
		return true
	}
	if len(trusted) == 0 {
		return false
	}
	peer, ok := parseAddr(request.RemoteAddr)
	if !ok || !TrustedBy(peer, trusted) {
		return false
	}
	values := request.Header.Values("X-Forwarded-Proto")
	if len(values) == 0 {
		return false
	}
	hops := strings.Split(values[len(values)-1], ",")
	return strings.EqualFold(strings.TrimSpace(hops[len(hops)-1]), "https")
}
