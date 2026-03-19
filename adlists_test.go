package pihole

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestListAdlists(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/lists" || r.Method != http.MethodGet {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"lists": []map[string]interface{}{
				{"address": "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts", "type": "block", "comment": "StevenBlack", "groups": []int{0}, "enabled": true, "id": 1},
			},
		})
	})
	defer server.Close()
	lists, err := client.ListAdlists()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lists) != 1 {
		t.Fatalf("expected 1, got %d", len(lists))
	}
	if lists[0].Type != "block" {
		t.Errorf("expected block, got %s", lists[0].Type)
	}
}

func TestGetAdlist(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"lists": []map[string]interface{}{
				{"address": "https://example.com/list.txt", "type": "block", "groups": []int{0}, "enabled": true, "id": 1},
			},
		})
	})
	defer server.Close()
	list, err := client.GetAdlist("https://example.com/list.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list.Address != "https://example.com/list.txt" {
		t.Errorf("unexpected: %s", list.Address)
	}
}

func TestCreateAdlist(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Query().Get("type") != "block" {
			t.Errorf("expected type=block")
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)
		if req["address"] != "https://example.com/list.txt" {
			t.Errorf("unexpected: %v", req["address"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"lists": []map[string]interface{}{
				{"address": "https://example.com/list.txt", "type": "block", "enabled": true, "id": 2, "groups": []int{0}},
			},
		})
	})
	defer server.Close()
	list, err := client.CreateAdlist(AdlistCreateRequest{Address: "https://example.com/list.txt", Type: "block", Groups: []int{0}, Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list.ID != 2 {
		t.Errorf("expected 2, got %d", list.ID)
	}
}

func TestUpdateAdlist(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"lists": []map[string]interface{}{
				{"address": "https://example.com/list.txt", "type": "block", "comment": "updated", "enabled": true, "id": 1, "groups": []int{0, 1}},
			},
		})
	})
	defer server.Close()
	list, err := client.UpdateAdlist("https://example.com/list.txt", "block", AdlistUpdateRequest{Comment: "updated", Groups: []int{0, 1}, Enabled: true, Type: "block"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list.Comment != "updated" {
		t.Errorf("expected updated, got %s", list.Comment)
	}
}

func TestDeleteAdlist(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()
	err := client.DeleteAdlist("https://example.com/list.txt", "block")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
