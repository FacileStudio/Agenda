package directory

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// heldAgain asks Holds whether a handled subject belongs to an account again.
func (d *Deletions) heldAgain(ctx context.Context, subject string) (bool, error) {
	if d.Holds == nil {
		return false, nil
	}
	held, err := d.Holds(ctx, d.Client.Issuer, subject)
	if err != nil {
		return false, fmt.Errorf("directory: check whether %s holds an account again: %w", subject, err)
	}
	if held {
		d.logger().Warn("directory: an erased subject holds an account again, erasing it again",
			slog.String("subject", subject))
	}
	return held, nil
}

// Erased reports whether Registre erased the person behind (provider, subject)
// and this app handled it, so the app can refuse to give that identity an
// account again: porte.ErasedSubjects, which porte/oidc's Deps.Erased takes.
//
// It reads State on every call and never the copy Poll and Handler hold, so
// it answers the same on a replica that does not run the receiver, and it
// never waits for a poll in progress. A provider other than Client.Issuer
// (with or without one trailing slash) and a nil State answer false. A
// subject stays erased for at least DeletionRetention after it was last handled.
func (d *Deletions) Erased(ctx context.Context, provider, subject string) (bool, error) {
	if d.State == nil || subject == "" || trimIssuer(provider) != trimIssuer(d.Client.Issuer) {
		return false, nil
	}
	raw, err := d.State.Load(ctx)
	if err != nil {
		return false, fmt.Errorf("directory: load the deletion state: %w", err)
	}
	state := &deletionState{Done: map[string]time.Time{}}
	if err := state.decode(raw); err != nil {
		return false, err
	}
	return state.done(subject), nil
}
