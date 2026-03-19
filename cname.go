package pihole

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type CNAMERecord struct {
	Domain string
	Target string
	TTL    int
}

type cnameResponse struct {
	Config struct {
		DNS struct {
			CNAMERecords []string `json:"cnameRecords"`
		} `json:"dns"`
	} `json:"config"`
}

func (c *Client) ListCNAMERecords() ([]CNAMERecord, error) {
	resp, err := c.doRequest(http.MethodGet, "/config/dns/cnameRecords", nil)
	if err != nil {
		return nil, fmt.Errorf("listing CNAME records: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}
	var result cnameResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding CNAME records: %w", err)
	}
	records := make([]CNAMERecord, 0, len(result.Config.DNS.CNAMERecords))
	for _, entry := range result.Config.DNS.CNAMERecords {
		record, err := parseCNAMEEntry(entry)
		if err != nil {
			continue
		}
		records = append(records, record)
	}
	return records, nil
}

func (c *Client) GetCNAMERecord(domain string) (*CNAMERecord, error) {
	records, err := c.ListCNAMERecords()
	if err != nil {
		return nil, err
	}
	for _, r := range records {
		if r.Domain == domain {
			return &r, nil
		}
	}
	return nil, &ErrNotFound{Resource: "CNAME record", ID: domain}
}

func (c *Client) CreateCNAMERecord(domain, target string, ttl int) error {
	entry := formatCNAMEEntry(domain, target, ttl)
	path := fmt.Sprintf("/config/dns/cnameRecords/%s?restart=false", url.PathEscape(entry))
	resp, err := c.doRequest(http.MethodPut, path, nil)
	if err != nil {
		return fmt.Errorf("creating CNAME record: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated {
		return nil
	}
	apiErr := parseError(resp)
	if e, ok := apiErr.(*APIError); ok && e.StatusCode == http.StatusBadRequest && strings.Contains(e.Message, "already present") {
		return nil
	}
	return apiErr
}

func (c *Client) DeleteCNAMERecord(domain, target string, ttl int) error {
	entry := formatCNAMEEntry(domain, target, ttl)
	path := fmt.Sprintf("/config/dns/cnameRecords/%s?restart=false", url.PathEscape(entry))
	resp, err := c.doRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("deleting CNAME record: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return parseError(resp)
	}
	return nil
}

func parseCNAMEEntry(entry string) (CNAMERecord, error) {
	parts := strings.Split(entry, ",")
	if len(parts) < 2 {
		return CNAMERecord{}, fmt.Errorf("invalid CNAME entry: %s", entry)
	}
	record := CNAMERecord{Domain: parts[0], Target: parts[1]}
	if len(parts) >= 3 {
		ttl, err := strconv.Atoi(parts[2])
		if err == nil {
			record.TTL = ttl
		}
	}
	return record, nil
}

func formatCNAMEEntry(domain, target string, ttl int) string {
	if ttl > 0 {
		return fmt.Sprintf("%s,%s,%d", domain, target, ttl)
	}
	return fmt.Sprintf("%s,%s", domain, target)
}
