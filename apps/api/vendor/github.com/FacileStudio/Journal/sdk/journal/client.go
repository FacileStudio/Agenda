package journal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	maxServerBatch = 1000
	maxRetryAfter  = 5 * time.Minute
)

// Config configures a Client. URL and Token are required; everything else
// has sensible defaults.
type Config struct {
	// URL is the Journal API base URL, e.g. http://journal-api:4010.
	URL string
	// Token is a per-app API key (journal_<app>_...) or the legacy INGEST_TOKEN.
	Token string
	// App is the source app name. Leave empty when Token is a per-app key —
	// the server fills it from the key's scope.
	App string
	// FlushInterval is how often buffered entries are shipped. Default 2s.
	FlushInterval time.Duration
	// MaxBatch triggers an early flush when the buffer reaches this size. Default 200.
	MaxBatch int
	// BufferCap bounds the buffer; the oldest entries are dropped beyond it. Default 5000.
	BufferCap int
	// HTTPClient overrides the default client (10s timeout).
	HTTPClient *http.Client
}

// Entry is one log entry as accepted by POST /ingest.
type Entry struct {
	App     string         `json:"app,omitempty"`
	Level   string         `json:"level,omitempty"`
	Message string         `json:"message"`
	Ts      string         `json:"ts,omitempty"`
	Meta    map[string]any `json:"meta,omitempty"`
}

// Client ships log entries to Journal in the background. Shipping is
// best-effort and never blocks or panics: batches are retried on 429/5xx and
// network errors, dropped on any other 4xx, and the oldest entries are
// dropped if the buffer overflows while Journal is unreachable.
type Client struct {
	cfg  Config
	http *http.Client

	mu      sync.Mutex
	buf     []Entry
	dropped int

	retryAt time.Time

	closeOnce sync.Once
	kick      chan struct{}
	stop      chan struct{}
	done      chan struct{}
}

// New starts a Client and its background shipper goroutine. Call Close to
// flush and stop it.
func New(cfg Config) *Client {
	cfg.URL = strings.TrimRight(cfg.URL, "/")
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = 2 * time.Second
	}
	if cfg.MaxBatch <= 0 {
		cfg.MaxBatch = 200
	}
	if cfg.MaxBatch > maxServerBatch {
		cfg.MaxBatch = maxServerBatch
	}
	if cfg.BufferCap <= 0 {
		cfg.BufferCap = 5000
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	c := &Client{
		cfg:  cfg,
		http: httpClient,
		kick: make(chan struct{}, 1),
		stop: make(chan struct{}),
		done: make(chan struct{}),
	}
	go c.run()
	return c
}

// Log buffers one entry. Level must be debug, info, warn or error; anything
// else is stored as-is and normalized by the server. Meta values are
// sanitized to JSON-safe forms immediately, so the caller may reuse or
// mutate the map after Log returns.
func (c *Client) Log(level, message string, meta map[string]any) {
	c.enqueue(Entry{
		App:     c.cfg.App,
		Level:   level,
		Message: message,
		Ts:      time.Now().UTC().Format(time.RFC3339Nano),
		Meta:    meta,
	})
}

// Debug logs at debug level.
func (c *Client) Debug(message string, meta map[string]any) { c.Log("debug", message, meta) }

// Info logs at info level.
func (c *Client) Info(message string, meta map[string]any) { c.Log("info", message, meta) }

// Warn logs at warn level.
func (c *Client) Warn(message string, meta map[string]any) { c.Log("warn", message, meta) }

// Error logs at error level.
func (c *Client) Error(message string, meta map[string]any) { c.Log("error", message, meta) }

// Dropped reports how many entries were discarded, either from buffer
// overflow or because they could not be delivered by Close.
func (c *Client) Dropped() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dropped
}

// Close drains buffered entries (best effort, bounded) and stops the shipper
// goroutine. Entries that cannot be delivered are counted in Dropped. Close
// is idempotent and safe to call concurrently.
func (c *Client) Close() {
	c.closeOnce.Do(func() { close(c.stop) })
	<-c.done
}

func (c *Client) enqueue(entry Entry) {
	entry.Meta = sanitizeMeta(entry.Meta)
	c.mu.Lock()
	if len(c.buf) >= c.cfg.BufferCap {
		c.buf = c.buf[1:]
		c.dropped++
	}
	c.buf = append(c.buf, entry)
	full := len(c.buf) >= c.cfg.MaxBatch
	c.mu.Unlock()
	if full {
		select {
		case c.kick <- struct{}{}:
		default:
		}
	}
}

func (c *Client) run() {
	ticker := time.NewTicker(c.cfg.FlushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.flush(false)
		case <-c.kick:
			c.flush(false)
		case <-c.stop:
			limit := c.cfg.BufferCap/maxServerBatch + 1
			for i := 0; i <= limit && c.flush(true); i++ {
			}
			c.mu.Lock()
			c.dropped += len(c.buf)
			c.buf = nil
			c.mu.Unlock()
			close(c.done)
			return
		}
	}
}

func (c *Client) flush(force bool) bool {
	if !force && time.Now().Before(c.retryAt) {
		return false
	}
	c.mu.Lock()
	if len(c.buf) == 0 {
		c.mu.Unlock()
		return false
	}
	size := len(c.buf)
	if size > maxServerBatch {
		size = maxServerBatch
	}
	batch := make([]Entry, size)
	copy(batch, c.buf[:size])
	c.buf = c.buf[size:]
	c.mu.Unlock()

	if c.ship(batch) {
		c.requeue(batch)
		return false
	}
	return true
}

func (c *Client) requeue(batch []Entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.buf = append(batch, c.buf...)
	if over := len(c.buf) - c.cfg.BufferCap; over > 0 {
		c.buf = c.buf[over:]
		c.dropped += over
	}
}

func (c *Client) ship(entries []Entry) bool {
	body, err := json.Marshal(map[string][]Entry{"entries": entries})
	if err != nil {
		return false
	}
	request, err := http.NewRequest(http.MethodPost, c.cfg.URL+"/ingest", bytes.NewReader(body))
	if err != nil {
		return false
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	response, err := c.http.Do(request)
	if err != nil {
		return true
	}
	io.Copy(io.Discard, io.LimitReader(response.Body, 1<<20))
	response.Body.Close()
	if response.StatusCode == http.StatusTooManyRequests {
		c.retryAt = time.Now().Add(retryDelay(response))
		return true
	}
	return response.StatusCode >= 500
}

func retryDelay(response *http.Response) time.Duration {
	seconds, err := strconv.Atoi(response.Header.Get("Retry-After"))
	if err != nil || seconds <= 0 {
		return 0
	}
	delay := time.Duration(seconds) * time.Second
	if delay > maxRetryAfter {
		return maxRetryAfter
	}
	return delay
}

func sanitizeMeta(meta map[string]any) map[string]any {
	if len(meta) == 0 {
		return nil
	}
	out := make(map[string]any, len(meta))
	for key, value := range meta {
		out[key] = sanitizeValue(value)
	}
	return out
}

func sanitizeValue(value any) any {
	switch typed := value.(type) {
	case nil, bool, string, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, json.Number, time.Time:
		return typed
	case float32:
		return sanitizeFloat(float64(typed), typed)
	case float64:
		return sanitizeFloat(typed, typed)
	case error:
		return safeString(typed.Error, typed)
	case fmt.Stringer:
		return safeString(typed.String, typed)
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, nested := range typed {
			out[key] = sanitizeValue(nested)
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for i, nested := range typed {
			out[i] = sanitizeValue(nested)
		}
		return out
	}
	if _, err := json.Marshal(value); err == nil {
		return value
	}
	return fmt.Sprintf("%v", value)
}

func sanitizeFloat(f float64, value any) any {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return fmt.Sprintf("%v", value)
	}
	return value
}

func safeString(fn func() string, value any) (out string) {
	defer func() {
		if recover() != nil {
			out = fmt.Sprintf("%v", value)
		}
	}()
	return fn()
}
