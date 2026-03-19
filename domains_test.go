package pihole

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestListDomains(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/domains" || r.Method != http.MethodGet {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"domains": []map[string]interface{}{
				{"domain": "ads.example.com", "type": "deny", "kind": "exact", "groups": []int{0}, "enabled": true, "id": 1},
				{"domain": ".*\\.tracking\\.com$", "type": "deny", "kind": "regex", "groups": []int{0}, "enabled": true, "id": 2},
			},
		})
	})
	defer server.Close()
	domains, err := client.ListDomains()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("expected 2, got %d", len(domains))
	}
	if domains[0].Domain != "ads.example.com" || domains[0].Type != "deny" || domains[0].Kind != "exact" {
		t.Errorf("unexpected: %+v", domains[0])
	}
}

func TestListDomainsByTypeAndKind(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/domains/deny/exact" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"domains": []map[string]interface{}{
				{"domain": "ads.example.com", "type": "deny", "kind": "exact", "enabled": true, "id": 1, "groups": []int{0}},
			},
		})
	})
	defer server.Close()
	domains, err := client.ListDomainsByTypeAndKind("deny", "exact")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("expected 1, got %d", len(domains))
	}
}

func TestGetDomain(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/domains/deny/exact/ads.example.com" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"domains": []map[string]interface{}{
				{"domain": "ads.example.com", "type": "deny", "kind": "exact", "enabled": true, "id": 1, "groups": []int{0}},
			},
		})
	})
	defer server.Close()
	domain, err := client.GetDomain("deny", "exact", "ads.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if domain.Domain != "ads.example.com" {
		t.Errorf("unexpected: %s", domain.Domain)
	}
}

func TestCreateDomain(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/domains/deny/exact" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)
		if req["domain"] != "ads.example.com" {
			t.Errorf("unexpected: %v", req["domain"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"domains": []map[string]interface{}{
				{"domain": "ads.example.com", "type": "deny", "kind": "exact", "enabled": true, "id": 3, "groups": []int{0}},
			},
		})
	})
	defer server.Close()
	domain, err := client.CreateDomain(DomainCreateRequest{Domain: "ads.example.com", Type: "deny", Kind: "exact", Groups: []int{0}, Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if domain.ID != 3 {
		t.Errorf("expected 3, got %d", domain.ID)
	}
}

func TestUpdateDomain(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/domains/deny/exact/ads.example.com" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"domains": []map[string]interface{}{
				{"domain": "ads.example.com", "type": "deny", "kind": "exact", "comment": "updated", "enabled": true, "id": 1, "groups": []int{0, 1}},
			},
		})
	})
	defer server.Close()
	domain, err := client.UpdateDomain("deny", "exact", "ads.example.com", DomainUpdateRequest{Comment: "updated", Groups: []int{0, 1}, Enabled: true, Type: "deny", Kind: "exact"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if domain.Comment != "updated" {
		t.Errorf("expected updated, got %s", domain.Comment)
	}
}

func TestDeleteDomain(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/domains/deny/exact/ads.example.com" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()
	err := client.DeleteDomain("deny", "exact", "ads.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
