package pihole

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Group struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Comment      string `json:"comment"`
	Enabled      bool   `json:"enabled"`
	DateAdded    int64  `json:"date_added"`
	DateModified int64  `json:"date_modified"`
}

type GroupCreateRequest struct {
	Name    string `json:"name"`
	Comment string `json:"comment,omitempty"`
	Enabled bool   `json:"enabled"`
}

type GroupUpdateRequest struct {
	Name    string `json:"name,omitempty"`
	Comment string `json:"comment,omitempty"`
	Enabled bool   `json:"enabled"`
}

type groupsResponse struct {
	Groups []Group `json:"groups"`
}

func (c *Client) ListGroups() ([]Group, error) {
	resp, err := c.doRequest(http.MethodGet, "/groups", nil)
	if err != nil {
		return nil, fmt.Errorf("listing groups: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}
	var result groupsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding groups: %w", err)
	}
	return result.Groups, nil
}

func (c *Client) GetGroup(name string) (*Group, error) {
	path := fmt.Sprintf("/groups/%s", url.PathEscape(name))
	resp, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting group: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, &ErrNotFound{Resource: "group", ID: name}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}
	var result groupsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding group: %w", err)
	}
	if len(result.Groups) == 0 {
		return nil, &ErrNotFound{Resource: "group", ID: name}
	}
	return &result.Groups[0], nil
}

func (c *Client) CreateGroup(req GroupCreateRequest) (*Group, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling group: %w", err)
	}
	resp, err := c.doRequest(http.MethodPost, "/groups", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating group: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return nil, parseError(resp)
	}
	var result groupsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding group response: %w", err)
	}
	if len(result.Groups) == 0 {
		return nil, fmt.Errorf("no group returned")
	}
	return &result.Groups[0], nil
}

func (c *Client) UpdateGroup(name string, req GroupUpdateRequest) (*Group, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling group update: %w", err)
	}
	path := fmt.Sprintf("/groups/%s", url.PathEscape(name))
	resp, err := c.doRequest(http.MethodPut, path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("updating group: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}
	var result groupsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding group response: %w", err)
	}
	if len(result.Groups) == 0 {
		return nil, fmt.Errorf("no group returned")
	}
	return &result.Groups[0], nil
}

func (c *Client) DeleteGroup(name string) error {
	path := fmt.Sprintf("/groups/%s", url.PathEscape(name))
	resp, err := c.doRequest(http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("deleting group: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return parseError(resp)
	}
	return nil
}
