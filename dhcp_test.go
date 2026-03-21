package pihole

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestListDHCPStaticLeases(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/config/dhcp/hosts" || r.Method != http.MethodGet {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"config": map[string]interface{}{
				"dhcp": map[string]interface{}{
					"hosts": []string{
						"11:22:33:44:55:66,192.168.1.100",
						"aa:bb:cc:dd:ee:ff,192.168.1.200,myhost",
					},
				},
			},
		})
	})
	defer server.Close()

	leases, err := client.ListDHCPStaticLeases()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(leases) != 2 {
		t.Fatalf("expected 2 leases, got %d", len(leases))
	}
	if leases[0].MAC != "11:22:33:44:55:66" || leases[0].IP != "192.168.1.100" || leases[0].Hostname != "" {
		t.Errorf("unexpected lease[0]: %+v", leases[0])
	}
	if leases[1].MAC != "aa:bb:cc:dd:ee:ff" || leases[1].IP != "192.168.1.200" || leases[1].Hostname != "myhost" {
		t.Errorf("unexpected lease[1]: %+v", leases[1])
	}
}

func TestGetDHCPStaticLease(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"config": map[string]interface{}{
				"dhcp": map[string]interface{}{
					"hosts": []string{"11:22:33:44:55:66,192.168.1.100,mynas"},
				},
			},
		})
	})
	defer server.Close()

	lease, err := client.GetDHCPStaticLease("11:22:33:44:55:66")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lease.IP != "192.168.1.100" || lease.Hostname != "mynas" {
		t.Errorf("unexpected lease: %+v", lease)
	}
}

func TestGetDHCPStaticLease_CaseInsensitive(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"config": map[string]interface{}{
				"dhcp": map[string]interface{}{
					"hosts": []string{"AA:BB:CC:DD:EE:FF,192.168.1.100"},
				},
			},
		})
	})
	defer server.Close()

	lease, err := client.GetDHCPStaticLease("aa:bb:cc:dd:ee:ff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lease.IP != "192.168.1.100" {
		t.Errorf("unexpected lease: %+v", lease)
	}
}

func TestGetDHCPStaticLease_NotFound(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"config": map[string]interface{}{
				"dhcp": map[string]interface{}{
					"hosts": []string{},
				},
			},
		})
	})
	defer server.Close()

	_, err := client.GetDHCPStaticLease("00:00:00:00:00:00")
	if err == nil {
		t.Fatal("expected error for missing lease")
	}
	if _, ok := err.(*ErrNotFound); !ok {
		t.Errorf("expected ErrNotFound, got %T", err)
	}
}

func TestCreateDHCPStaticLease(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		expected := "/api/config/dhcp/hosts/11:22:33:44:55:66%2C192.168.1.100%2Cmynas"
		if r.URL.RawPath != expected {
			t.Errorf("unexpected path: %s\nexpected: %s", r.URL.RawPath, expected)
		}
		w.WriteHeader(http.StatusCreated)
	})
	defer server.Close()

	err := client.CreateDHCPStaticLease(DHCPStaticLease{
		MAC:      "11:22:33:44:55:66",
		IP:       "192.168.1.100",
		Hostname: "mynas",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteDHCPStaticLease(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	err := client.DeleteDHCPStaticLease(DHCPStaticLease{
		MAC: "11:22:33:44:55:66",
		IP:  "192.168.1.100",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseDHCPEntry(t *testing.T) {
	tests := []struct {
		entry    string
		wantOK   bool
		wantMAC  string
		wantIP   string
		wantHost string
	}{
		{"11:22:33:44:55:66,192.168.1.100", true, "11:22:33:44:55:66", "192.168.1.100", ""},
		{"11:22:33:44:55:66,192.168.1.100,myhost", true, "11:22:33:44:55:66", "192.168.1.100", "myhost"},
		{"invalid", false, "", "", ""},
	}
	for _, tt := range tests {
		lease, ok := parseDHCPEntry(tt.entry)
		if ok != tt.wantOK {
			t.Errorf("parseDHCPEntry(%q) ok=%v, want %v", tt.entry, ok, tt.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if lease.MAC != tt.wantMAC || lease.IP != tt.wantIP || lease.Hostname != tt.wantHost {
			t.Errorf("parseDHCPEntry(%q) = %+v, want MAC=%s IP=%s Host=%s", tt.entry, lease, tt.wantMAC, tt.wantIP, tt.wantHost)
		}
	}
}

func TestFormatDHCPEntry(t *testing.T) {
	tests := []struct {
		lease DHCPStaticLease
		want  string
	}{
		{DHCPStaticLease{MAC: "aa:bb:cc:dd:ee:ff", IP: "10.0.0.1"}, "aa:bb:cc:dd:ee:ff,10.0.0.1"},
		{DHCPStaticLease{MAC: "aa:bb:cc:dd:ee:ff", IP: "10.0.0.1", Hostname: "host"}, "aa:bb:cc:dd:ee:ff,10.0.0.1,host"},
	}
	for _, tt := range tests {
		got := formatDHCPEntry(tt.lease)
		if got != tt.want {
			t.Errorf("formatDHCPEntry(%+v) = %q, want %q", tt.lease, got, tt.want)
		}
	}
}
