package exchange

import (
	stderrors "errors"
	"fmt"

	"github.com/FacileStudio/tronc/errors"
)

// ErrReconnect means the user holds no Registre grant the app can use: no
// stored refresh token, or one Registre no longer accepts. Only a new sign-in
// fixes it. It is a tronc envelope (failed_precondition, 412), so a handler
// that writes it with httpjson.WriteError tells the user what to do.
var ErrReconnect = errors.Failed("Your Facile session has expired. Sign in again to reconnect it.")

// ErrNoUser means the context carries no authenticated porte identity, so a
// [Source] has nobody to act for.
var ErrNoUser = errors.Unauthorized("not authenticated")

// ErrUnavailable is wrapped by every failure to reach Registre: a network
// error, a timeout, a 5xx or an unreadable answer.
var ErrUnavailable = stderrors.New("exchange: registre is unavailable")

// RefusedError is Registre refusing a request for a reason other than a dead
// grant: invalid_scope when the app's seed entry does not grant the scopes,
// unauthorized_client or invalid_target when it may not exchange for the
// audience, invalid_client when the app's credentials are wrong.
type RefusedError struct {
	Status int
	Code   string
}

func (e *RefusedError) Error() string {
	return fmt.Sprintf("exchange: registre refused the request (%d %s)", e.Status, e.Code)
}
