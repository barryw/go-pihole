package pihole

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthenticate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/auth" || r.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var body struct {
			Password string `json:"password"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if body.Password != "test-password" {
			t.Errorf("unexpected password: %s", body.Password)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"session": map[string]interface{}{
				"valid":    true,
				"sid":      "test-sid-abc123",
				"csrf":     "test-csrf-xyz",
				"validity": 1800,
				"message":  "app-password correct",
			},
			"took": 0.003,
		})
	}))
	defer server.Close()

	client, _ := NewClient(server.URL, "test-password")
	err := client.authenticate()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.sid != "test-sid-abc123" {
		t.Errorf("expected sid test-sid-abc123, got %s", client.sid)
	}
}

func TestAuthenticate_WrongPassword(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"key":     "unauthorized",
				"message": "Invalid password",
			},
		})
	}))
	defer server.Close()

	client, _ := NewClient(server.URL, "wrong-password")
	err := client.authenticate()
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

func TestAuthenticate_SessionInvalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"session": map[string]interface{}{
				"valid":   false,
				"message": "password incorrect",
			},
			"took": 0.001,
		})
	}))
	defer server.Close()

	client, _ := NewClient(server.URL, "bad-password")
	err := client.authenticate()
	if err == nil {
		t.Fatal("expected error for invalid session")
	}
}
