package directory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// DeletionsPath is the route an app mounts Deletions.Handler on, and the
// path to set as the app's deletion_uri in Registre. It is never SyncPath: the
// sync handler answers 202 to any POST, so a notice sent there would count
// as delivered while nothing erased anything.
const DeletionsPath = "/api/registre/deletions"

// DeletionRetention is how long Registre lists an erasure. An app that polls
// less often than this can miss a person; the receiver keeps its own record
// of handled subjects for the same span.
const DeletionRetention = 60 * 24 * time.Hour

const overlap = time.Minute

const listLimit = 1000

var errNoCallback = errors.New("directory: Deletions.OnPersonDeleted is nil")

// Deleted is one erased account as Registre lists it.
type Deleted struct {
	Subject   string    `json:"sub"`
	DeletedAt time.Time `json:"deleted_at"`
}

// Deletions lists the accounts Registre erased since a moment, for this app.
// Registre answers at most 1000 per call, oldest first.
func (c Client) Deletions(ctx context.Context, since time.Time) ([]Deleted, error) {
	endpoint := "/api/directory/deletions?since=" + url.QueryEscape(since.UTC().Format(time.RFC3339Nano))
	body, err := c.fetch(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Deleted []Deleted `json:"deleted"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("%w: decode the deletions: %w", ErrUnavailable, err)
	}
	return out.Deleted, nil
}

// StateStore persists the receiver's state between runs, as forfait.Store's
// Export and Import do for plans. Load returns nil, nil when nothing was
// saved yet. The app stores the bytes as they are, in one row or file.
type StateStore interface {
	Load(ctx context.Context) ([]byte, error)
	Save(ctx context.Context, state []byte) error
}

// Deletions erases the people Registre erased. Run exactly one replica's Run
// (a leader, or a single worker): two replicas would race on State and could
// each call the callback for the same person. It lists them from Registre
// (Poll, Run) and, when the app declared a deletion_uri, also accepts
// Registre's signed push (Handler). Both paths hand each person to
// OnPersonDeleted once.
//
// A person counts as handled only after OnPersonDeleted and Erase both
// returned nil and State saved that fact. Any failure leaves the person
// unhandled, so the next poll or Registre's retried push tries again: delivery
// is at least once, and a crash between the callback and the save repeats the
// callback, which must therefore be idempotent.
type Deletions struct {
	Client Client

	// OnPersonDeleted erases the app's own data for the person. provider is
	// Client.Issuer, the value porte stores as an identity's Provider, so the
	// app can still resolve the account from its identity rows: Erase runs
	// after it.
	OnPersonDeleted func(ctx context.Context, provider, subject string) error

	// Erase removes porte's own rows for the identity: pg.Store.EraseIdentity.
	// Nil skips it, for an app whose storage is not porte/pg.
	Erase func(ctx context.Context, provider, subject string) error

	State    StateStore
	Interval time.Duration
	Logger   *slog.Logger

	mu    sync.Mutex
	state *deletionState
	keys  keySet
}

// Run polls at once, then every Interval (DefaultInterval when zero), until
// ctx ends. A failed poll is logged and retried at the next tick.
func (d *Deletions) Run(ctx context.Context) {
	interval := d.Interval
	if interval <= 0 {
		interval = DefaultInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := d.Poll(ctx); err != nil {
			d.logger().Warn("directory: deletion poll failed, will retry", slog.Any("error", err))
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// handle erases one person unless they were handled already, and records it
// in memory. The caller saves, once per page for a poll and once per notice
// for a push, and holds d.mu.
func (d *Deletions) handle(ctx context.Context, subject string) error {
	state, err := d.load(ctx)
	if err != nil {
		return err
	}
	if state.done(subject) {
		return nil
	}
	if err := d.erase(ctx, subject); err != nil {
		return err
	}
	state.record(subject, time.Now())
	return nil
}

// persist saves the state; on failure it forgets the in-memory copy, so the
// next call reloads what was last stored and handles the people again.
func (d *Deletions) persist(ctx context.Context) error {
	if err := d.save(ctx, d.state); err != nil {
		d.state = nil
		return err
	}
	return nil
}

// erase runs the app's callback, then porte's own erasure.
func (d *Deletions) erase(ctx context.Context, subject string) error {
	if err := d.OnPersonDeleted(ctx, d.Client.Issuer, subject); err != nil {
		return fmt.Errorf("directory: erase %s: %w", subject, err)
	}
	if d.Erase == nil {
		return nil
	}
	if err := d.Erase(ctx, d.Client.Issuer, subject); err != nil {
		return fmt.Errorf("directory: erase porte rows of %s: %w", subject, err)
	}
	return nil
}

func (d *Deletions) logger() *slog.Logger {
	if d.Logger != nil {
		return d.Logger
	}
	return slog.Default()
}

func readAll(body io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(body, limit+1))
	if err == nil && int64(len(data)) > limit {
		err = fmt.Errorf("body over %d bytes", limit)
	}
	return data, err
}

func trimIssuer(s string) string { return strings.TrimSuffix(s, "/") }
