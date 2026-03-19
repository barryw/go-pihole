package pihole

import (
	"bytes"
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
		SID     string `json:"sid"`
		CSRF    string `json:"csrf"`
		Validity int   `json:"validity"`
		Message  string `json:"message"`
	} `json:"session"`
}

func (c *Client) authenticate() error {
	body, err := json.Marshal(authRequest{Password: c.password})
	if err != nil {
		return fmt.Errorf("marshaling auth request: %w", err)
	}

	url := fmt.Sprintf("%s/api/auth", c.baseURL)
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
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

	if !authResp.Session.Valid {
		return &ErrAuth{Message: authResp.Session.Message}
	}

	c.mu.Lock()
	c.sid = authResp.Session.SID
	c.mu.Unlock()

	return nil
}
