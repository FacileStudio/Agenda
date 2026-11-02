package keys

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/chi/v5"
)

// Manager coordinates API key registration, revocation, and authentication middleware.
type Manager struct {
	store         Store
	servicePrefix string
}

// NewManager builds an API key manager for a named service.
func NewManager(store Store, servicePrefix string) *Manager {
	if servicePrefix == "" {
		servicePrefix = "porte"
	}
	return &Manager{
		store:         store,
		servicePrefix: servicePrefix,
	}
}

// Mount registers the standard /apikeys administrative endpoints onto r.
func (m *Manager) Mount(r chi.Router) {
	r.Get("/apikeys", m.list)
	r.Post("/apikeys", m.create)
	r.Delete("/apikeys/{id}", m.revoke)
}

func (m *Manager) list(w http.ResponseWriter, r *http.Request) {
	appFilter := r.URL.Query().Get("app")
	records, err := m.store.List(r.Context(), appFilter)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	usage, err := m.store.UsageToday(r.Context())
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	keys := make([]Key, 0, len(records))
	for _, record := range records {
		record.UsedToday = usage[record.ID]
		keys = append(keys, record)
	}

	httpjson.WriteJSON(w, http.StatusOK, map[string]any{"keys": keys})
}

func (m *Manager) create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := httpjson.DecodeJSON(w, r, &req); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	if strings.TrimSpace(req.App) == "" {
		httpjson.WriteError(w, errors.Invalid("app is required"))
		return
	}

	kind := req.Kind
	if kind != KindPublic {
		kind = KindSecret
	}

	rawToken, prefix, tokenHash, err := GenerateToken(m.servicePrefix, req.App, kind)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	key := Key{
		App:            req.App,
		Kind:           kind,
		Prefix:         prefix,
		TokenHash:      tokenHash,
		AllowedOrigins: req.AllowedOrigins,
		DailyQuota:     req.DailyQuota,
		CreatedAt:      time.Now().UTC(),
	}

	created, err := m.store.Create(r.Context(), key)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusCreated, CreateResponse{Key: created, Token: rawToken})
}

func (m *Manager) revoke(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		httpjson.WriteError(w, errors.Invalid("id must be an integer"))
		return
	}

	if err := m.store.Revoke(r.Context(), id); err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusNoContent, nil)
}

// Authenticate verifies an API key from Bearer header or key query param and attaches it to Context.
func (m *Manager) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			httpjson.WriteError(w, errors.Unauthorized("missing API key"))
			return
		}

		key, err := m.store.FindByHash(r.Context(), porte.HashToken(token))
		if err != nil || key.RevokedAt != nil {
			httpjson.WriteError(w, errors.Unauthorized("invalid or revoked API key"))
			return
		}

		if err := validateOrigin(r, key); err != nil {
			httpjson.WriteError(w, err)
			return
		}

		if err := m.checkQuota(r.Context(), key); err != nil {
			httpjson.WriteError(w, err)
			return
		}

		_ = m.store.RecordUsage(r.Context(), key.ID, 1)
		next.ServeHTTP(w, r.WithContext(With(r.Context(), key)))
	})
}

func extractToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	return r.URL.Query().Get("key")
}

func validateOrigin(r *http.Request, key Key) error {
	if key.Kind != KindPublic || len(key.AllowedOrigins) == 0 {
		return nil
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		if ref := r.Header.Get("Referer"); ref != "" {
			if parsed, err := url.Parse(ref); err == nil {
				origin = parsed.Scheme + "://" + parsed.Host
			}
		}
	}

	for _, allowed := range key.AllowedOrigins {
		if strings.EqualFold(allowed, origin) {
			return nil
		}
	}

	return errors.Forbidden("origin not allowed for this API key")
}

func (m *Manager) checkQuota(ctx context.Context, key Key) error {
	if key.DailyQuota <= 0 {
		return nil
	}

	usage, err := m.store.UsageToday(ctx)
	if err != nil {
		return err
	}

	if usage[key.ID] >= int64(key.DailyQuota) {
		return errors.RateLimited("daily API quota exceeded")
	}

	return nil
}
