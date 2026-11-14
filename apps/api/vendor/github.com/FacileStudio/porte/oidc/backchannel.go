package oidc

import (
	"log/slog"
	"strings"

	"github.com/FacileStudio/tronc/errors"
)

// keyFetchFailure is how go-oidc's Verify starts the error of a key set it
// could not fetch: unreachable, a non-200 answer or an unreadable body. The
// signature is checked before any claim, so no token content precedes it, and
// Verify formats the cause with %v, so the text is all that survives.
const keyFetchFailure = "failed to verify signature: fetching keys"

// logoutRefusal is the answer to a logout token that did not verify. When the
// provider's keys could not be fetched nothing was checked, so it is a 503: the
// failure is the app's, not the token's, and a provider that retries can try
// again. Registre delivers a logout once today, so the 503 does not yet save
// that logout. Every other failure is the token's own and stays a 401.
func (k *Kit) logoutRefusal(err error) error {
	if !strings.HasPrefix(err.Error(), keyFetchFailure) {
		return errors.Unauthorized("invalid logout token")
	}
	k.logger.Warn("porte: back-channel logout could not fetch the provider keys", slog.Any("error", err))
	return errors.Unavailable("could not fetch the provider keys")
}
