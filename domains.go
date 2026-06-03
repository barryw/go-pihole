package pihole

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type DomainEntry struct {
	ID           int    `json:"id"`
	Domain       string `json:"domain"`
	Unicode      string `json:"unicode"`
	Type         string `json:"type"`
	Kind         string `json:"kind"`
	Comment      string `json:"comment"`
	Groups       []int  `json:"groups"`
	Enabled      bool   `json:"enabled"`
	DateAdded    int64  `json:"date_added"`
	DateModified int64  `json:"date_modified"`
}

type DomainCreateRequest struct {
	Domain  string `json:"domain"`
	Type    string `json:"-"`
	Kind    string `json:"-"`
	Comment string `json:"comment,omitempty"`
	Groups  []int  `json:"groups"`
	Enabled bool   `json:"enabled"`
}

type DomainUpdateRequest struct {
	// Type and Kind are encoded in the request URL, not the body.
	Type    string `json:"-"`
	Kind    string `json:"-"`
	Comment string `json:"comment,omitempty"`
	Groups  []int  `json:"groups"`
	Enabled bool   `json:"enabled"`
}

type domainsResponse struct {
	Domains []DomainEntry `json:"domains"`
}

func (c *Client) ListDomains(ctx context.Context) ([]DomainEntry, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/domains", nil)
	if err != nil {
		return nil, fmt.Errorf("listing domains: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}
	var result domainsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding domains: %w", err)
	}
	return result.Domains, nil
}

func (c *Client) ListDomainsByTypeAndKind(ctx context.Context, domainType, kind string) ([]DomainEntry, error) {
	path := fmt.Sprintf("/domains/%s/%s", url.PathEscape(domainType), url.PathEscape(kind))
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("listing domains: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}
	var result domainsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding domains: %w", err)
	}
	return result.Domains, nil
}

func (c *Client) GetDomain(ctx context.Context, domainType, kind, domain string) (*DomainEntry, error) {
	path := fmt.Sprintf("/domains/%s/%s/%s", url.PathEscape(domainType), url.PathEscape(kind), url.PathEscape(domain))
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting domain: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, &ErrNotFound{Resource: "domain", ID: fmt.Sprintf("%s/%s/%s", domainType, kind, domain)}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}
	var result domainsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding domain: %w", err)
	}
	if len(result.Domains) == 0 {
		return nil, &ErrNotFound{Resource: "domain", ID: fmt.Sprintf("%s/%s/%s", domainType, kind, domain)}
	}
	return &result.Domains[0], nil
}

func (c *Client) CreateDomain(ctx context.Context, req DomainCreateRequest) (*DomainEntry, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling domain: %w", err)
	}
	path := fmt.Sprintf("/domains/%s/%s", url.PathEscape(req.Type), url.PathEscape(req.Kind))
	resp, err := c.doRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, fmt.Errorf("creating domain: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return nil, parseError(resp)
	}
	var result domainsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding domain response: %w", err)
	}
	if len(result.Domains) == 0 {
		return nil, fmt.Errorf("no domain returned")
	}
	return &result.Domains[0], nil
}

func (c *Client) UpdateDomain(ctx context.Context, domainType, kind, domain string, req DomainUpdateRequest) (*DomainEntry, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling domain update: %w", err)
	}
	path := fmt.Sprintf("/domains/%s/%s/%s", url.PathEscape(domainType), url.PathEscape(kind), url.PathEscape(domain))
	resp, err := c.doRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return nil, fmt.Errorf("updating domain: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}
	var result domainsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding domain response: %w", err)
	}
	if len(result.Domains) == 0 {
		return nil, fmt.Errorf("no domain returned")
	}
	return &result.Domains[0], nil
}

func (c *Client) DeleteDomain(ctx context.Context, domainType, kind, domain string) error {
	path := fmt.Sprintf("/domains/%s/%s/%s", url.PathEscape(domainType), url.PathEscape(kind), url.PathEscape(domain))
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("deleting domain: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return parseError(resp)
	}
	return nil
}
