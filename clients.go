package pihole

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type PiholeClient struct {
	ID           int    `json:"id"`
	Client       string `json:"client"`
	Comment      string `json:"comment"`
	Groups       []int  `json:"groups"`
	Enabled      bool   `json:"enabled"`
	Name         string `json:"name"`
	DateAdded    int64  `json:"date_added"`
	DateModified int64  `json:"date_modified"`
}

type ClientCreateRequest struct {
	Client  string `json:"client"`
	Comment string `json:"comment,omitempty"`
	Groups  []int  `json:"groups"`
}

type ClientUpdateRequest struct {
	Comment string `json:"comment,omitempty"`
	Groups  []int  `json:"groups"`
}

type clientsResponse struct {
	Clients []PiholeClient `json:"clients"`
}

func (c *Client) ListClients(ctx context.Context) ([]PiholeClient, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/clients", nil)
	if err != nil {
		return nil, fmt.Errorf("listing clients: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}
	var result clientsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding clients: %w", err)
	}
	return result.Clients, nil
}

func (c *Client) GetClient(ctx context.Context, clientID string) (*PiholeClient, error) {
	path := fmt.Sprintf("/clients/%s", url.PathEscape(clientID))
	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("getting client: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, &ErrNotFound{Resource: "client", ID: clientID}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}
	var result clientsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding client: %w", err)
	}
	if len(result.Clients) == 0 {
		return nil, &ErrNotFound{Resource: "client", ID: clientID}
	}
	return &result.Clients[0], nil
}

func (c *Client) CreateClient(ctx context.Context, req ClientCreateRequest) (*PiholeClient, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling client: %w", err)
	}
	resp, err := c.doRequest(ctx, http.MethodPost, "/clients", body)
	if err != nil {
		return nil, fmt.Errorf("creating client: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return nil, parseError(resp)
	}
	var result clientsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding client response: %w", err)
	}
	if len(result.Clients) == 0 {
		return nil, fmt.Errorf("no client returned")
	}
	return &result.Clients[0], nil
}

func (c *Client) UpdateClient(ctx context.Context, clientID string, req ClientUpdateRequest) (*PiholeClient, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling client update: %w", err)
	}
	path := fmt.Sprintf("/clients/%s", url.PathEscape(clientID))
	resp, err := c.doRequest(ctx, http.MethodPut, path, body)
	if err != nil {
		return nil, fmt.Errorf("updating client: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, parseError(resp)
	}
	var result clientsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding client response: %w", err)
	}
	if len(result.Clients) == 0 {
		return nil, fmt.Errorf("no client returned")
	}
	return &result.Clients[0], nil
}

func (c *Client) DeleteClient(ctx context.Context, clientID string) error {
	path := fmt.Sprintf("/clients/%s", url.PathEscape(clientID))
	resp, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return fmt.Errorf("deleting client: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return parseError(resp)
	}
	return nil
}
