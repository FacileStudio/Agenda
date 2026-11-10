package forfait

import (
	"log/slog"
	"sync"
)

// Reset forgets every applied value and its versions, so the Store fails
// open again until the next authoritative snapshot, which it then accepts
// whatever its versions.
//
// It is the recovery step after Registre's database is restored from a
// backup. A restore can lower EntitlementsRevision, and Apply then refuses
// every snapshot as older than the state held, logging a warning each time.
// Call Reset, and delete the state saved from Export, or the next boot's
// Import brings the same refusals back. The next pull then applies normally.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.spaces, s.people, s.version, s.revision = nil, nil, 0, 0
	s.warned = sync.Once{}
}

// logger is the configured logger, or the process default.
func (s *Store) logger() *slog.Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return slog.Default()
}
