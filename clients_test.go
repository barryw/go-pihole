package pihole

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestListClients(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/clients" || r.Method != http.MethodGet {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"clients": []map[string]interface{}{
				{"client": "192.168.1.100", "comment": "Desktop", "groups": []int{0}, "enabled": true, "id": 1, "name": "desktop"},
			},
		})
	})
	defer server.Close()
	clients, err := client.ListClients()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(clients) != 1 {
		t.Fatalf("expected 1, got %d", len(clients))
	}
	if clients[0].Client != "192.168.1.100" {
		t.Errorf("unexpected: %s", clients[0].Client)
	}
}

func TestGetClient(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/clients/192.168.1.100" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"clients": []map[string]interface{}{
				{"client": "192.168.1.100", "comment": "Desktop", "groups": []int{0}, "enabled": true, "id": 1, "name": "desktop"},
			},
		})
	})
	defer server.Close()
	cl, err := client.GetClient("192.168.1.100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cl.Client != "192.168.1.100" {
		t.Errorf("unexpected: %s", cl.Client)
	}
}

func TestCreateClient(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/clients" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)
		if req["client"] != "192.168.1.200" {
			t.Errorf("unexpected: %v", req["client"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"clients": []map[string]interface{}{
				{"client": "192.168.1.200", "comment": "new device", "groups": []int{0}, "enabled": true, "id": 2},
			},
		})
	})
	defer server.Close()
	cl, err := client.CreateClient(ClientCreateRequest{Client: "192.168.1.200", Comment: "new device", Groups: []int{0}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cl.ID != 2 {
		t.Errorf("expected 2, got %d", cl.ID)
	}
}

func TestUpdateClient(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/clients/192.168.1.100" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"clients": []map[string]interface{}{
				{"client": "192.168.1.100", "comment": "updated", "groups": []int{0, 1}, "enabled": true, "id": 1},
			},
		})
	})
	defer server.Close()
	cl, err := client.UpdateClient("192.168.1.100", ClientUpdateRequest{Comment: "updated", Groups: []int{0, 1}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cl.Comment != "updated" {
		t.Errorf("expected updated, got %s", cl.Comment)
	}
}

func TestDeleteClient(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/clients/192.168.1.100" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()
	err := client.DeleteClient("192.168.1.100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
