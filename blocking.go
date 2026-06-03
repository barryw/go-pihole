package pihole

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// DNSBlockingStatus reports whether ad/DNS blocking is currently active and,
// when temporarily toggled, how many seconds remain on the revert timer.
type DNSBlockingStatus struct {
	Enabled bool
	// Timer is the number of seconds until blocking reverts to its previous
	// state, or nil if no timer is active.
	Timer *float64
}

type dnsBlockingResponse struct {
	Blocking string   `json:"blocking"`
	Timer    *float64 `json:"timer"`
}

type dnsBlockingRequest struct {
	Blocking bool     `json:"blocking"`
	Timer    *float64 `json:"timer"`
}

// GetDNSBlocking returns the current DNS blocking state.
func (c *Client) GetDNSBlocking(ctx context.Context) (*DNSBlockingStatus, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/dns/blocking", nil)
	if err != nil {
		return nil, fmt.Errorf("getting DNS blocking status: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}
	var result dnsBlockingResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding DNS blocking status: %w", err)
	}
	return &DNSBlockingStatus{
		Enabled: result.Blocking == "enabled",
		Timer:   result.Timer,
	}, nil
}

// SetDNSBlocking enables or disables DNS blocking. If timerSeconds is non-nil,
// blocking reverts to its previous state after that many seconds.
func (c *Client) SetDNSBlocking(ctx context.Context, enabled bool, timerSeconds *int) error {
	req := dnsBlockingRequest{Blocking: enabled}
	if timerSeconds != nil {
		t := float64(*timerSeconds)
		req.Timer = &t
	}
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshaling DNS blocking request: %w", err)
	}
	resp, err := c.doRequest(ctx, http.MethodPost, "/dns/blocking", body)
	if err != nil {
		return fmt.Errorf("setting DNS blocking: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return parseError(resp)
	}
	return nil
}
