package pihole

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type authRequest struct {
	Password string `json:"password"`
}

type authResponse struct {
	Session struct {
		Valid    bool   `json:"valid"`
		SID      string `json:"sid"`
		CSRF     string `json:"csrf"`
		Validity int    `json:"validity"`
		Message  string `json:"message"`
	} `json:"session"`
}

func (c *Client) authenticate(ctx context.Context) error {
	body, err := json.Marshal(authRequest{Password: c.password})
	if err != nil {
		return fmt.Errorf("marshaling auth request: %w", err)
	}

	url := fmt.Sprintf("%s/api/auth", c.baseURL)

	var resp *http.Response
	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, rerr := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
		if rerr != nil {
			return fmt.Errorf("creating auth request: %w", rerr)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err = c.httpClient.Do(req)
		if err != nil {
			// PiHole-FTL restarts when its config changes (e.g. a DNS upstreams
			// update), briefly refusing connections. Retry transient errors so
			// authentication survives the reload rather than failing outright.
			if isTransientError(err) && attempt < maxRetries {
				if werr := waitWithContext(ctx, backoff(attempt)); werr != nil {
					return werr
				}
				continue
			}
			return fmt.Errorf("auth request failed: %w", err)
		}
		break
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return &ErrAuth{Message: "invalid password or app-password"}
	}
	if resp.StatusCode != http.StatusOK {
		return parseError(resp)
	}

	var authResp authResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("decoding auth response: %w", err)
	}

	// PiHole v6 returns session.valid==true even for a wrong password, with a
	// null sid and an explanatory message — so an empty sid is the real signal
	// of failure, not the valid flag alone.
	if !authResp.Session.Valid || authResp.Session.SID == "" {
		msg := authResp.Session.Message
		if msg == "" {
			msg = "authentication did not return a session id"
		}
		return &ErrAuth{Message: msg}
	}

	c.setSession(authResp.Session.SID)
	return nil
}
