package pihole

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestListDNSRecords(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/config/dns/hosts" || r.Method != http.MethodGet {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"config": map[string]interface{}{
				"dns": map[string]interface{}{
					"hosts": []string{"192.168.1.100 myhost.lan", "10.0.0.1 other.lan"},
				},
			},
		})
	})
	defer server.Close()

	records, err := client.ListDNSRecords()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].IP != "192.168.1.100" || records[0].Domain != "myhost.lan" {
		t.Errorf("unexpected record: %+v", records[0])
	}
}

func TestGetDNSRecord(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"config": map[string]interface{}{
				"dns": map[string]interface{}{
					"hosts": []string{"192.168.1.100 myhost.lan", "10.0.0.1 other.lan"},
				},
			},
		})
	})
	defer server.Close()

	record, err := client.GetDNSRecord("myhost.lan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.IP != "192.168.1.100" {
		t.Errorf("expected IP 192.168.1.100, got %s", record.IP)
	}
}

func TestGetDNSRecord_NotFound(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"config": map[string]interface{}{
				"dns": map[string]interface{}{
					"hosts": []string{},
				},
			},
		})
	})
	defer server.Close()

	_, err := client.GetDNSRecord("missing.lan")
	if err == nil {
		t.Fatal("expected error for missing record")
	}
	if _, ok := err.(*ErrNotFound); !ok {
		t.Errorf("expected ErrNotFound, got %T", err)
	}
}

func TestCreateDNSRecord(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
	})
	defer server.Close()

	err := client.CreateDNSRecord("192.168.1.100", "myhost.lan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteDNSRecord(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	err := client.DeleteDNSRecord("192.168.1.100", "myhost.lan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
