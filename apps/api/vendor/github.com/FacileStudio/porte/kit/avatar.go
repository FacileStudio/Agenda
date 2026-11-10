package kit

import (
	"context"
	"strings"

	"github.com/FacileStudio/tronc/errors"
)

// PictureURL keeps an OIDC picture claim only when it is an https URL.
// Registre publishes a stable <issuer>/avatars/<subject> for every person;
// anything else, a data: placeholder or plain http, reads as no picture and
// the client draws the initials.
func PictureURL(claim string) string {
	if strings.HasPrefix(strings.ToLower(claim), "https://") {
		return claim
	}
	return ""
}

// BackfillPictures gives every account linked to issuer the picture URL its
// next sign-in would write, <issuer>/avatars/<subject>, and returns how many
// rows it filled. It fills empty pictures only, so it is idempotent and never
// overwrites what a sign-in stored. A trailing slash on the issuer or on the
// stored provider is ignored, and a non-https issuer fills nothing.
//
// It is a URL, not a download: porte/avatarfs is for apps that keep the bytes.
func (s *Store) BackfillPictures(ctx context.Context, issuer string) (int64, error) {
	base := strings.TrimRight(issuer, "/")
	prefix := PictureURL(base + "/avatars/")
	if prefix == "" {
		return 0, nil
	}
	spellings := providerSpellings(base)
	result, err := s.DB.ExecContext(ctx, s.Columns.resolved().backfillSQL(), prefix, spellings[0], spellings[1])
	if err != nil {
		return 0, errors.Internal("failed to backfill pictures", err)
	}
	filled, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Internal("failed to count the backfilled pictures", err)
	}
	return filled, nil
}
