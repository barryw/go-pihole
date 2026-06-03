package pihole

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestGetDNSBlocking(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/dns/blocking" || r.Method != http.MethodGet {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"blocking":"enabled","timer":null,"took":0.003}`))
	})
	defer server.Close()

	status, err := client.GetDNSBlocking(context.Background())
	if err != nil {
		t.Fatalf("GetDNSBlocking: %v", err)
	}
	if !status.Enabled {
		t.Error("expected blocking enabled")
	}
	if status.Timer != nil {
		t.Errorf("expected nil timer, got %v", *status.Timer)
	}
}

func TestGetDNSBlocking_DisabledWithTimer(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"blocking":"disabled","timer":29.5,"took":0.003}`))
	})
	defer server.Close()

	status, err := client.GetDNSBlocking(context.Background())
	if err != nil {
		t.Fatalf("GetDNSBlocking: %v", err)
	}
	if status.Enabled {
		t.Error("expected blocking disabled")
	}
	if status.Timer == nil || *status.Timer != 29.5 {
		t.Errorf("expected timer 29.5, got %v", status.Timer)
	}
}

func TestSetDNSBlocking(t *testing.T) {
	var gotBody map[string]interface{}
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/dns/blocking" || r.Method != http.MethodPost {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"blocking":"disabled","timer":60,"took":0.003}`))
	})
	defer server.Close()

	timer := 60
	if err := client.SetDNSBlocking(context.Background(), false, &timer); err != nil {
		t.Fatalf("SetDNSBlocking: %v", err)
	}
	if gotBody["blocking"] != false {
		t.Errorf("expected blocking=false in body, got %v", gotBody["blocking"])
	}
	if gotBody["timer"].(float64) != 60 {
		t.Errorf("expected timer=60 in body, got %v", gotBody["timer"])
	}
}

func TestSetDNSBlocking_NoTimerSendsNull(t *testing.T) {
	var raw map[string]json.RawMessage
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &raw)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"blocking":"enabled","timer":null,"took":0.003}`))
	})
	defer server.Close()

	if err := client.SetDNSBlocking(context.Background(), true, nil); err != nil {
		t.Fatalf("SetDNSBlocking: %v", err)
	}
	if string(raw["timer"]) != "null" {
		t.Errorf("expected timer:null in body, got %s", raw["timer"])
	}
}
