package middleware

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// DefaultTrustedProxies is the trust set an app gets when it configures none:
// loopback and the private ranges.
//
// It is the deployment every Facile app actually has. Traefik and the API share
// a Docker network, so the peer is always an RFC1918 address and the forwarded
// header is the only way to tell two visitors apart. Trusting nothing by
// default would be strictly safer on paper and actively harmful in practice —
// every request would carry Traefik's single address, so a per-IP rate limit
// would become one global bucket and the login limiter would lock the whole
// world out together.
var DefaultTrustedProxies = mustPrefixes(
	"127.0.0.0/8",
	"::1/128",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"169.254.0.0/16",
	"fc00::/7",
	"fe80::/10",
)

// RealIP rewrites RemoteAddr from X-Forwarded-For, but only when the peer is
// one of trusted.
//
// This is the whole difference from chi's middleware of the same name, and it
// is not a detail: chi's rewrites RemoteAddr from the header on **every**
// request, whatever the peer, so anything able to set a header can hand itself
// a fresh identity. Every per-IP rate limit in an app built that way is
// decorative — measured on Journal, 70 requests carrying a rotating
// X-Forwarded-For were all accepted where a 60/min bucket should have refused
// ten of them.
//
// The header is walked right to left, skipping addresses that are themselves
// trusted proxies, and the first address that is not becomes RemoteAddr. Each
// proxy appends the address it received from, so the rightmost entry is the one
// written by the hop nearest this server — the only entry no client could have
// forged — and walking left from there survives a chain (Cloudflare in front of
// Traefik, say) without trusting anything the client wrote.
//
// A nil or empty trusted set means trust nothing: the header is ignored and
// RemoteAddr stands. Callers that want the default must pass
// DefaultTrustedProxies explicitly, so a zero value can never widen trust.
func RealIP(trusted []netip.Prefix) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
			if client, ok := clientAddr(request, trusted); ok {
				request = request.Clone(request.Context())
				request.RemoteAddr = client
			}
			next.ServeHTTP(w, request)
		})
	}
}

// clientAddr resolves the caller's address, reporting false when the header
// must not be believed.
func clientAddr(request *http.Request, trusted []netip.Prefix) (string, bool) {
	if len(trusted) == 0 {
		return "", false
	}
	peer, ok := parseAddr(request.RemoteAddr)
	if !ok || !TrustedBy(peer, trusted) {
		return "", false
	}

	forwarded := request.Header.Get("X-Forwarded-For")
	if forwarded == "" {
		return "", false
	}

	hops := strings.Split(forwarded, ",")
	for i := len(hops) - 1; i >= 0; i-- {
		hop, ok := parseAddr(hops[i])
		if !ok {
			// An unparsable hop is a forged one. Everything to its left
			// was written by whoever wrote it, so the walk stops here
			// rather than reaching past it.
			return "", false
		}
		if !TrustedBy(hop, trusted) {
			return hop.String(), true
		}
	}
	return "", false
}

// TrustedBy reports whether addr falls in any of the prefixes.
func TrustedBy(addr netip.Addr, trusted []netip.Prefix) bool {
	addr = addr.Unmap()
	for _, prefix := range trusted {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// ParseTrustedProxies turns CIDR blocks, or bare addresses, into prefixes.
// It is what reads a TRUSTED_PROXIES setting.
func ParseTrustedProxies(values []string) ([]netip.Prefix, error) {
	prefixes := make([]netip.Prefix, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if prefix, err := netip.ParsePrefix(trimmed); err == nil {
			prefixes = append(prefixes, prefix.Masked())
			continue
		}
		addr, err := netip.ParseAddr(trimmed)
		if err != nil {
			return nil, &net.ParseError{Type: "CIDR address or IP", Text: trimmed}
		}
		prefixes = append(prefixes, netip.PrefixFrom(addr.Unmap(), addr.BitLen()))
	}
	return prefixes, nil
}

// parseAddr accepts "host:port", a bare address, and the bracketed IPv6 form.
func parseAddr(value string) (netip.Addr, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return netip.Addr{}, false
	}
	if addr, err := netip.ParseAddr(trimmed); err == nil {
		return addr.Unmap(), true
	}
	if host, _, err := net.SplitHostPort(trimmed); err == nil {
		if addr, err := netip.ParseAddr(host); err == nil {
			return addr.Unmap(), true
		}
	}
	return netip.Addr{}, false
}

func mustPrefixes(values ...string) []netip.Prefix {
	prefixes, err := ParseTrustedProxies(values)
	if err != nil {
		panic("tronc/middleware: bad default trusted proxy: " + err.Error())
	}
	return prefixes
}
