package pihole

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

func newTestServer(handler http.HandlerFunc) (*httptest.Server, *Client) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" && r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"session": map[string]interface{}{
					"valid": true,
					"sid":   "test-sid",
					"csrf":  "test-csrf",
				},
			})
			return
		}
		handler(w, r)
	}))

	client, _ := NewClient(server.URL, "test-password")
	return server, client
}
