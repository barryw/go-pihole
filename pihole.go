package pihole

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

type Client struct {
	baseURL    string
	password   string
	httpClient *http.Client
	sid        string
	mu         sync.Mutex
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

// doRequest executes an HTTP request with authentication.
// Auto-authenticates if no session, retries once on 401.
func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
	if c.sid == "" {
		if err := c.authenticate(); err != nil {
			return nil, err
		}
	}

	resp, err := c.executeRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()
		if err := c.authenticate(); err != nil {
			return nil, err
		}
		resp, err = c.executeRequest(method, path, body)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode == http.StatusUnauthorized {
			resp.Body.Close()
			return nil, &ErrAuth{Message: "session expired and re-authentication failed"}
		}
	}

	return resp, nil
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
