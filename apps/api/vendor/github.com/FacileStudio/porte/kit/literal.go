package kit

import (
	"log/slog"
	"sync/atomic"
)

var literalWarned atomic.Bool

func (s *Store) logger() *slog.Logger {
	if s.Logger == nil {
		return slog.Default()
	}
	return s.Logger
}

// warnLiteral logs, once per process, that a store NewStore did not build is
// writing an account, so a GORM table's timestamps go NULL without a trace.
func (s *Store) warnLiteral() {
	if s.built || !literalWarned.CompareAndSwap(false, true) {
		return
	}
	s.logger().Warn("porte/kit: this Store was built as a literal; created_at and updated_at columns "+
		"without a database default will be NULL; build it with kit.NewStore",
		slog.String("table", s.Columns.resolved().Table))
}
