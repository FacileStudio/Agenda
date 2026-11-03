package directory

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// DefaultInterval is how often a Syncer pulls without being nudged. It bounds
// how long a missed nudge can leave an app behind.
const DefaultInterval = 5 * time.Minute

// debounce is the shortest gap between two pulls a burst of nudges causes.
// Registre nudges once per change, so an operator adding five people sends
// five; one pull after the last is enough.
const debounce = 2 * time.Second

// Syncer keeps one app's spaces equal to Registre's. Apply is the app's own
// code: it receives the whole snapshot and makes the app's tables match it.
// It must be idempotent, because it runs again whenever anything may have
// changed.
type Syncer struct {
	Client   Client
	Apply    func(ctx context.Context, snapshot Snapshot) error
	Interval time.Duration
	Logger   *slog.Logger

	nudges  chan struct{}
	once    sync.Once
	applied int64
}

// Handler answers Registre's nudge on SyncPath. It accepts any POST with 202
// and only schedules a pull, so it needs no credentials: the worst a forged
// nudge can do is make the app pull a little sooner.
func (s *Syncer) Handler() http.Handler {
	s.init()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.Nudge()
		w.WriteHeader(http.StatusAccepted)
	})
}

// Nudge schedules a pull. It never blocks; nudges arriving while one is
// already pending collapse into it.
func (s *Syncer) Nudge() {
	s.init()
	select {
	case s.nudges <- struct{}{}:
	default:
	}
}

// Run pulls once immediately, then on every interval tick and after every
// nudge, until ctx ends. A failed pull is logged and retried at the next
// trigger; the app keeps serving the last state it applied.
func (s *Syncer) Run(ctx context.Context) {
	s.init()
	ticker := time.NewTicker(s.interval())
	defer ticker.Stop()
	s.pull(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-s.nudges:
			if !sleep(ctx, debounce) {
				return
			}
		}
		s.pull(ctx)
	}
}

// Sync pulls and applies once, skipping a snapshot whose version was already
// applied. It is what Run calls, exported for boot paths and tests.
func (s *Syncer) Sync(ctx context.Context) error {
	snapshot, err := s.Client.Snapshot(ctx)
	if err != nil {
		return err
	}
	if snapshot.Version != 0 && snapshot.Version == s.applied {
		return nil
	}
	if err := s.Apply(ctx, snapshot); err != nil {
		return err
	}
	s.applied = snapshot.Version
	return nil
}

// pull runs Sync and logs the outcome.
func (s *Syncer) pull(ctx context.Context) {
	if err := s.Sync(ctx); err != nil {
		s.logger().Warn("directory: sync failed, keeping the last applied spaces", slog.Any("error", err))
	}
}

// init allocates the nudge channel once.
func (s *Syncer) init() {
	s.once.Do(func() { s.nudges = make(chan struct{}, 1) })
}

// interval is the configured pull interval, or DefaultInterval.
func (s *Syncer) interval() time.Duration {
	if s.Interval > 0 {
		return s.Interval
	}
	return DefaultInterval
}

// logger is the configured logger, or the process default.
func (s *Syncer) logger() *slog.Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return slog.Default()
}

// sleep waits for d or until ctx ends, reporting whether it waited fully.
func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
