package pihole

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestListGroups(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/groups" || r.Method != http.MethodGet {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"groups": []map[string]interface{}{
				{"name": "Default", "comment": nil, "enabled": true, "id": 0},
				{"name": "IoT", "comment": "IoT devices", "enabled": true, "id": 1},
			},
		})
	})
	defer server.Close()
	groups, err := client.ListGroups()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(groups) != 2 {
		t.Fatalf("expected 2, got %d", len(groups))
	}
	if groups[1].Name != "IoT" {
		t.Errorf("unexpected: %+v", groups[1])
	}
}

func TestGetGroup(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/groups/IoT" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"groups": []map[string]interface{}{
				{"name": "IoT", "comment": "IoT devices", "enabled": true, "id": 1},
			},
		})
	})
	defer server.Close()
	group, err := client.GetGroup("IoT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group.Name != "IoT" {
		t.Errorf("expected IoT, got %s", group.Name)
	}
}

func TestCreateGroup(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/groups" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)
		if req["name"] != "TestGroup" {
			t.Errorf("unexpected name: %v", req["name"])
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"groups": []map[string]interface{}{
				{"name": "TestGroup", "comment": "test", "enabled": true, "id": 2},
			},
		})
	})
	defer server.Close()
	group, err := client.CreateGroup(GroupCreateRequest{Name: "TestGroup", Comment: "test", Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group.Name != "TestGroup" {
		t.Errorf("expected TestGroup, got %s", group.Name)
	}
}

func TestUpdateGroup(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/groups/OldName" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"groups": []map[string]interface{}{
				{"name": "NewName", "comment": "updated", "enabled": true, "id": 1},
			},
		})
	})
	defer server.Close()
	group, err := client.UpdateGroup("OldName", GroupUpdateRequest{Name: "NewName", Comment: "updated", Enabled: true})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if group.Name != "NewName" {
		t.Errorf("expected NewName, got %s", group.Name)
	}
}

func TestDeleteGroup(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api/groups/TestGroup" {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()
	err := client.DeleteGroup("TestGroup")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
