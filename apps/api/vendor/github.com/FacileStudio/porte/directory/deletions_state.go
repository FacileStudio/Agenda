package directory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// deletionState is the persisted record: the newest erasure time handled in
// order, and every subject handled with the moment, so a notice that arrives
// by push and again by poll, or twice by push, reaches the callback once.
type deletionState struct {
	Cursor time.Time            `json:"cursor"`
	Done   map[string]time.Time `json:"done"`
}

// since is where the next poll starts. It backs up a minute, so a notice
// that committed after a later one is still listed; handled subjects are
// skipped, and paging never depends on the backup.
func (s *deletionState) since() time.Time {
	if s.Cursor.IsZero() {
		return time.Now().Add(-DeletionRetention)
	}
	return s.Cursor.Add(-overlap)
}

func (s *deletionState) done(subject string) bool {
	_, ok := s.Done[subject]
	return ok
}

// record marks subject handled at now and forgets subjects Registre no longer
// lists.
func (s *deletionState) record(subject string, now time.Time) {
	s.Done[subject] = now
	for done, when := range s.Done {
		if now.Sub(when) > DeletionRetention+overlap {
			delete(s.Done, done)
		}
	}
}

// load reads the state once; later calls reuse it. The caller holds d.mu.
func (d *Deletions) load(ctx context.Context) (*deletionState, error) {
	if d.state != nil {
		return d.state, nil
	}
	state := &deletionState{Done: map[string]time.Time{}}
	if d.State != nil {
		raw, err := d.State.Load(ctx)
		if err != nil {
			return nil, fmt.Errorf("directory: load the deletion state: %w", err)
		}
		if err := state.decode(raw); err != nil {
			return nil, err
		}
	}
	d.state = state
	return state, nil
}

func (s *deletionState) decode(raw []byte) error {
	if len(raw) == 0 {
		return nil
	}
	if err := json.Unmarshal(raw, s); err != nil {
		return fmt.Errorf("directory: decode the deletion state: %w", err)
	}
	if s.Done == nil {
		s.Done = map[string]time.Time{}
	}
	return nil
}

func (d *Deletions) save(ctx context.Context, state *deletionState) error {
	if d.State == nil {
		return nil
	}
	raw, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("directory: encode the deletion state: %w", err)
	}
	if err := d.State.Save(ctx, raw); err != nil {
		return fmt.Errorf("directory: save the deletion state: %w", err)
	}
	return nil
}
