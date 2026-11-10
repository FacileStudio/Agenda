package exchange

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/FacileStudio/porte"
)

const (
	exchangeGrant   = "urn:ietf:params:oauth:grant-type:token-exchange"
	accessTokenType = "urn:ietf:params:oauth:token-type:access_token"
	maxBody         = 1 << 20
	fallbackLife    = time.Minute
)

var errGrantRejected = stderrors.New("exchange: invalid_grant")

type tokenReply struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Error        string `json:"error"`
}

type registre struct {
	cfg *Config

	mu       sync.Mutex
	endpoint string
}

func (r *registre) tokenEndpoint(ctx context.Context) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.endpoint != "" {
		return r.endpoint, nil
	}
	target := strings.TrimRight(r.cfg.Issuer, "/") + "/.well-known/openid-configuration"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", err
	}
	response, err := r.cfg.Client.Do(request)
	if err != nil {
		return "", fmt.Errorf("%w: discovery: %w", ErrUnavailable, err)
	}
	defer func() { _ = response.Body.Close() }()
	var document struct {
		TokenEndpoint string `json:"token_endpoint"`
	}
	decodeErr := json.NewDecoder(io.LimitReader(response.Body, maxBody)).Decode(&document)
	if response.StatusCode != http.StatusOK || decodeErr != nil || document.TokenEndpoint == "" {
		return "", fmt.Errorf("%w: discovery answered %d with no token endpoint", ErrUnavailable, response.StatusCode)
	}
	r.endpoint = document.TokenEndpoint
	return r.endpoint, nil
}

func (r *registre) post(ctx context.Context, form url.Values) (tokenReply, error) {
	endpoint, err := r.tokenEndpoint(ctx)
	if err != nil {
		return tokenReply{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenReply{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")
	request.SetBasicAuth(url.QueryEscape(r.cfg.ClientID), url.QueryEscape(r.cfg.ClientSecret))
	response, err := r.cfg.Client.Do(request)
	if err != nil {
		return tokenReply{}, fmt.Errorf("%w: %w", ErrUnavailable, err)
	}
	defer func() { _ = response.Body.Close() }()
	var reply tokenReply
	decodeErr := json.NewDecoder(io.LimitReader(response.Body, maxBody)).Decode(&reply)
	return reply, replyError(response.StatusCode, reply, decodeErr)
}

func replyError(status int, reply tokenReply, decodeErr error) error {
	switch {
	case status >= 500 || (status == http.StatusOK && (decodeErr != nil || reply.AccessToken == "")):
		return fmt.Errorf("%w: the token endpoint answered %d", ErrUnavailable, status)
	case status == http.StatusOK:
		return nil
	case reply.Error == "invalid_grant":
		return errGrantRejected
	case reply.Error == "":
		return &RefusedError{Status: status, Code: "unknown"}
	}
	return &RefusedError{Status: status, Code: reply.Error}
}

func (r *registre) expiry(seconds int64) time.Time {
	if seconds <= 0 {
		return r.cfg.Now().Add(fallbackLife)
	}
	return r.cfg.Now().Add(time.Duration(seconds) * time.Second)
}

func (r *registre) refresh(ctx context.Context, refreshToken string) (porte.TokenSet, error) {
	reply, err := r.post(ctx, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refreshToken}})
	if err != nil {
		return porte.TokenSet{}, err
	}
	next := reply.RefreshToken
	if next == "" {
		next = refreshToken
	}
	return porte.TokenSet{AccessToken: reply.AccessToken, RefreshToken: next, Expiry: r.expiry(reply.ExpiresIn)}, nil
}

func (r *registre) exchange(ctx context.Context, subject string, target Target) (cachedToken, error) {
	reply, err := r.post(ctx, url.Values{
		"grant_type":         {exchangeGrant},
		"subject_token":      {subject},
		"subject_token_type": {accessTokenType},
		"audience":           {target.Audience},
		"scope":              {strings.Join(target.Scopes, " ")},
	})
	if err != nil {
		return cachedToken{}, err
	}
	return cachedToken{value: reply.AccessToken, expiry: r.expiry(reply.ExpiresIn)}, nil
}
