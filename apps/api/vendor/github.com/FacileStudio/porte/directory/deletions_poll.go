package directory

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var errSameInstant = fmt.Errorf("directory: %d or more deletion notices share one timestamp, "+
	"the rest cannot be listed", listLimit)

// Poll lists Registre's erasures since the saved cursor, page by page, and
// handles each person. A person who fails does not block the others: the rest
// of the page is still handled, the cursor never moves to or past the earliest
// failure, and the errors come back joined. The next poll lists the failed
// person again and skips the ones already handled.
func (d *Deletions) Poll(ctx context.Context) error {
	if d.OnPersonDeleted == nil {
		return errNoCallback
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	state, err := d.load(ctx)
	if err != nil {
		return err
	}
	result := d.drain(ctx, state.since())
	if d.state == nil {
		return errors.Join(result.errs...)
	}
	if cursor := cursorAfter(result.handled, result.failed); cursor.After(state.Cursor) {
		state.Cursor = cursor
		if err := d.persist(ctx); err != nil {
			result.errs = append(result.errs, err)
		}
	}
	return errors.Join(result.errs...)
}

type pollResult struct {
	handled []time.Time
	failed  time.Time
	errs    []error
}

func (r *pollResult) fail(at time.Time, err error) {
	r.errs = append(r.errs, err)
	if r.failed.IsZero() || at.Before(r.failed) {
		r.failed = at
	}
}

// drain walks the pages from since. It reports the times of the people it
// handled, the earliest failure (zero when none) and every error.
func (d *Deletions) drain(ctx context.Context, since time.Time) pollResult {
	var result pollResult
	for {
		list, err := d.Client.Deletions(ctx, since)
		if err != nil {
			result.errs = append(result.errs, err)
			return result
		}
		d.handleAll(ctx, list, &result)
		if err := d.persist(ctx); err != nil {
			result.errs = append(result.errs, err)
			return result
		}
		next, more, err := nextPage(list, since)
		if err != nil {
			result.errs = append(result.errs, err)
		}
		if !more {
			return result
		}
		since = next
	}
}

// nextPage says where the following page starts, and whether there is one.
// A full page means more may follow; if its last time gives no progress, the
// rest cannot be reached.
func nextPage(list []Deleted, since time.Time) (time.Time, bool, error) {
	if len(list) < listLimit {
		return since, false, nil
	}
	next := list[len(list)-1].DeletedAt.Add(-time.Microsecond)
	if !next.After(since) {
		return since, false, errSameInstant
	}
	return next, true, nil
}

func (d *Deletions) handleAll(ctx context.Context, list []Deleted, result *pollResult) {
	for _, person := range list {
		if err := d.handle(ctx, person.Subject); err != nil {
			result.fail(person.DeletedAt, err)
			continue
		}
		result.handled = append(result.handled, person.DeletedAt)
	}
}

// cursorAfter is the newest handled time strictly before the earliest failure.
func cursorAfter(handled []time.Time, failed time.Time) time.Time {
	var cursor time.Time
	for _, at := range handled {
		if (failed.IsZero() || at.Before(failed)) && at.After(cursor) {
			cursor = at
		}
	}
	return cursor
}
