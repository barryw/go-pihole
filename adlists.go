package pihole

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Adlist struct {
	ID             int    `json:"id"`
	Address        string `json:"address"`
	Type           string `json:"type"`
	Comment        string `json:"comment"`
	Groups         []int  `json:"groups"`
	Enabled        bool   `json:"enabled"`
	DateAdded      int64  `json:"date_added"`
	DateModified   int64  `json:"date_modified"`
	DateUpdated    int64  `json:"date_updated"`
	Number         int    `json:"number"`
	InvalidDomains int    `json:"invalid_domains"`
	ABPEntries     int    `json:"abp_entries"`
	Status         int    `json:"status"`
}

type AdlistCreateRequest struct {
	Address string `json:"address"`
	Type    string `json:"-"`
	Comment string `json:"comment,omitempty"`
	Groups  []int  `json:"groups"`
	Enabled bool   `json:"enabled"`
}

type AdlistUpdateRequest struct {
	Comment string `json:"comment,omitempty"`
	Type    string `json:"type"`
	Groups  []int  `json:"groups"`
	Enabled bool   `json:"enabled"`
}

type adlistsResponse struct {
	Lists []Adlist `json:"lists"`
}

func (c *Client) ListAdlists() ([]Adlist, error) {
	resp, err := c.doRequest(http.MethodGet, "/lists", nil)
	if err != nil {
		return nil, fmt.Errorf("listing adlists: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}
	var result adlistsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding adlists: %w", err)
	}
	return result.Lists, nil
}

func (c *Client) GetAdlist(address string) (*Adlist, error) {
	path := fmt.Sprintf("/lists/%s", url.PathEscape(address))
	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting adlist: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, &ErrNotFound{Resource: "adlist", ID: address}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}
	var result adlistsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding adlist: %w", err)
	}
	if len(result.Lists) == 0 {
		return nil, &ErrNotFound{Resource: "adlist", ID: address}
	}
	return &result.Lists[0], nil
}

func (c *Client) CreateAdlist(req AdlistCreateRequest) (*Adlist, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling adlist: %w", err)
	}
	path := fmt.Sprintf("/lists?type=%s", url.QueryEscape(req.Type))
	resp, err := c.doRequest(http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating adlist: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return nil, parseError(resp)
	}
	var result adlistsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding adlist response: %w", err)
	}
	if len(result.Lists) == 0 {
		return nil, fmt.Errorf("no adlist returned")
	}
	return &result.Lists[0], nil
}

func (c *Client) UpdateAdlist(address, listType string, req AdlistUpdateRequest) (*Adlist, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling adlist update: %w", err)
	}
	path := fmt.Sprintf("/lists/%s?type=%s", url.PathEscape(address), url.QueryEscape(listType))
	resp, err := c.doRequest(http.MethodPut, path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("updating adlist: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}
	var result adlistsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding adlist response: %w", err)
	}
	if len(result.Lists) == 0 {
		return nil, fmt.Errorf("no adlist returned")
	}
	return &result.Lists[0], nil
}

func (c *Client) DeleteAdlist(address, listType string) error {
	path := fmt.Sprintf("/lists/%s?type=%s", url.PathEscape(address), url.QueryEscape(listType))
	resp, err := c.doRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("deleting adlist: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return parseError(resp)
	}
	return nil
}
