package middleware

// CloudflareProxies are Cloudflare's published edge ranges, from
// https://www.cloudflare.com/ips-v4 and /ips-v6, fetched 2026-08-10.
//
// It is opt-in: an app fronted by Cloudflare adds it to its trust set so
// RealIP walks past the edge and reaches the visitor. Without it, every
// visitor behind one edge shares a rate-limit bucket and one visitor moving
// between edges looks like several — the limits still hold, they just stop
// describing people.
//
// Adding it is safe **because RealIP walks the chain right to left.**
// Cloudflare appends the visitor's address to X-Forwarded-For rather than
// replacing it, so the rightmost entry is always the one Cloudflare wrote.
// The attack this normally invites — an attacker pointing their own
// Cloudflare zone at your origin so their traffic arrives from a trusted edge
// — cannot reach the entries they pre-seeded, because those sit to the left of
// the address Cloudflare appended for them. A scheme that took the *first* hop,
// or trusted Cf-Connecting-Ip outright, would hand them the forged value.
//
// The list is a snapshot, not a subscription: fetching it at boot would make
// startup depend on the network and turn a hijacked response into a trust
// bypass. A range that drifts out of date degrades attribution for that edge;
// it does not open anything. Refresh it here when Cloudflare publishes a change.
var CloudflareProxies = mustPrefixes(
	"173.245.48.0/20",
	"103.21.244.0/22",
	"103.22.200.0/22",
	"103.31.4.0/22",
	"141.101.64.0/18",
	"108.162.192.0/18",
	"190.93.240.0/20",
	"188.114.96.0/20",
	"197.234.240.0/22",
	"198.41.128.0/17",
	"162.158.0.0/15",
	"104.16.0.0/13",
	"104.24.0.0/14",
	"172.64.0.0/13",
	"131.0.72.0/22",
	"2400:cb00::/32",
	"2606:4700::/32",
	"2803:f800::/32",
	"2405:b500::/32",
	"2405:8100::/32",
	"2a06:98c0::/29",
	"2c0f:f248::/32",
)
