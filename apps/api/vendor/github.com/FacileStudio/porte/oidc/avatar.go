package oidc

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"
)

// MaxAvatarBytes is the ceiling on a fetched avatar, unchanged from the apps.
const MaxAvatarBytes = 5 << 20

// avatarTimeout bounds the whole fetch. An IdP that hangs must not hang the
// login with it.
const avatarTimeout = 10 * time.Second

// FetchAvatar downloads a picture URL asserted by an identity provider and
// returns the bytes and their content type.
//
// This is the one function in porte where copy-paste drift would be a
// vulnerability rather than an inconsistency: the URL comes from an ID token,
// so it is attacker-influenced input handed to an HTTP client running inside
// the app's network. Six divergent copies exist today; this is the version
// they should all have been.
//
// The guard is applied at connect time, not before it. Every existing copy
// resolves the hostname, checks the addresses, and then calls http.Get, which
// resolves again — a DNS entry that answers publicly on the first lookup and
// privately on the second walks straight through. Checking inside the dialer
// closes that window, and covers redirects for free.
func FetchAvatar(ctx context.Context, pictureURL string) (data []byte, contentType string, err error) {
	if err := requireHTTPS(pictureURL); err != nil {
		return nil, "", err
	}

	client := &http.Client{
		Timeout: avatarTimeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: 5 * time.Second,
				Control: func(network, address string, _ syscall.RawConn) error {
					host, _, splitErr := net.SplitHostPort(address)
					if splitErr != nil {
						return fmt.Errorf("porte: unparseable dial address %q", address)
					}
					if ip := net.ParseIP(host); ip == nil || !isPublicIP(ip) {
						return fmt.Errorf("porte: refusing to fetch an avatar from a non-public address (%s)", host)
					}
					return nil
				},
			}).DialContext,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("porte: too many redirects fetching the avatar")
			}
			return requireHTTPS(req.URL.String())
		},
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, pictureURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("porte: invalid picture URL: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, "", fmt.Errorf("porte: avatar fetch failed: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("porte: avatar fetch returned status %d", response.StatusCode)
	}
	contentType = strings.TrimSpace(strings.SplitN(response.Header.Get("Content-Type"), ";", 2)[0])
	if !isImageType(contentType) {
		return nil, "", fmt.Errorf("porte: unsupported avatar content type %q", contentType)
	}

	// One byte past the limit, so an oversized body is detected rather
	// than silently truncated into a corrupt image.
	data, err = io.ReadAll(io.LimitReader(response.Body, MaxAvatarBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("porte: reading the avatar failed: %w", err)
	}
	if len(data) > MaxAvatarBytes {
		return nil, "", fmt.Errorf("porte: avatar exceeds the %d byte limit", MaxAvatarBytes)
	}
	return data, contentType, nil
}

func requireHTTPS(rawURL string) error {
	if !strings.HasPrefix(strings.ToLower(rawURL), "https://") {
		return fmt.Errorf("porte: avatar URLs must be https, got %q", rawURL)
	}
	return nil
}

// blockedNetworks are the ranges net.IP's own predicates do not cover.
// Together with those predicates this is the deny list an avatar fetch is
// checked against; anything not named here is treated as public.
var blockedNetworks = mustParseCIDRs(
	// IPv4. 169.254.0.0/16 is link-local and already covered — it is where
	// every cloud metadata service lives, and it is the reason this guard
	// exists at all.
	"0.0.0.0/8",          // "this network"
	"100.64.0.0/10",      // carrier-grade NAT, RFC 6598
	"192.0.0.0/24",       // IETF protocol assignments
	"192.0.2.0/24",       // TEST-NET-1
	"198.18.0.0/15",      // benchmarking, RFC 2544
	"198.51.100.0/24",    // TEST-NET-2
	"203.0.113.0/24",     // TEST-NET-3
	"240.0.0.0/4",        // reserved
	"255.255.255.255/32", // broadcast

	// IPv6.
	"100::/64",       // discard-only, RFC 6666
	"2001::/32",      // Teredo — carries an embedded IPv4 nobody needs to reach
	"64:ff9b:1::/48", // local-use NAT64, RFC 8215
	"2001:db8::/32",  // documentation
	"3fff::/20",      // documentation, RFC 9637
)

// isPublicIP reports whether an address may be dialled for an avatar fetch.
//
// The list is a deny list, and the subtle half is the IPv6 forms that embed an
// IPv4 address: net.IP's predicates only understand the plain forms and the
// IPv4-mapped one, so 64:ff9b::a9fe:a9fe (NAT64) and 2002:a9fe:a9fe:: (6to4)
// both reach 169.254.169.254 while looking like ordinary public IPv6. Those
// are unwrapped to the address they actually reach and checked again.
func isPublicIP(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if embedded := embeddedIPv4(ip); embedded != nil {
		return isPublicIP(embedded)
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() ||
		ip.IsInterfaceLocalMulticast() {
		return false
	}
	for _, blocked := range blockedNetworks {
		if blocked.Contains(ip) {
			return false
		}
	}
	return true
}

// embeddedIPv4 returns the IPv4 address an IPv6 address actually reaches, or
// nil when it reaches no IPv4 address. NAT64 is unwrapped rather than blocked
// outright because on an IPv6-only host every IPv4 destination — including
// every legitimate avatar host — arrives in that form.
func embeddedIPv4(ip net.IP) net.IP {
	if mapped := ip.To4(); mapped != nil {
		// Either a plain IPv4 or the ::ffff:a.b.c.d form. Only the
		// second is a different value worth re-checking.
		if len(ip) == net.IPv6len && !mapped.Equal(ip) {
			return mapped
		}
		return nil
	}
	if len(ip) != net.IPv6len {
		return nil
	}
	switch {
	case ip[0] == 0x00 && ip[1] == 0x64 && ip[2] == 0xff && ip[3] == 0x9b:
		// NAT64 well-known prefix 64:ff9b::/96, RFC 6052.
		return net.IPv4(ip[12], ip[13], ip[14], ip[15])
	case ip[0] == 0x20 && ip[1] == 0x02:
		// 6to4, 2002::/16, RFC 3056: the IPv4 is bytes 2 through 5.
		return net.IPv4(ip[2], ip[3], ip[4], ip[5])
	case isZeros(ip[:12]) && !isZeros(ip[12:]):
		// The deprecated IPv4-compatible form ::a.b.c.d, which
		// ::127.0.0.1 uses and net.IP.IsLoopback does not recognise.
		return net.IPv4(ip[12], ip[13], ip[14], ip[15])
	}
	return nil
}

func isZeros(bytes []byte) bool {
	for _, b := range bytes {
		if b != 0 {
			return false
		}
	}
	return true
}

func mustParseCIDRs(blocks ...string) []*net.IPNet {
	networks := make([]*net.IPNet, 0, len(blocks))
	for _, block := range blocks {
		_, network, err := net.ParseCIDR(block)
		if err != nil {
			panic("porte: unparseable blocked network " + block)
		}
		networks = append(networks, network)
	}
	return networks
}

func isImageType(contentType string) bool {
	switch contentType {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}
