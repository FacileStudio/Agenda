package antenne

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"gopkg.in/yaml.v3"
)

// ErrNotConnected is returned by EmitNow when no live connection exists.
var ErrNotConnected = errors.New("antenne: not connected")

// ErrClosed is returned when a disconnected client is asked to connect again.
// Disconnect is terminal: build a new client instead.
var ErrClosed = errors.New("antenne: client is closed")

type Config struct {
	App        string      `yaml:"app" json:"app"`
	Instance   string      `yaml:"instance" json:"instance"`
	Secret     string      `yaml:"secret" json:"secret"`
	InstanceID string      `yaml:"instance_id" json:"instance_id"`
	Events     EventConfig `yaml:"events" json:"events"`
}

type EventConfig struct {
	Emit   []string `yaml:"emit" json:"emit"`
	Listen []string `yaml:"listen" json:"listen"`
}

func LoadConfig(path string) (*Config, error) {
	if path == "" {
		found := findConfigFile()
		if found == "" {
			return nil, fmt.Errorf("no antenne.yaml found")
		}
		path = found
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	if cfg.App == "" || cfg.Instance == "" || cfg.Secret == "" {
		return nil, fmt.Errorf("config: app, instance, and secret are required")
	}
	return &cfg, nil
}

func findConfigFile() string {
	for _, name := range []string{"antenne.yaml", "antenne.yml"} {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}
	if p := os.Getenv("ANTENNE_CONFIG_PATH"); p != "" {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

type EventMeta struct {
	ID        string `json:"id"`
	Channel   string `json:"channel"`
	Offset    int64  `json:"offset"`
	Timestamp int64  `json:"timestamp"`
	Sender    string `json:"sender"`
}

type EventHandler func(payload json.RawMessage, meta EventMeta)

type ClientOption func(*Client)

func WithOnConnect(fn func()) ClientOption {
	return func(c *Client) { c.onConnect = fn }
}

func WithOnDisconnect(fn func()) ClientOption {
	return func(c *Client) { c.onDisconnect = fn }
}

func WithOnError(fn func(error)) ClientOption {
	return func(c *Client) { c.onError = fn }
}

func WithMaxReconnect(n int) ClientOption {
	return func(c *Client) { c.maxReconnect = n }
}

type Client struct {
	config           *Config
	conn             *websocket.Conn
	token            string
	appID            string
	epoch            string
	connected        bool
	closed           bool
	maxReconnect     int
	reconnectAttempt int
	handlers         map[string][]EventHandler
	offsets          map[string]int64
	mu               sync.RWMutex
	writeMu          sync.Mutex
	done             chan struct{}
	closeOnce        sync.Once
	onConnect        func()
	onDisconnect     func()
	onError          func(error)
	pending          [][]byte
}

func NewClient(config *Config, opts ...ClientOption) *Client {
	c := &Client{
		config:       config,
		maxReconnect: 20,
		handlers:     make(map[string][]EventHandler),
		offsets:      make(map[string]int64),
		done:         make(chan struct{}),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func NewClientFromYAML(path string, opts ...ClientOption) (*Client, error) {
	cfg, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}
	return NewClient(cfg, opts...), nil
}

func (c *Client) Connect(ctx context.Context) error {
	if c.isClosed() {
		return ErrClosed
	}
	if err := c.register(ctx); err != nil {
		return fmt.Errorf("registration failed: %w", err)
	}
	if err := c.openWebSocket(ctx); err != nil {
		return fmt.Errorf("websocket connect failed: %w", err)
	}
	return nil
}

// Disconnect closes the connection for good. It is terminal and idempotent: no
// reconnect already in flight may outlive it. A client that could resurrect
// itself after being discarded would re-register under the same identity and
// evict whichever connection its replacement had opened, and the two would then
// take turns kicking each other off the pool.
func (c *Client) Disconnect() {
	c.mu.Lock()
	conn := c.conn
	c.connected = false
	c.closed = true
	c.conn = nil
	c.mu.Unlock()

	c.closeOnce.Do(func() { close(c.done) })

	if conn != nil {
		c.writeMu.Lock()
		conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		c.writeMu.Unlock()
		conn.Close()
	}
}

func (c *Client) isClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

func (c *Client) Emit(channel string, payload any) error {
	id := generateID()
	msg := map[string]any{
		"type":      "event",
		"id":        id,
		"channel":   channel,
		"payload":   payload,
		"timestamp": time.Now().UnixMilli(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	c.mu.RLock()
	connected := c.connected
	conn := c.conn
	c.mu.RUnlock()

	if connected && conn != nil {
		c.writeMu.Lock()
		err := conn.WriteMessage(websocket.TextMessage, data)
		c.writeMu.Unlock()
		return err
	}

	c.mu.Lock()
	c.pending = append(c.pending, data)
	c.mu.Unlock()
	return nil
}

// EmitNow sends immediately or returns an error; it never buffers. Callers
// that provide their own durability (e.g. an outbox drainer) must use this
// instead of Emit, whose in-memory buffering would let them mark an event as
// sent when it only sits in RAM.
func (c *Client) EmitNow(channel string, payload any) error {
	msg := map[string]any{
		"type":      "event",
		"id":        generateID(),
		"channel":   channel,
		"payload":   payload,
		"timestamp": time.Now().UnixMilli(),
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	c.mu.RLock()
	connected := c.connected
	conn := c.conn
	c.mu.RUnlock()

	if !connected || conn == nil {
		return ErrNotConnected
	}

	c.writeMu.Lock()
	err = conn.WriteMessage(websocket.TextMessage, data)
	c.writeMu.Unlock()
	return err
}

func (c *Client) Listen(channel string, handler EventHandler) func() {
	c.mu.Lock()
	c.handlers[channel] = append(c.handlers[channel], handler)
	c.mu.Unlock()

	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		handlers := c.handlers[channel]
		for i := range handlers {
			if &handlers[i] == &handler {
				c.handlers[channel] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
	}
}

func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// Identity returns the app identity used by the pool for echo filtering:
// "app" alone, or "app:instance_id" when an instance id is configured.
func (c *Client) Identity() string {
	if c.config.InstanceID != "" {
		return c.config.App + ":" + c.config.InstanceID
	}
	return c.config.App
}

func (c *Client) register(ctx context.Context) error {
	regURL := c.config.Instance + "/api/antenne/register"
	body, _ := json.Marshal(map[string]any{
		"app":         c.config.App,
		"instance_id": c.config.InstanceID,
		"secret":      c.config.Secret,
		"events":      c.config.Events,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", regURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("registration failed: %d %s", resp.StatusCode, string(body))
	}

	var result struct {
		Token string `json:"token"`
		AppID string `json:"app_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	c.token = result.Token
	c.appID = result.AppID
	return nil
}

func (c *Client) openWebSocket(ctx context.Context) error {
	wsURL := strings.Replace(c.config.Instance, "http", "ws", 1) +
		"/api/antenne/ws?token=" + url.QueryEscape(c.token)

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return err
	}

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		conn.Close()
		return ErrClosed
	}
	c.conn = conn
	c.connected = true
	c.reconnectAttempt = 0

	channels := make(map[string]map[string]int64)
	for _, ch := range c.config.Events.Listen {
		channels[ch] = map[string]int64{"last_offset": c.offsets[ch]}
	}
	for ch := range c.handlers {
		if _, ok := channels[ch]; !ok {
			channels[ch] = map[string]int64{"last_offset": c.offsets[ch]}
		}
	}

	subMsg, _ := json.Marshal(map[string]any{
		"type":     "subscribe",
		"channels": channels,
	})

	c.writeMu.Lock()
	conn.WriteMessage(websocket.TextMessage, subMsg)
	for _, msg := range c.pending {
		conn.WriteMessage(websocket.TextMessage, msg)
	}
	c.writeMu.Unlock()
	c.pending = nil
	c.mu.Unlock()

	if c.onConnect != nil {
		c.onConnect()
	}

	go c.readLoop()
	go c.pingLoop()

	return nil
}

func (c *Client) readLoop() {
	for {
		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()
		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			c.mu.Lock()
			c.connected = false
			closed := c.closed
			c.mu.Unlock()

			if c.onDisconnect != nil {
				c.onDisconnect()
			}
			if !closed {
				go c.scheduleReconnect()
			}
			return
		}

		c.handleMessage(message)
	}
}

func (c *Client) pingLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.mu.RLock()
			conn := c.conn
			connected := c.connected
			c.mu.RUnlock()
			if !connected || conn == nil {
				return
			}
			msg, _ := json.Marshal(map[string]string{"type": "ping"})
			c.writeMu.Lock()
			conn.WriteMessage(websocket.TextMessage, msg)
			c.writeMu.Unlock()
		case <-c.done:
			return
		}
	}
}

func (c *Client) handleMessage(raw []byte) {
	var msg map[string]any
	if err := json.Unmarshal(raw, &msg); err != nil {
		return
	}

	msgType, _ := msg["type"].(string)

	switch msgType {
	case "welcome":
		c.mu.Lock()
		c.epoch, _ = msg["epoch"].(string)
		c.mu.Unlock()

	case "event":
		sender, _ := msg["sender"].(string)
		if sender == c.Identity() {
			return
		}

		channel, _ := msg["channel"].(string)
		offsetFloat, _ := msg["offset"].(float64)
		offset := int64(offsetFloat)
		timestampFloat, _ := msg["timestamp"].(float64)
		id, _ := msg["id"].(string)

		c.mu.Lock()
		c.offsets[channel] = offset
		c.mu.Unlock()

		payloadBytes, _ := json.Marshal(msg["payload"])

		meta := EventMeta{
			ID:        id,
			Channel:   channel,
			Offset:    offset,
			Timestamp: int64(timestampFloat),
			Sender:    sender,
		}

		c.mu.RLock()
		handlers := make([]EventHandler, len(c.handlers[channel]))
		copy(handlers, c.handlers[channel])
		c.mu.RUnlock()

		for _, h := range handlers {
			h(payloadBytes, meta)
		}

		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()
		if conn != nil {
			ack, _ := json.Marshal(map[string]any{
				"type":    "ack",
				"channel": channel,
				"offset":  offset,
			})
			c.writeMu.Lock()
			conn.WriteMessage(websocket.TextMessage, ack)
			c.writeMu.Unlock()
		}

	case "error":
		code, _ := msg["code"].(string)
		message, _ := msg["message"].(string)
		if c.onError != nil {
			c.onError(fmt.Errorf("pool error [%s]: %s", code, message))
		}
	}
}

// TransientError marks a failure the client is going to retry on its own.
//
// It exists because the reverse is what consumers were doing: logging every
// failed reconnect attempt at error level. A proxy restart takes a few seconds
// and produces three of them, so a deploy painted the dashboard red for an
// incident that resolved itself — and a stream that cries wolf on every deploy
// is one people learn to scroll past, which is the real cost.
//
// Log it at warn (or not at all) and act on the terminal error instead: the
// client reports "max reconnection attempts reached" as a plain error once it
// gives up, and that one means something.
type TransientError struct {
	// Attempt is the reconnection attempt that failed, starting at 1.
	Attempt int
	Err     error
}

func (e *TransientError) Error() string {
	return fmt.Sprintf("%v (reconnect attempt %d, retrying)", e.Err, e.Attempt)
}

func (e *TransientError) Unwrap() error { return e.Err }

func (c *Client) scheduleReconnect() {
	c.mu.RLock()
	attempt := c.reconnectAttempt
	maxReconnect := c.maxReconnect
	closed := c.closed
	c.mu.RUnlock()

	if closed {
		return
	}
	if attempt >= maxReconnect {
		if c.onError != nil {
			c.onError(fmt.Errorf("max reconnection attempts reached"))
		}
		return
	}

	delay := time.Duration(math.Min(
		float64(500*time.Millisecond)*math.Pow(2, float64(attempt)),
		float64(30*time.Second),
	)) + time.Duration(float64(500*time.Millisecond)*randFloat())

	c.mu.Lock()
	c.reconnectAttempt++
	c.mu.Unlock()

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-c.done:
		return
	}
	if c.isClosed() {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := c.register(ctx); err != nil {
		if c.onError != nil {
			c.onError(&TransientError{Attempt: attempt + 1, Err: fmt.Errorf("reconnect register: %w", err)})
		}
		c.scheduleReconnect()
		return
	}
	if err := c.openWebSocket(ctx); err != nil {
		if c.onError != nil {
			c.onError(&TransientError{Attempt: attempt + 1, Err: fmt.Errorf("reconnect ws: %w", err)})
		}
		c.scheduleReconnect()
		return
	}
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func randFloat() float64 {
	b := make([]byte, 8)
	rand.Read(b)
	var n uint64
	for i, v := range b {
		n |= uint64(v) << (8 * i)
	}
	return float64(n) / float64(^uint64(0))
}
