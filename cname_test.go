package pihole

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestListCNAMERecords(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/config/dns/cnameRecords" || r.Method != http.MethodGet {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"config": map[string]interface{}{
				"dns": map[string]interface{}{
					"cnameRecords": []string{"alias.lan,target.lan", "other.lan,target2.lan,3600"},
				},
			},
		})
	})
	defer server.Close()

	records, err := client.ListCNAMERecords()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Domain != "alias.lan" || records[0].Target != "target.lan" || records[0].TTL != 0 {
		t.Errorf("unexpected record[0]: %+v", records[0])
	}
	if records[1].TTL != 3600 {
		t.Errorf("expected TTL 3600, got %d", records[1].TTL)
	}
}

func TestGetCNAMERecord(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"config": map[string]interface{}{
				"dns": map[string]interface{}{
					"cnameRecords": []string{"alias.lan,target.lan"},
				},
			},
		})
	})
	defer server.Close()

	record, err := client.GetCNAMERecord("alias.lan")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.Target != "target.lan" {
		t.Errorf("expected target target.lan, got %s", record.Target)
	}
}

func TestGetCNAMERecord_NotFound(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"config": map[string]interface{}{
				"dns": map[string]interface{}{
					"cnameRecords": []string{},
				},
			},
		})
	})
	defer server.Close()

	_, err := client.GetCNAMERecord("missing.lan")
	if err == nil {
		t.Fatal("expected error")
	}
	if _, ok := err.(*ErrNotFound); !ok {
		t.Errorf("expected ErrNotFound, got %T", err)
	}
}

func TestCreateCNAMERecord(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
	})
	defer server.Close()

	err := client.CreateCNAMERecord("alias.lan", "target.lan", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateCNAMERecord_WithTTL(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	defer server.Close()

	err := client.CreateCNAMERecord("alias.lan", "target.lan", 3600)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteCNAMERecord(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	err := client.DeleteCNAMERecord("alias.lan", "target.lan", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
