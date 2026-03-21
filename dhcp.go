package pihole

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type DHCPStaticLease struct {
	MAC      string
	IP       string
	Hostname string
}

type dhcpHostsResponse struct {
	Config struct {
		DHCP struct {
			Hosts []string `json:"hosts"`
		} `json:"dhcp"`
	} `json:"config"`
}

// parseDHCPEntry parses "MAC,IP" or "MAC,IP,hostname" into a DHCPStaticLease.
func parseDHCPEntry(entry string) (DHCPStaticLease, bool) {
	parts := strings.SplitN(entry, ",", 3)
	if len(parts) < 2 {
		return DHCPStaticLease{}, false
	}
	lease := DHCPStaticLease{
		MAC: parts[0],
		IP:  parts[1],
	}
	if len(parts) == 3 {
		lease.Hostname = parts[2]
	}
	return lease, true
}

// formatDHCPEntry formats a DHCPStaticLease as "MAC,IP" or "MAC,IP,hostname".
func formatDHCPEntry(lease DHCPStaticLease) string {
	if lease.Hostname != "" {
		return fmt.Sprintf("%s,%s,%s", lease.MAC, lease.IP, lease.Hostname)
	}
	return fmt.Sprintf("%s,%s", lease.MAC, lease.IP)
}

func (c *Client) ListDHCPStaticLeases() ([]DHCPStaticLease, error) {
	resp, err := c.doRequest(http.MethodGet, "/config/dhcp/hosts", nil)
	if err != nil {
		return nil, fmt.Errorf("listing DHCP static leases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var result dhcpHostsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding DHCP static leases: %w", err)
	}

	leases := make([]DHCPStaticLease, 0, len(result.Config.DHCP.Hosts))
	for _, entry := range result.Config.DHCP.Hosts {
		if lease, ok := parseDHCPEntry(entry); ok {
			leases = append(leases, lease)
		}
	}

	return leases, nil
}

func (c *Client) GetDHCPStaticLease(mac string) (*DHCPStaticLease, error) {
	leases, err := c.ListDHCPStaticLeases()
	if err != nil {
		return nil, err
	}
	mac = strings.ToLower(mac)
	for _, l := range leases {
		if strings.ToLower(l.MAC) == mac {
			return &l, nil
		}
	}
	return nil, &ErrNotFound{Resource: "DHCP static lease", ID: mac}
}

func (c *Client) CreateDHCPStaticLease(lease DHCPStaticLease) error {
	entry := formatDHCPEntry(lease)
	path := fmt.Sprintf("/config/dhcp/hosts/%s", url.PathEscape(entry))
	resp, err := c.doRequest(http.MethodPut, path, nil)
	if err != nil {
		return fmt.Errorf("creating DHCP static lease: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated {
		return nil
	}
	apiErr := parseError(resp)
	if e, ok := apiErr.(*APIError); ok && e.StatusCode == http.StatusBadRequest && e.Key == "bad_request" && strings.Contains(e.Message, "already present") {
		return nil
	}
	return apiErr
}

func (c *Client) DeleteDHCPStaticLease(lease DHCPStaticLease) error {
	entry := formatDHCPEntry(lease)
	path := fmt.Sprintf("/config/dhcp/hosts/%s", url.PathEscape(entry))
	resp, err := c.doRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("deleting DHCP static lease: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return parseError(resp)
	}
	return nil
}
