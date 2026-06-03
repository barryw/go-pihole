package pihole

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type dnsUpstreamsResponse struct {
	Config struct {
		DNS struct {
			Upstreams []string `json:"upstreams"`
		} `json:"dns"`
	} `json:"config"`
}

// GetDNSUpstreams returns the ordered list of upstream DNS servers PiHole
// forwards queries to (e.g. "1.1.1.1", "8.8.8.8#5353").
func (c *Client) GetDNSUpstreams(ctx context.Context) ([]string, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/config/dns/upstreams", nil)
	if err != nil {
		return nil, fmt.Errorf("getting DNS upstreams: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}
	var result dnsUpstreamsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding DNS upstreams: %w", err)
	}
	return result.Config.DNS.Upstreams, nil
}

// SetDNSUpstreams replaces the entire upstream DNS server list.
func (c *Client) SetDNSUpstreams(ctx context.Context, upstreams []string) error {
	if upstreams == nil {
		upstreams = []string{}
	}
	body := map[string]interface{}{
		"config": map[string]interface{}{
			"dns": map[string]interface{}{
				"upstreams": upstreams,
			},
		},
	}
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling DNS upstreams: %w", err)
	}
	resp, err := c.doRequest(ctx, http.MethodPatch, "/config", bodyBytes)
	if err != nil {
		return fmt.Errorf("setting DNS upstreams: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return parseError(resp)
	}
	return nil
}
