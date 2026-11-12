package oidc

import (
	"context"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"
)

// refuseErased stops an erased identity before UpsertFromOIDC can give it an
// account again. A ledger that cannot be read refuses too: a sign-in that
// waits is better than an erasure that silently comes undone.
func (k *Kit) refuseErased(ctx context.Context, claims porte.Claims) error {
	if k.deps.Erased == nil {
		return nil
	}
	erased, err := k.deps.Erased.Erased(ctx, claims.Provider, claims.Subject)
	if err != nil {
		return errors.Internal("failed to check the identity against the erasures", err)
	}
	if erased {
		return errors.New("permission_denied", "this account was deleted", porte.ErrSubjectErased)
	}
	return nil
}
