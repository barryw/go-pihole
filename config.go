package pihole

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GetConfig reads a config value by its dot-notation path (e.g. "webserver.api.app_sudo").
// Returns the raw JSON-encoded value. Returns ErrNotFound if the path doesn't exist.
func (c *Client) GetConfig(key string) (json.RawMessage, error) {
	apiPath := configKeyToAPIPath(key)
	resp, err := c.doRequest(http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, fmt.Errorf("getting config %s: %w", key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding config response: %w", err)
	}

	// Navigate the nested config response to extract the leaf value.
	// Response looks like: {"config":{"webserver":{"api":{"app_sudo":true}}}}
	value, ok := extractNestedValue(result, key)
	if !ok {
		return nil, &ErrNotFound{Resource: "config setting", ID: key}
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshaling config value: %w", err)
	}

	return raw, nil
}

// SetConfig writes a config value using PATCH /api/config.
// The value should be a JSON-encoded value (e.g. json.RawMessage from jsonencode).
// Returns ErrNotFound if the path doesn't exist (verified via GET before PATCH).
func (c *Client) SetConfig(key string, value json.RawMessage) error {
	// Verify the path exists first
	_, err := c.GetConfig(key)
	if err != nil {
		return err
	}

	// Build the nested config object for PATCH
	body := buildNestedConfig(key, value)
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling config patch: %w", err)
	}

	resp, err := c.doRequest(http.MethodPatch, "/config", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("setting config %s: %w", key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return parseError(resp)
	}

	return nil
}

// configKeyToAPIPath converts "webserver.api.app_sudo" to "/config/webserver/api/app_sudo".
func configKeyToAPIPath(key string) string {
	parts := strings.Split(key, ".")
	escaped := make([]string, len(parts))
	for i, p := range parts {
		escaped[i] = url.PathEscape(p)
	}
	return "/config/" + strings.Join(escaped, "/")
}

// extractNestedValue navigates into the API response to get the leaf value.
// For key "webserver.api.app_sudo" and response {"config":{"webserver":{"api":{"app_sudo":true}}}},
// returns (true, true).
func extractNestedValue(result map[string]interface{}, key string) (interface{}, bool) {
	// Start from the "config" wrapper
	current, ok := result["config"]
	if !ok {
		return nil, false
	}

	parts := strings.Split(key, ".")
	for i, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
		// If this is the last part, return the value
		if i == len(parts)-1 {
			return current, true
		}
	}

	return nil, false
}

// buildNestedConfig creates {"config":{"webserver":{"api":{"app_sudo": <value>}}}}
// from key "webserver.api.app_sudo" and a JSON value.
func buildNestedConfig(key string, value json.RawMessage) map[string]interface{} {
	parts := strings.Split(key, ".")

	// Build from inside out
	var current interface{} = value
	for i := len(parts) - 1; i >= 0; i-- {
		current = map[string]interface{}{parts[i]: current}
	}

	return map[string]interface{}{"config": current}
}
