package pihole

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"
)

func TestGetDNSUpstreams(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/config/dns/upstreams" || r.Method != http.MethodGet {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"config":{"dns":{"upstreams":["1.1.1.1","8.8.8.8#5353"]}},"took":0.001}`))
	})
	defer server.Close()

	ups, err := client.GetDNSUpstreams(context.Background())
	if err != nil {
		t.Fatalf("GetDNSUpstreams: %v", err)
	}
	want := []string{"1.1.1.1", "8.8.8.8#5353"}
	if !reflect.DeepEqual(ups, want) {
		t.Errorf("expected %v, got %v", want, ups)
	}
}

func TestSetDNSUpstreams(t *testing.T) {
	var gotBody map[string]interface{}
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/config" || r.Method != http.MethodPatch {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"took":0.001}`))
	})
	defer server.Close()

	err := client.SetDNSUpstreams(context.Background(), []string{"1.1.1.1", "1.0.0.1"})
	if err != nil {
		t.Fatalf("SetDNSUpstreams: %v", err)
	}
	// Body must be {"config":{"dns":{"upstreams":["1.1.1.1","1.0.0.1"]}}}
	cfg := gotBody["config"].(map[string]interface{})
	dns := cfg["dns"].(map[string]interface{})
	ups := dns["upstreams"].([]interface{})
	if len(ups) != 2 || ups[0] != "1.1.1.1" || ups[1] != "1.0.0.1" {
		t.Errorf("unexpected upstreams body: %v", ups)
	}
}
