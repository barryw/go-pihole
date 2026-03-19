package pihole

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type DNSRecord struct {
	IP     string
	Domain string
}

type dnsHostsResponse struct {
	Config struct {
		DNS struct {
			Hosts []string `json:"hosts"`
		} `json:"dns"`
	} `json:"config"`
}

func (c *Client) ListDNSRecords() ([]DNSRecord, error) {
	resp, err := c.doRequest(http.MethodGet, "/config/dns/hosts", nil)
	if err != nil {
		return nil, fmt.Errorf("listing DNS records: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}

	var result dnsHostsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding DNS records: %w", err)
	}

	records := make([]DNSRecord, 0, len(result.Config.DNS.Hosts))
	for _, entry := range result.Config.DNS.Hosts {
		parts := strings.SplitN(entry, " ", 2)
		if len(parts) != 2 {
			continue
		}
		records = append(records, DNSRecord{IP: parts[0], Domain: parts[1]})
	}

	return records, nil
}

func (c *Client) GetDNSRecord(domain string) (*DNSRecord, error) {
	records, err := c.ListDNSRecords()
	if err != nil {
		return nil, err
	}
	for _, r := range records {
		if r.Domain == domain {
			return &r, nil
		}
	}
	return nil, &ErrNotFound{Resource: "DNS record", ID: domain}
}

func (c *Client) CreateDNSRecord(ip, domain string) error {
	entry := fmt.Sprintf("%s %s", ip, domain)
	path := fmt.Sprintf("/config/dns/hosts/%s", url.PathEscape(entry))
	resp, err := c.doRequest(http.MethodPut, path, nil)
	if err != nil {
		return fmt.Errorf("creating DNS record: %w", err)
	}
	defer resp.Body.Close()
	// 201 = created, 400 with "already present" = idempotent success
	if resp.StatusCode == http.StatusCreated {
		return nil
	}
	apiErr := parseError(resp)
	if e, ok := apiErr.(*APIError); ok && e.StatusCode == http.StatusBadRequest && e.Key == "bad_request" && strings.Contains(e.Message, "already present") {
		return nil
	}
	return apiErr
}

func (c *Client) DeleteDNSRecord(ip, domain string) error {
	entry := fmt.Sprintf("%s %s", ip, domain)
	path := fmt.Sprintf("/config/dns/hosts/%s", url.PathEscape(entry))
	resp, err := c.doRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("deleting DNS record: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return parseError(resp)
	}
	return nil
}
