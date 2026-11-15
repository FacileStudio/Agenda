// Package live turns the database's change notifications into per-user
// frames for the web client. Triggers on events, calendars, their shares,
// space memberships and activities call pg_notify on Channel, so every writer
// is covered: the API, CalDAV, MCP, the Opus mirror, erasure and raw SQL.
package live

import (
	"strconv"
	"strings"
	"sync"
)

// Channel is the Postgres notification channel the migrations' triggers use.
const Channel = "agenda_changes"

// Kind tells what a change touched.
type Kind byte

const (
	// KindEvent is an event written in the calendar ID.
	KindEvent Kind = 'e'
	// KindCalendar is the calendar ID itself, its shares or its space's members.
	KindCalendar Kind = 'c'
	// KindActivity is an activity of the user ID.
	KindActivity Kind = 'u'
	// KindResync follows a lost listener: notifications may have been missed.
	KindResync Kind = 'r'
)

// Change is one decoded notification.
type Change struct {
	Kind Kind
	ID   int64
}

// ParseChange decodes a trigger payload such as "e:12".
func ParseChange(payload string) (Change, bool) {
	prefix, value, found := strings.Cut(payload, ":")
	if !found || len(prefix) != 1 {
		return Change{}, false
	}
	id, err := strconv.ParseInt(value, 10, 64)
	kind := Kind(prefix[0])
	if err != nil || (kind != KindEvent && kind != KindCalendar && kind != KindActivity) {
		return Change{}, false
	}
	return Change{Kind: kind, ID: id}, true
}

const subscriberBuffer = 256

// Hub fans every change out to the open streams. A stream that falls a full
// buffer behind misses changes rather than stalling the others; the client
// refetches its whole view on any frame, so a later one catches it up.
type Hub struct {
	mu     sync.Mutex
	subs   map[chan Change]struct{}
	done   chan struct{}
	closed bool
}

// NewHub returns an open hub.
func NewHub() *Hub {
	return &Hub{subs: map[chan Change]struct{}{}, done: make(chan struct{})}
}

// Subscribe opens a stream of changes and returns it with its cancel.
func (h *Hub) Subscribe() (<-chan Change, func()) {
	ch := make(chan Change, subscriberBuffer)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
	}
}

// Publish hands c to every stream without blocking.
func (h *Hub) Publish(c Change) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- c:
		default:
		}
	}
}

// Done is closed when the hub shuts down, which ends every stream.
func (h *Hub) Done() <-chan struct{} { return h.done }

// Close ends every stream. The server calls it when shutdown starts, since
// it would otherwise wait on streams that never go idle.
func (h *Hub) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.closed {
		h.closed = true
		close(h.done)
	}
}
