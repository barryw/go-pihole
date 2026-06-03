package pihole

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestRetryResendsBodyAfter401 verifies that when a write request receives a
// 401 (session expired), the retried request after re-authentication still
// carries the original request body. The original bug reused a single
// io.Reader across attempts, so the retry sent an empty body.
func TestRetryResendsBodyAfter401(t *testing.T) {
	var bodies [][]byte
	var mu sync.Mutex
	var protectedCalls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" && r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"session":{"valid":true,"sid":"sid","csrf":"c"}}`))
			return
		}
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, b)
		mu.Unlock()
		// First protected call: pretend the session expired.
		if atomic.AddInt32(&protectedCalls, 1) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"clients":[{"client":"10.0.0.1","comment":"x"}]}`))
	}))
	defer server.Close()

	client, _ := NewClient(server.URL, "pw")
	_, err := client.CreateClient(context.Background(), ClientCreateRequest{Client: "10.0.0.1", Comment: "x"})
	if err != nil {
		t.Fatalf("CreateClient: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(bodies) < 2 {
		t.Fatalf("expected at least 2 protected requests (initial + retry), got %d", len(bodies))
	}
	last := bodies[len(bodies)-1]
	if len(last) == 0 {
		t.Fatalf("retry sent an empty body; body was not resent")
	}
}

// TestContextCancellationAborts verifies that cancelling the context aborts an
// in-flight request rather than blocking until the server responds.
func TestContextCancellationAborts(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" {
			_, _ = w.Write([]byte(`{"session":{"valid":true,"sid":"sid","csrf":"c"}}`))
			return
		}
		<-release // block until the test releases
	}))
	defer server.Close()
	defer close(release)

	client, _ := NewClient(server.URL, "pw")
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	done := make(chan error, 1)
	go func() {
		_, err := client.ListGroups(ctx)
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected an error from cancelled context, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("request did not abort on context cancellation")
	}
}

// TestReauthSingleFlight verifies that when many concurrent requests all see a
// 401 at once, only a single re-authentication occurs instead of one per
// goroutine (which could storm/crash the PiHole API).
func TestReauthSingleFlight(t *testing.T) {
	var authCount int32
	var protectedCalls int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" && r.Method == http.MethodPost {
			n := atomic.AddInt32(&authCount, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"session":{"valid":true,"sid":"sid-%d","csrf":"c"}}`, n)))
			return
		}
		// Force exactly one wave of 401s: every request made before the
		// re-auth (authCount < 2) is rejected; afterwards they succeed.
		atomic.AddInt32(&protectedCalls, 1)
		if atomic.LoadInt32(&authCount) < 2 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"groups":[]}`))
	}))
	defer server.Close()

	client, _ := NewClient(server.URL, "pw")
	// Authenticate eagerly so all goroutines share sid-1 (authCount==1).
	if err := client.Authenticate(context.Background()); err != nil {
		t.Fatalf("eager auth: %v", err)
	}

	const n = 20
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, _ = client.ListGroups(context.Background())
		}()
	}
	close(start)
	wg.Wait()

	if got := atomic.LoadInt32(&authCount); got != 2 {
		t.Fatalf("expected exactly 2 authentications (initial + 1 re-auth), got %d", got)
	}
}

// TestAuthRecoversAfterTransientFailure verifies that a transient failure on
// the first authentication does not permanently poison the client (the old
// sync.Once cached the error forever).
func TestAuthRecoversAfterTransientFailure(t *testing.T) {
	var authAttempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/auth" && r.Method == http.MethodPost {
			if atomic.AddInt32(&authAttempts, 1) == 1 {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":{"key":"boom","message":"temporary"}}`))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"session":{"valid":true,"sid":"sid","csrf":"c"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"groups":[]}`))
	}))
	defer server.Close()

	client, _ := NewClient(server.URL, "pw")
	ctx := context.Background()

	if err := client.Authenticate(ctx); err == nil {
		t.Fatal("expected first authentication to fail")
	}
	// Second attempt must be allowed to succeed, not return the cached error.
	if _, err := client.ListGroups(ctx); err != nil {
		t.Fatalf("client did not recover after transient auth failure: %v", err)
	}
}

// TestConcurrentRequestsNoRace exercises many parallel reads to surface data
// races on the shared session state under `go test -race`.
func TestConcurrentRequestsNoRace(t *testing.T) {
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"groups":[]}`))
	})
	defer server.Close()

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = client.ListGroups(context.Background())
		}()
	}
	wg.Wait()
}

// TestAuthenticateRetriesTransient verifies that authentication survives a
// transient connection drop — PiHole-FTL restarts when config changes (e.g. a
// DNS upstreams update), briefly refusing connections, and auth must retry
// rather than fail outright.
func TestAuthenticateRetriesTransient(t *testing.T) {
	var authCalls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&authCalls, 1) == 1 {
			// Simulate FTL mid-restart: drop the connection before responding.
			if hj, ok := w.(http.Hijacker); ok {
				conn, _, _ := hj.Hijack()
				_ = conn.Close()
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"session":{"valid":true,"sid":"sid","csrf":"c"}}`))
	}))
	defer server.Close()

	client, _ := NewClient(server.URL, "pw")
	if err := client.Authenticate(context.Background()); err != nil {
		t.Fatalf("expected auth to survive a transient connection drop, got: %v", err)
	}
	if got := atomic.LoadInt32(&authCalls); got < 2 {
		t.Fatalf("expected auth to retry (>=2 attempts), got %d", got)
	}
}

// TestHTTPClientHasTimeout guards against the bare &http.Client{} that could
// hang forever on a stalled connection.
func TestHTTPClientHasTimeout(t *testing.T) {
	client, _ := NewClient("http://example.invalid", "pw")
	if client.httpClient.Timeout == 0 {
		t.Fatal("expected a non-zero default HTTP client timeout")
	}
}
