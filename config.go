package pihole

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GetConfig reads a config value by its dot-notation path (e.g. "webserver.api.app_sudo").
// Returns the raw JSON-encoded value. Returns ErrNotFound if the path doesn't exist.
func (c *Client) GetConfig(ctx context.Context, key string) (json.RawMessage, error) {
	apiPath := configKeyToAPIPath(key)
	resp, err := c.doRequest(ctx, http.MethodGet, apiPath, nil)
	if err != nil {
		return nil, fmt.Errorf("getting config %s: %w", key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, &ErrNotFound{Resource: "config setting", ID: key}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	dec := json.NewDecoder(resp.Body)
	// Preserve numeric literals (don't coerce integers to float64, which loses
	// precision above 2^53 and would round-trip e.g. 1000000000000 as 1e+12).
	dec.UseNumber()
	var result map[string]interface{}
	if err := dec.Decode(&result); err != nil {
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
// Returns ErrNotFound if the path doesn't exist.
func (c *Client) SetConfig(ctx context.Context, key string, value json.RawMessage) error {
	// Build the nested config object for PATCH. The API validates the path
	// itself and returns 404 for an unknown key, so no pre-flight GET is needed.
	body := buildNestedConfig(key, value)
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling config patch: %w", err)
	}

	resp, err := c.doRequest(ctx, http.MethodPatch, "/config", bodyBytes)
	if err != nil {
		return fmt.Errorf("setting config %s: %w", key, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return &ErrNotFound{Resource: "config setting", ID: key}
	}
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
