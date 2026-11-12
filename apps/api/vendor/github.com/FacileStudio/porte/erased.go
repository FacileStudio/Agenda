package porte

import (
	"context"
	"errors"
)

// ErrSubjectErased is the cause porte/oidc wraps when it refuses a sign-in
// or an account for an identity the app already erased. The error it returns
// is a tronc permission_denied (403); errors.Is(err, ErrSubjectErased) tells
// it apart from every other refusal.
var ErrSubjectErased = errors.New("porte: this identity was erased")

// ErasedSubjects answers whether an identity belongs to a person the app has
// erased. porte/directory's *Deletions implements it from its StateStore.
type ErasedSubjects interface {
	Erased(ctx context.Context, provider, subject string) (bool, error)
}
