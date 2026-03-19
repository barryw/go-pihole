package pihole

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestGetConfig(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/config/webserver/api/app_sudo" || r.Method != http.MethodGet {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"config": map[string]interface{}{
				"webserver": map[string]interface{}{
					"api": map[string]interface{}{
						"app_sudo": true,
					},
				},
			},
		})
	})
	defer server.Close()

	val, err := client.GetConfig("webserver.api.app_sudo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(val) != "true" {
		t.Errorf("expected true, got %s", string(val))
	}
}

func TestGetConfig_Int(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"config": map[string]interface{}{
				"dns": map[string]interface{}{
					"blockTTL": 2,
				},
			},
		})
	})
	defer server.Close()

	val, err := client.GetConfig("dns.blockTTL")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// JSON numbers come back as float64 from encoding/json
	if string(val) != "2" {
		t.Errorf("expected 2, got %s", string(val))
	}
}

func TestGetConfig_String(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"config": map[string]interface{}{
				"dns": map[string]interface{}{
					"blocking": map[string]interface{}{
						"mode": "NULL",
					},
				},
			},
		})
	})
	defer server.Close()

	val, err := client.GetConfig("dns.blocking.mode")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(val) != `"NULL"` {
		t.Errorf("expected \"NULL\", got %s", string(val))
	}
}

func TestGetConfig_NotFound(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"config": map[string]interface{}{},
		})
	})
	defer server.Close()

	_, err := client.GetConfig("fake.nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
	if _, ok := err.(*ErrNotFound); !ok {
		t.Errorf("expected ErrNotFound, got %T: %v", err, err)
	}
}

func TestSetConfig(t *testing.T) {
	getCalled := false
	patchCalled := false

	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/config/dns/blockTTL" {
			getCalled = true
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"config": map[string]interface{}{
					"dns": map[string]interface{}{
						"blockTTL": 2,
					},
				},
			})
			return
		}
		if r.Method == http.MethodPatch && r.URL.Path == "/api/config" {
			patchCalled = true
			body, _ := io.ReadAll(r.Body)
			var req map[string]interface{}
			json.Unmarshal(body, &req)

			// Verify nested structure
			config, ok := req["config"].(map[string]interface{})
			if !ok {
				t.Error("expected config wrapper")
			}
			dns, ok := config["dns"].(map[string]interface{})
			if !ok {
				t.Error("expected dns key")
			}
			// JSON numbers decode as float64
			if val, ok := dns["blockTTL"].(float64); !ok || val != 5 {
				t.Errorf("expected blockTTL=5, got %v", dns["blockTTL"])
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"took": 0.001})
			return
		}
		t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := client.SetConfig("dns.blockTTL", json.RawMessage(`5`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !getCalled {
		t.Error("expected GET to validate path")
	}
	if !patchCalled {
		t.Error("expected PATCH to set value")
	}
}

func TestSetConfig_NotFound(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"config": map[string]interface{}{},
		})
	})
	defer server.Close()

	err := client.SetConfig("fake.path", json.RawMessage(`true`))
	if err == nil {
		t.Fatal("expected error for nonexistent path")
	}
	if _, ok := err.(*ErrNotFound); !ok {
		t.Errorf("expected ErrNotFound, got %T", err)
	}
}

func TestConfigKeyToAPIPath(t *testing.T) {
	tests := []struct {
		key      string
		expected string
	}{
		{"webserver.api.app_sudo", "/config/webserver/api/app_sudo"},
		{"dns.blockTTL", "/config/dns/blockTTL"},
		{"dhcp.active", "/config/dhcp/active"},
	}
	for _, tt := range tests {
		got := configKeyToAPIPath(tt.key)
		if got != tt.expected {
			t.Errorf("configKeyToAPIPath(%q) = %q, want %q", tt.key, got, tt.expected)
		}
	}
}

func TestBuildNestedConfig(t *testing.T) {
	result := buildNestedConfig("webserver.api.app_sudo", json.RawMessage(`true`))
	b, _ := json.Marshal(result)
	expected := `{"config":{"webserver":{"api":{"app_sudo":true}}}}`
	if string(b) != expected {
		t.Errorf("expected %s, got %s", expected, string(b))
	}
}
