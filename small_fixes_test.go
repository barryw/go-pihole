package pihole

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestUpdateDomainBodyOmitsTypeAndKind verifies that type and kind are encoded
// in the URL only and never leak into the PUT request body (where the API does
// not expect them).
func TestUpdateDomainBodyOmitsTypeAndKind(t *testing.T) {
	var gotBody map[string]json.RawMessage
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"domains":[{"domain":"bad.com","type":"deny","kind":"exact"}]}`))
	})
	defer server.Close()

	_, err := client.UpdateDomain(context.Background(), "deny", "exact", "bad.com", DomainUpdateRequest{
		Type: "deny", Kind: "exact", Comment: "x", Enabled: true,
	})
	if err != nil {
		t.Fatalf("UpdateDomain: %v", err)
	}
	if _, ok := gotBody["type"]; ok {
		t.Error("request body must not contain 'type'")
	}
	if _, ok := gotBody["kind"]; ok {
		t.Error("request body must not contain 'kind'")
	}
}

// TestUpdateDHCPLeaseReportsRestoredOldLease verifies that when creating the
// new lease fails, the error states that the old lease was restored.
func TestUpdateDHCPLeaseReportsRestoredOldLease(t *testing.T) {
	old := DHCPStaticLease{MAC: "aa:bb:cc:dd:ee:ff", IP: "10.0.0.1", Hostname: "old"}
	updated := DHCPStaticLease{MAC: "aa:bb:cc:dd:ee:ff", IP: "10.0.0.2", Hostname: "new"}

	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "10.0.0.2"):
			w.WriteHeader(http.StatusInternalServerError) // new lease fails
		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "10.0.0.1"):
			w.WriteHeader(http.StatusCreated) // restore succeeds
		default:
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := client.UpdateDHCPStaticLease(context.Background(), old, updated)
	if err == nil {
		t.Fatal("expected error when creating new lease fails")
	}
	if !strings.Contains(err.Error(), "old lease restored") {
		t.Errorf("expected error to report old lease restored, got: %v", err)
	}
}

// TestUpdateDHCPLeaseReportsFailedRestore verifies that when both the new-lease
// create AND the restore fail, the error surfaces the restore failure rather
// than silently swallowing it.
func TestUpdateDHCPLeaseReportsFailedRestore(t *testing.T) {
	old := DHCPStaticLease{MAC: "aa:bb:cc:dd:ee:ff", IP: "10.0.0.1", Hostname: "old"}
	updated := DHCPStaticLease{MAC: "aa:bb:cc:dd:ee:ff", IP: "10.0.0.2", Hostname: "new"}

	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodPut:
			w.WriteHeader(http.StatusInternalServerError) // both new and restore fail
		default:
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := client.UpdateDHCPStaticLease(context.Background(), old, updated)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "restoring old lease also failed") {
		t.Errorf("expected error to surface restore failure, got: %v", err)
	}
}
