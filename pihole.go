package pihole

import (
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

type Client struct {
	baseURL    string
	password   string
	httpClient *http.Client
	sid        string
	mu         sync.Mutex
	authOnce   sync.Once
	authErr    error
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
		httpClient: &http.Client{},
	}, nil
}

type apiErrorResponse struct {
	Error *struct {
		Key     string `json:"key"`
		Message string `json:"message"`
		Hint    string `json:"hint"`
	} `json:"error"`
}

// Authenticate establishes a session with the PiHole API.
// Call this once after creating the client to avoid rate limiting
// when making many concurrent requests. If not called explicitly,
// authentication happens automatically on the first request.
func (c *Client) Authenticate() error {
	return c.authenticate()
}

const (
	maxRetries    = 5
	retryBaseWait = 2 * time.Second
)

// doRequest executes an HTTP request with authentication.
// Auto-authenticates if no session (using sync.Once to prevent
// parallel auth storms), retries on transient errors and 401.
func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	if c.sid == "" {
		c.authOnce.Do(func() {
			c.authErr = c.authenticate()
		})
		if c.authErr != nil {
			return nil, c.authErr
		}
	}

	var resp *http.Response
	var err error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err = c.executeRequest(method, path, body)
		if err != nil {
			if isTransientError(err) && attempt < maxRetries {
				time.Sleep(retryBaseWait * time.Duration(attempt+1))
				continue
			}
			return nil, err
		}

		if resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			if err := c.authenticate(); err != nil {
				if isTransientError(err) && attempt < maxRetries {
					time.Sleep(retryBaseWait * time.Duration(attempt+1))
					continue
				}
				return nil, err
			}
			resp, err = c.executeRequest(method, path, body)
			if err != nil {
				if isTransientError(err) && attempt < maxRetries {
					time.Sleep(retryBaseWait * time.Duration(attempt+1))
					continue
				}
				return nil, err
			}
			if resp.StatusCode == http.StatusUnauthorized {
				resp.Body.Close()
				return nil, &ErrAuth{Message: "session expired and re-authentication failed"}
			}
		}

		// Success or non-transient error
		break
	}

	return resp, nil
}

// isTransientError returns true for network errors that may resolve on retry
// (connection refused, EOF, timeout) — typically caused by PiHole reloading
// its config after a write operation.
func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	// Connection refused, reset, etc.
	var netErr *net.OpError
	if errors.As(err, &netErr) {
		return true
	}
	// EOF during read
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	// String matching as fallback for wrapped errors
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "broken pipe")
}

func (c *Client) executeRequest(method, path string, body io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("%s/api%s", c.baseURL, path)
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	c.mu.Lock()
	sid := c.sid
	c.mu.Unlock()

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
