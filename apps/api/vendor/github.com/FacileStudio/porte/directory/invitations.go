package directory

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// MaxInvitationsBytes is the largest invitations answer Invitations reads.
// One invitation is a few hundred bytes, so the cap holds hundreds of them.
const MaxInvitationsBytes = 64 << 10

// InvitationsTimeout bounds one Invitations call, whatever deadline the
// caller's context carries: the answer feeds a menu badge, and a page must
// not wait on it.
const InvitationsTimeout = 2 * time.Second

// InvitationsPath is Registre's route for one person's pending invitations.
const InvitationsPath = "/api/directory/invitations"

// ErrNoSubject is an Invitations call without a subject. No request is sent.
var ErrNoSubject = errors.New("directory: a subject is required")

// Invitation is one space invitation waiting for a person. URL is the
// Registre page where they accept or decline it.
type Invitation struct {
	ID        string    `json:"id"`
	SpaceName string    `json:"space_name"`
	Role      string    `json:"role"`
	InvitedBy string    `json:"invited_by"`
	ExpiresAt time.Time `json:"expires_at"`
	URL       string    `json:"url"`
}

type invitationsAnswer struct {
	Count       *int         `json:"count"`
	Invitations []Invitation `json:"invitations"`
}

// Invitations reads the invitations Registre holds for subject, a Registre
// `sub`. An unknown or disabled subject has none. Every failure to reach
// Registre within InvitationsTimeout, a non-200, a body over
// MaxInvitationsBytes, and an answer whose count disagrees with its list or
// whose URL is not on the issuer's origin wraps ErrUnavailable.
func (c Client) Invitations(ctx context.Context, subject string) ([]Invitation, error) {
	if subject == "" {
		return nil, ErrNoSubject
	}
	ctx, cancel := context.WithTimeout(ctx, InvitationsTimeout)
	defer cancel()
	response, err := c.do(ctx, http.MethodGet, InvitationsPath+"?sub="+url.QueryEscape(subject), nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	data, err := readAll(response.Body, MaxInvitationsBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: read the invitations: %w", ErrUnavailable, err)
	}
	invitations, err := c.decodeInvitations(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	return invitations, nil
}

func (c Client) decodeInvitations(data []byte) ([]Invitation, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	var answer invitationsAnswer
	if err := decoder.Decode(&answer); err != nil {
		return nil, fmt.Errorf("decode the invitations: %w", err)
	}
	if decoder.More() {
		return nil, errors.New("trailing data after the invitations")
	}
	if answer.Count == nil || *answer.Count != len(answer.Invitations) {
		return nil, errors.New("the invitations count disagrees with the list")
	}
	origin, err := originOf(c.Issuer)
	if err != nil {
		return nil, err
	}
	for _, invitation := range answer.Invitations {
		if invitation.ID == "" || !onOrigin(invitation.URL, origin) {
			return nil, fmt.Errorf("invitation %q has no id or a link off %s", invitation.ID, origin)
		}
	}
	return answer.Invitations, nil
}

func originOf(issuer string) (string, error) {
	parsed, err := url.Parse(issuer)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return "", fmt.Errorf("the issuer %q is not an absolute http(s) URL", issuer)
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

func onOrigin(link, origin string) bool {
	parsed, err := url.Parse(link)
	if err != nil {
		return false
	}
	return parsed.Scheme+"://"+parsed.Host == origin && strings.HasPrefix(parsed.Path, "/")
}
