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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth request failed: %w", err)
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
