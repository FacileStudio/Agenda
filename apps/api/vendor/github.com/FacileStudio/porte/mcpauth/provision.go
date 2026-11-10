package mcpauth

import (
	"context"
	"errors"
	"fmt"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/oidc"
	"github.com/FacileStudio/porte/oidc/jwt"
)

// Provisioner creates the local account for a verified token. *oidc.Kit is
// the implementation: its Config names the issuer the account is filed
// under, and ProvisionBearer runs the web login's write path.
type Provisioner interface {
	Config() porte.Config
	ProvisionBearer(ctx context.Context, claims jwt.Claims) (porte.Identity, error)
}

var _ Provisioner = (*oidc.Kit)(nil)

// validateProvision refuses a Provision without a kit, and a kit filing
// identities under another issuer than the one tokens are checked against:
// every call would then create an account the next lookup cannot find.
func (c Config) validateProvision() error {
	if !c.Provision {
		return nil
	}
	if c.Accounts == nil {
		return errors.New("porte/mcpauth: Provision needs Accounts, the app's porte OIDC kit")
	}
	if issuer := c.Accounts.Config().Issuer; issuer != c.Issuer {
		return fmt.Errorf("porte/mcpauth: Accounts files identities under %q, not the issuer %q", issuer, c.Issuer)
	}
	return nil
}

// provision creates the account of a subject seen for the first time. A
// refusal (no verified email, the app's store declining the person) is an
// invalid token, like any other; a store that failed is unavailability.
func (v verifier) provision(ctx context.Context, claims jwt.Claims) (porte.Identity, error) {
	identity, err := v.cfg.Accounts.ProvisionBearer(ctx, claims)
	switch {
	case err == nil:
		return identity, nil
	case errors.Is(err, oidc.ErrProvisionRefused):
		return porte.Identity{}, v.refuse(ctx, err)
	}
	return porte.Identity{}, v.unavailable(ctx, err)
}
