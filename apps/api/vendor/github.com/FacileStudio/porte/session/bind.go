package session

import (
	"context"

	"github.com/FacileStudio/porte"
)

// bindToCaller binds an unbound named token to the request's caller, as
// IssueFor describes. Any other template is returned unchanged.
func (m *Manager) bindToCaller(ctx context.Context, template porte.Session) porte.Session {
	caller, ok := porte.From(ctx)
	if !ok || template.Label == "" || template.Provider != "" || template.IssuedBy != 0 || caller.UserID != template.UserID {
		return template
	}
	switch {
	case caller.SessionID != 0:
		template.IssuedBy = caller.SessionID
	case caller.Subject != "":
		template.Provider, template.Subject = m.cfg.Issuer, caller.Subject
	}
	return template
}
