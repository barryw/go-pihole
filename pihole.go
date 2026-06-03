package pihole

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// defaultTimeout bounds every HTTP request so a stalled connection (e.g. PiHole
// mid-restart) cannot hang a goroutine — and therefore a Terraform apply —
// forever.
const defaultTimeout = 30 * time.Second

type Client struct {
	baseURL    string
	password   string
	httpClient *http.Client

	// mu guards sid and generation. generation is bumped on every successful
	// authentication so concurrent callers can tell whether the session they
	// used has already been refreshed by another goroutine.
	mu         sync.Mutex
	sid        string
	generation uint64

	// authMu serializes (re-)authentication so that a burst of concurrent 401s
	// results in a single auth request rather than one per goroutine.
	authMu sync.Mutex
}

func NewClient(baseURL, password string) (*Client, error) {
	if baseURL == "" {
		return nil, errors.New("baseURL is required")
	}
	if password == "" {
		return nil, errors.New("password is required")
	}
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL:    baseURL,
		password:   password,
		httpClient: &http.Client{Timeout: defaultTimeout},
	}, nil
}

type apiErrorResponse struct {
	Error *struct {
		Key     string `json:"key"`
		Message string `json:"message"`
		Hint    string `json:"hint"`
	} `json:"error"`
}

// Authenticate establishes a session with the PiHole API. Calling it once
// eagerly after construction avoids an authentication storm when Terraform
// fires many concurrent requests at a freshly-created client. If not called
// explicitly, authentication happens automatically on the first request.
func (c *Client) Authenticate(ctx context.Context) error {
	return c.ensureAuthenticated(ctx)
}

const (
	maxRetries    = 5
	retryBaseWait = 2 * time.Second
)

// session returns the current session id and its generation under lock.
func (c *Client) session() (sid string, generation uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sid, c.generation
}

// setSession stores a new session id and advances the generation counter.
func (c *Client) setSession(sid string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sid = sid
	c.generation++
}

// ensureAuthenticated authenticates if there is no session yet. It is safe to
// call concurrently: only one goroutine performs the auth, the rest observe
// the established session and return.
func (c *Client) ensureAuthenticated(ctx context.Context) error {
	if sid, _ := c.session(); sid != "" {
		return nil
	}
	c.authMu.Lock()
	defer c.authMu.Unlock()
	if sid, _ := c.session(); sid != "" {
		return nil
	}
	return c.authenticate(ctx)
}

// reauthenticate refreshes the session after a 401, but only if no other
// goroutine has already refreshed it since the caller made its request
// (detected via the generation counter). This collapses a burst of concurrent
// 401s into a single re-authentication.
func (c *Client) reauthenticate(ctx context.Context, usedGeneration uint64) error {
	c.authMu.Lock()
	defer c.authMu.Unlock()
	if _, gen := c.session(); gen != usedGeneration {
		// Another goroutine already re-authenticated; use its session.
		return nil
	}
	return c.authenticate(ctx)
}

// doRequest executes an authenticated request, retrying on transient network
// errors and re-authenticating on 401. The body is passed as bytes (not an
// io.Reader) so it can be safely re-sent on every retry attempt.
func (c *Client) doRequest(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	if err := c.ensureAuthenticated(ctx); err != nil {
		return nil, err
	}

	for attempt := 0; attempt <= maxRetries; attempt++ {
		sid, generation := c.session()

		resp, err := c.executeRequest(ctx, method, path, body, sid)
		if err != nil {
			if isTransientError(err) && attempt < maxRetries {
				if werr := waitWithContext(ctx, backoff(attempt)); werr != nil {
					return nil, werr
				}
				continue
			}
			return nil, err
		}

		if resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			if err := c.reauthenticate(ctx, generation); err != nil {
				if isTransientError(err) && attempt < maxRetries {
					if werr := waitWithContext(ctx, backoff(attempt)); werr != nil {
						return nil, werr
					}
					continue
				}
				return nil, err
			}
			// Retry with the refreshed session.
			continue
		}

		return resp, nil
	}

	return nil, &ErrAuth{Message: "session expired and re-authentication failed"}
}

// backoff returns the wait before the given (zero-based) retry attempt.
func backoff(attempt int) time.Duration {
	return retryBaseWait * time.Duration(attempt+1)
}

// waitWithContext sleeps for d unless the context is cancelled first.
func waitWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// isTransientError returns true for network errors that may resolve on retry
// (connection refused, EOF, timeout) — typically caused by PiHole reloading
// its config after a write operation. Context cancellation is NOT transient.
func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe")
}

func (c *Client) executeRequest(ctx context.Context, method, path string, body []byte, sid string) (*http.Response, error) {
	url := fmt.Sprintf("%s/api%s", c.baseURL, path)

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("X-FTL-SID", sid)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpClient.Do(req)
}

func parseError(resp *http.Response) error {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &APIError{StatusCode: resp.StatusCode, Key: "unknown", Message: "failed to read error response"}
	}

	var apiErr apiErrorResponse
	if err := json.Unmarshal(bodyBytes, &apiErr); err == nil && apiErr.Error != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Key:        apiErr.Error.Key,
			Message:    apiErr.Error.Message,
			Hint:       apiErr.Error.Hint,
		}
	}

	return &APIError{StatusCode: resp.StatusCode, Key: "unknown", Message: string(bodyBytes)}
}
