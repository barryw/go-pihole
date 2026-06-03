package pihole

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
)

// TestIntegration exercises the client against a REAL PiHole, gated behind
// PIHOLE_INTEGRATION=1. It only touches scratch resources (unique prefix) and
// cleans them up; it reads — never mutates — the global singletons (blocking,
// upstreams) so it is safe to run against production. Writes are serial because
// the PiHole v6 API is fragile under concurrent writes; reads are run in
// parallel to exercise the concurrency/re-auth fixes.
func TestIntegration(t *testing.T) {
	if os.Getenv("PIHOLE_INTEGRATION") != "1" {
		t.Skip("set PIHOLE_INTEGRATION=1 to run live integration tests")
	}
	ctx := context.Background()
	client := integrationClient(t)
	const px = "zz-tfacc-" // scratch prefix

	t.Run("group CRUD", func(t *testing.T) {
		name := px + "group"
		_ = client.DeleteGroup(ctx, name) // pre-clean
		g, err := client.CreateGroup(ctx, GroupCreateRequest{Name: name, Comment: "created by integration test", Enabled: true})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		t.Cleanup(func() { _ = client.DeleteGroup(ctx, name) })
		if g.Name != name {
			t.Fatalf("expected name %q, got %q", name, g.Name)
		}
		u, err := client.UpdateGroup(ctx, name, GroupUpdateRequest{Name: name, Comment: "updated", Enabled: false})
		if err != nil {
			t.Fatalf("update: %v", err)
		}
		if u.Comment != "updated" || u.Enabled {
			t.Fatalf("update not applied: %+v", u)
		}
		got, err := client.GetGroup(ctx, name)
		if err != nil || got.Comment != "updated" {
			t.Fatalf("get after update: %+v err=%v", got, err)
		}
		if err := client.DeleteGroup(ctx, name); err != nil {
			t.Fatalf("delete: %v", err)
		}
		if _, err := client.GetGroup(ctx, name); err == nil {
			t.Fatal("expected group to be gone")
		}
	})

	t.Run("dns record CRUD", func(t *testing.T) {
		domain := px + "host.lan"
		ip := "192.0.2.123" // TEST-NET-1, never real
		_ = client.DeleteDNSRecord(ctx, ip, domain)
		if err := client.CreateDNSRecord(ctx, ip, domain); err != nil {
			t.Fatalf("create: %v", err)
		}
		t.Cleanup(func() { _ = client.DeleteDNSRecord(ctx, ip, domain) })
		rec, err := client.GetDNSRecord(ctx, domain)
		if err != nil || rec.IP != ip {
			t.Fatalf("get: %+v err=%v", rec, err)
		}
		if err := client.DeleteDNSRecord(ctx, ip, domain); err != nil {
			t.Fatalf("delete: %v", err)
		}
		if _, err := client.GetDNSRecord(ctx, domain); err == nil {
			t.Fatal("expected dns record to be gone")
		}
	})

	t.Run("cname CRUD", func(t *testing.T) {
		domain := px + "alias.lan"
		target := px + "host.lan"
		_ = client.DeleteCNAMERecord(ctx, domain, target, 0)
		if err := client.CreateCNAMERecord(ctx, domain, target, 0); err != nil {
			t.Fatalf("create: %v", err)
		}
		t.Cleanup(func() { _ = client.DeleteCNAMERecord(ctx, domain, target, 0) })
		if _, err := client.GetCNAMERecord(ctx, domain); err != nil {
			t.Fatalf("get: %v", err)
		}
		if err := client.DeleteCNAMERecord(ctx, domain, target, 0); err != nil {
			t.Fatalf("delete: %v", err)
		}
	})

	t.Run("domain list CRUD", func(t *testing.T) {
		domain := px + "blocked.example"
		_ = client.DeleteDomain(ctx, "deny", "exact", domain)
		d, err := client.CreateDomain(ctx, DomainCreateRequest{Domain: domain, Type: "deny", Kind: "exact", Comment: "integration", Enabled: true})
		if err != nil {
			t.Fatalf("create: %v", err)
		}
		t.Cleanup(func() { _ = client.DeleteDomain(ctx, "deny", "exact", domain) })
		if d.Domain != domain {
			t.Fatalf("expected %q, got %q", domain, d.Domain)
		}
		if err := client.DeleteDomain(ctx, "deny", "exact", domain); err != nil {
			t.Fatalf("delete: %v", err)
		}
	})

	t.Run("read-only singletons", func(t *testing.T) {
		b, err := client.GetDNSBlocking(ctx)
		if err != nil {
			t.Fatalf("GetDNSBlocking: %v", err)
		}
		t.Logf("blocking enabled=%v timer=%v", b.Enabled, b.Timer)
		ups, err := client.GetDNSUpstreams(ctx)
		if err != nil {
			t.Fatalf("GetDNSUpstreams: %v", err)
		}
		t.Logf("upstreams=%v", ups)
		if len(ups) == 0 {
			t.Fatal("expected at least one upstream")
		}
	})

	// Parallel READS exercise the data-race / re-auth fixes against the real
	// server. Reads only — no concurrent writes.
	t.Run("parallel reads", func(t *testing.T) {
		var wg sync.WaitGroup
		errs := make([]error, 24)
		for i := 0; i < 24; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				switch i % 4 {
				case 0:
					_, errs[i] = client.ListGroups(ctx)
				case 1:
					_, errs[i] = client.ListDNSRecords(ctx)
				case 2:
					_, errs[i] = client.ListAdlists(ctx)
				case 3:
					_, errs[i] = client.ListClients(ctx)
				}
			}(i)
		}
		wg.Wait()
		for i, err := range errs {
			if err != nil {
				t.Fatalf("parallel read %d failed: %v", i, err)
			}
		}
	})
}

func integrationClient(t *testing.T) *Client {
	t.Helper()
	url := os.Getenv("PIHOLE_URL")
	pw := os.Getenv("PIHOLE_PASSWORD")
	if url == "" || pw == "" {
		t.Skip("set PIHOLE_URL and PIHOLE_PASSWORD to run integration tests")
	}
	client, err := NewClient(url, pw)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if err := client.Authenticate(context.Background()); err != nil {
		t.Fatalf("auth against %s: %v", url, err)
	}
	fmt.Fprintf(os.Stderr, "integration: authenticated against %s\n", url)
	return client
}
