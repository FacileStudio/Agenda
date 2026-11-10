package bulle

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const maxBody = 1 << 20

// ErrBadSegment is returned before any call when a path segment is empty, a
// dot segment, or longer than 200 bytes.
var ErrBadSegment = stderrors.New("bulle: invalid path segment")

// Request is one call to Bulle's API. Segments follow /api/ and are escaped
// one by one, so an id holding a slash or a query stays one segment.
type Request struct {
	Method   string
	Segments []string
	Query    url.Values
	Body     any
}

func (c *Client) target(req Request) (string, error) {
	escaped := make([]string, 0, len(req.Segments))
	for _, segment := range req.Segments {
		if segment == "" || segment == "." || segment == ".." || len(segment) > 200 {
			return "", ErrBadSegment
		}
		escaped = append(escaped, url.PathEscape(segment))
	}
	target := c.base.String() + "/api/" + strings.Join(escaped, "/")
	if encoded := req.Query.Encode(); encoded != "" {
		target += "?" + encoded
	}
	return target, nil
}

// Call sends req as the user on ctx and decodes a 2xx JSON answer into out,
// which may be nil. A 401 retries once after Config.Forget.
func (c *Client) Call(ctx context.Context, req Request, out any) error {
	target, err := c.target(req)
	if err != nil {
		return err
	}
	var payload []byte
	if req.Body != nil {
		if payload, err = json.Marshal(req.Body); err != nil {
			return fmt.Errorf("bulle: encoding the request: %w", err)
		}
	}
	err = c.send(ctx, req.Method, target, payload, out)
	var refused *StatusError
	if c.cfg.Forget != nil && stderrors.As(err, &refused) && refused.Status == http.StatusUnauthorized {
		c.cfg.Forget(ctx)
		err = c.send(ctx, req.Method, target, payload, out)
	}
	return err
}

func (c *Client) send(ctx context.Context, method, target string, payload []byte, out any) error {
	token, err := c.cfg.TokenSource(ctx)
	if err != nil {
		return err
	}
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	request, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return fmt.Errorf("bulle: building the request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnreachable, err)
	}
	defer func() { _ = response.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxBody))
	if err != nil {
		return fmt.Errorf("%w: reading the answer: %w", ErrUnreachable, err)
	}
	return answer(response.StatusCode, raw, out)
}

func answer(status int, raw []byte, out any) error {
	switch {
	case status >= 500:
		return fmt.Errorf("%w: answered %d", ErrUnreachable, status)
	case status == http.StatusPaymentRequired:
		return planRequired(raw)
	case status >= 400:
		fields := envelope(raw)
		return &StatusError{Status: status, Code: fields.Code, Message: fields.Message}
	case status < 200 || status >= 300:
		return fmt.Errorf("%w: answered %d", ErrUnreachable, status)
	case out == nil || len(bytes.TrimSpace(raw)) == 0:
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("%w: unreadable answer: %w", ErrUnreachable, err)
	}
	return nil
}
