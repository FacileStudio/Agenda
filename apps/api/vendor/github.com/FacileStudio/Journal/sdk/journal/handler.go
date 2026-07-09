package journal

import (
	"context"
	"log/slog"
	"time"
)

// Handler is a slog.Handler that ships every record to Journal and forwards
// it to an optional next handler, so existing logging keeps working
// unchanged. Wrap your current handler once at startup:
//
//	client := journal.New(journal.Config{URL: url, Token: token})
//	slog.SetDefault(slog.New(journal.NewHandler(client, slog.Default().Handler())))
type Handler struct {
	client *Client
	next   slog.Handler
	meta   map[string]any
	groups []string
}

// NewHandler wraps next with Journal shipping. next may be nil to only ship.
func NewHandler(client *Client, next slog.Handler) *Handler {
	return &Handler{client: client, next: next}
}

// Enabled defers to the next handler, or accepts info and above when only shipping.
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	if h.next != nil {
		return h.next.Enabled(ctx, level)
	}
	return level >= slog.LevelInfo
}

// Handle forwards the record to the next handler and buffers it for shipping.
func (h *Handler) Handle(ctx context.Context, record slog.Record) error {
	var err error
	if h.next != nil {
		err = h.next.Handle(ctx, record.Clone())
	}
	meta := make(map[string]any, len(h.meta)+record.NumAttrs())
	for key, value := range h.meta {
		meta[key] = value
	}
	prefix := groupPrefix(h.groups)
	record.Attrs(func(attr slog.Attr) bool {
		flattenAttr(prefix, attr, meta)
		return true
	})
	if len(meta) == 0 {
		meta = nil
	}
	if h.client == nil {
		return err
	}
	ts := record.Time
	if ts.IsZero() {
		ts = time.Now()
	}
	h.client.enqueue(Entry{
		App:     h.client.cfg.App,
		Level:   levelName(record.Level),
		Message: record.Message,
		Ts:      ts.UTC().Format(time.RFC3339Nano),
		Meta:    meta,
	})
	return err
}

// WithAttrs returns a handler whose shipped entries include the given attrs.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := h.clone()
	prefix := groupPrefix(h.groups)
	for _, attr := range attrs {
		flattenAttr(prefix, attr, clone.meta)
	}
	if h.next != nil {
		clone.next = h.next.WithAttrs(attrs)
	}
	return clone
}

// WithGroup returns a handler that prefixes subsequent attr keys with name.
func (h *Handler) WithGroup(name string) slog.Handler {
	clone := h.clone()
	if name != "" {
		clone.groups = append(clone.groups, name)
	}
	if h.next != nil {
		clone.next = h.next.WithGroup(name)
	}
	return clone
}

func (h *Handler) clone() *Handler {
	meta := make(map[string]any, len(h.meta))
	for key, value := range h.meta {
		meta[key] = value
	}
	groups := make([]string, len(h.groups))
	copy(groups, h.groups)
	return &Handler{client: h.client, next: h.next, meta: meta, groups: groups}
}

func groupPrefix(groups []string) string {
	prefix := ""
	for _, group := range groups {
		prefix += group + "."
	}
	return prefix
}

func flattenAttr(prefix string, attr slog.Attr, into map[string]any) {
	value := attr.Value.Resolve()
	if value.Kind() == slog.KindGroup {
		nested := prefix
		if attr.Key != "" {
			nested += attr.Key + "."
		}
		for _, sub := range value.Group() {
			flattenAttr(nested, sub, into)
		}
		return
	}
	if attr.Key == "" {
		return
	}
	into[prefix+attr.Key] = value.Any()
}

func levelName(level slog.Level) string {
	switch {
	case level < slog.LevelInfo:
		return "debug"
	case level < slog.LevelWarn:
		return "info"
	case level < slog.LevelError:
		return "warn"
	default:
		return "error"
	}
}
