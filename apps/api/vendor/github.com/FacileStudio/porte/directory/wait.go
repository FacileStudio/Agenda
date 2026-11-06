package directory

import (
	"context"
	"time"
)

// drain drops a nudge that arrived during the debounce: the pull about to
// run already covers it, so a burst costs one pull rather than two.
func (s *Syncer) drain() {
	select {
	case <-s.nudges:
	default:
	}
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
