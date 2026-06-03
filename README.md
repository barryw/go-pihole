# go-pihole

[![Go Reference](https://pkg.go.dev/badge/github.com/barryw/go-pihole.svg)](https://pkg.go.dev/github.com/barryw/go-pihole)
[![Build Status](https://ci.barrywalker.io/api/badges/barryw/go-pihole/status.svg)](https://ci.barrywalker.io/repos/barryw/go-pihole)
[![Latest Release](https://img.shields.io/github/v/release/barryw/go-pihole)](https://github.com/barryw/go-pihole/releases/latest)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL--2.0-blue.svg)](LICENSE)

Go client library (v6 API) for the [Pi-hole](https://pi-hole.net/) v6 HTTP API. Provides typed access to DNS records, CNAME records, groups, adlists, domain allow/deny lists, clients, DHCP static leases, configuration settings, DNS blocking, and upstream DNS servers.

**v1.0.0 is a breaking release: every public method now takes a `context.Context` as its first argument.** See [Migrating to v1](#migrating-to-v1) for details.

## Installation

```
go get github.com/barryw/go-pihole
```

Requires Go 1.25 or later.

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"

    pihole "github.com/barryw/go-pihole"
)

func main() {
    client, err := pihole.NewClient("http://pihole.local", "your-app-password")
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Create a DNS record
    err = client.CreateDNSRecord(ctx, "192.168.1.50", "myserver.lan")
    if err != nil {
        log.Fatal(err)
    }

    // List all DNS records
    records, err := client.ListDNSRecords(ctx)
    if err != nil {
        log.Fatal(err)
    }
    for _, r := range records {
        fmt.Printf("%s -> %s\n", r.Domain, r.IP)
    }
}
```

## Migrating to v1

v1.0.0 is a breaking release. The only change required of callers is that **every public method now takes a `context.Context` as its first argument.** No other signatures or return types changed.

```go
// Before (v0.x)
records, err := client.ListDNSRecords()
err := client.CreateDNSRecord("192.168.1.50", "myserver.lan")
err := client.Authenticate()

// After (v1)
ctx := context.Background()
records, err := client.ListDNSRecords(ctx)
err := client.CreateDNSRecord(ctx, "192.168.1.50", "myserver.lan")
err := client.Authenticate(ctx)
```

The mechanical fix is to thread a `context.Context` through your call sites and pass it as the first argument to every client call. This lets you set deadlines and cancel in-flight requests (including during the library's retry/backoff loop).

v1 also adds two new feature areas: DNS blocking (`GetDNSBlocking` / `SetDNSBlocking`) and upstream DNS servers (`GetDNSUpstreams` / `SetDNSUpstreams`). See the [API Reference](#api-reference) for details.

## Authentication

This library authenticates using Pi-hole v6 **app-passwords**. Generate one in the Pi-hole web UI under Settings > API.

The client sends the password to `/api/auth` and receives a session ID (`SID`), which is then passed as the `X-FTL-SID` header on all subsequent requests. No CSRF token is needed for API access.

**Session handling is automatic and concurrency-safe.** The client authenticates lazily on the first request, or eagerly if you call `Authenticate(ctx)` yourself. A burst of parallel requests against a fresh client triggers a single authentication rather than one per goroutine, and an HTTP 401 (expired session) triggers a single coordinated re-authentication shared across all in-flight requests rather than one re-auth per goroutine. Transient network errors (e.g. connection refused while Pi-hole reloads its config after a write) and 401s are retried with backoff that respects context cancellation. Every request is bounded by a default 30-second HTTP timeout. You never need to call authenticate manually.

A wrong app-password is correctly detected: Pi-hole v6 returns `valid=true` with a null `sid` even on a bad password, so the client treats an empty `sid` as the real failure signal and returns an `ErrAuth`.

Your Pi-hole must have `app_sudo` enabled for write operations (creating, updating, and deleting resources).

## API Reference

### DNS Records

Local DNS records map a domain name to an IP address.

```go
// List all DNS records
records, err := client.ListDNSRecords(ctx)

// Get a single record by domain name
record, err := client.GetDNSRecord(ctx, "myserver.lan")
// record.IP, record.Domain

// Create a record
err := client.CreateDNSRecord(ctx, "192.168.1.50", "myserver.lan")

// Delete a record (both IP and domain must match)
err := client.DeleteDNSRecord(ctx, "192.168.1.50", "myserver.lan")
```

| Method | Signature |
|--------|-----------|
| `ListDNSRecords` | `(ctx context.Context) ([]DNSRecord, error)` |
| `GetDNSRecord` | `(ctx context.Context, domain string) (*DNSRecord, error)` |
| `CreateDNSRecord` | `(ctx context.Context, ip, domain string) error` |
| `DeleteDNSRecord` | `(ctx context.Context, ip, domain string) error` |

### CNAME Records

CNAME records alias one domain to another, with optional TTL.

```go
// Create a CNAME with a 300-second TTL
err := client.CreateCNAMERecord(ctx, "app.lan", "loadbalancer.lan", 300)

// Create a CNAME with default TTL (pass 0)
err := client.CreateCNAMERecord(ctx, "app.lan", "loadbalancer.lan", 0)

// Look up a CNAME by its domain
record, err := client.GetCNAMERecord(ctx, "app.lan")
// record.Domain, record.Target, record.TTL

// List all CNAME records
records, err := client.ListCNAMERecords(ctx)

// Delete (all three fields must match)
err := client.DeleteCNAMERecord(ctx, "app.lan", "loadbalancer.lan", 300)
```

| Method | Signature |
|--------|-----------|
| `ListCNAMERecords` | `(ctx context.Context) ([]CNAMERecord, error)` |
| `GetCNAMERecord` | `(ctx context.Context, domain string) (*CNAMERecord, error)` |
| `CreateCNAMERecord` | `(ctx context.Context, domain, target string, ttl int) error` |
| `DeleteCNAMERecord` | `(ctx context.Context, domain, target string, ttl int) error` |

### Groups

Groups organize clients, domains, and adlists together.

```go
// Create a group
group, err := client.CreateGroup(ctx, pihole.GroupCreateRequest{
    Name:    "iot-devices",
    Comment: "IoT devices with restricted DNS",
    Enabled: true,
})

// Get a group by name
group, err := client.GetGroup(ctx, "iot-devices")
// group.ID, group.Name, group.Comment, group.Enabled

// Update a group
group, err := client.UpdateGroup(ctx, "iot-devices", pihole.GroupUpdateRequest{
    Comment: "Updated comment",
    Enabled: false,
})

// List all groups
groups, err := client.ListGroups(ctx)

// Delete a group by name
err := client.DeleteGroup(ctx, "iot-devices")
```

| Method | Signature |
|--------|-----------|
| `ListGroups` | `(ctx context.Context) ([]Group, error)` |
| `GetGroup` | `(ctx context.Context, name string) (*Group, error)` |
| `CreateGroup` | `(ctx context.Context, req GroupCreateRequest) (*Group, error)` |
| `UpdateGroup` | `(ctx context.Context, name string, req GroupUpdateRequest) (*Group, error)` |
| `DeleteGroup` | `(ctx context.Context, name string) error` |

### Adlists

Adlists are blocklist or allowlist URLs that Pi-hole imports. The `type` parameter distinguishes between them (typically `"block"` or `"allow"`).

```go
// Add a blocklist
list, err := client.CreateAdlist(ctx, pihole.AdlistCreateRequest{
    Address: "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
    Type:    "block",
    Comment: "Steven Black unified hosts",
    Groups:  []int{0},
    Enabled: true,
})

// Get an adlist by address
list, err := client.GetAdlist(ctx, "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts")
// list.ID, list.Address, list.Type, list.Enabled, list.Number

// Update an adlist
list, err := client.UpdateAdlist(
    ctx,
    "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
    "block",
    pihole.AdlistUpdateRequest{
        Comment: "Updated comment",
        Type:    "block",
        Groups:  []int{0, 1},
        Enabled: true,
    },
)

// List all adlists
lists, err := client.ListAdlists(ctx)

// Delete an adlist
err := client.DeleteAdlist(
    ctx,
    "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
    "block",
)
```

| Method | Signature |
|--------|-----------|
| `ListAdlists` | `(ctx context.Context) ([]Adlist, error)` |
| `GetAdlist` | `(ctx context.Context, address string) (*Adlist, error)` |
| `CreateAdlist` | `(ctx context.Context, req AdlistCreateRequest) (*Adlist, error)` |
| `UpdateAdlist` | `(ctx context.Context, address, listType string, req AdlistUpdateRequest) (*Adlist, error)` |
| `DeleteAdlist` | `(ctx context.Context, address, listType string) error` |

### Domains (Allow/Deny Lists)

Domain entries control per-domain allow and deny behavior. Each entry has a **type** (`"allow"` or `"deny"`) and a **kind** (`"exact"` or `"regex"`).

```go
// Add an exact deny-list entry
entry, err := client.CreateDomain(ctx, pihole.DomainCreateRequest{
    Domain:  "ads.example.com",
    Type:    "deny",
    Kind:    "exact",
    Comment: "Block this domain",
    Groups:  []int{0},
    Enabled: true,
})

// Add a regex allow-list entry
entry, err := client.CreateDomain(ctx, pihole.DomainCreateRequest{
    Domain:  `(\.|^)example\.com$`,
    Type:    "allow",
    Kind:    "regex",
    Groups:  []int{0},
    Enabled: true,
})

// Get a specific domain entry
entry, err := client.GetDomain(ctx, "deny", "exact", "ads.example.com")

// List all domains
domains, err := client.ListDomains(ctx)

// List domains filtered by type and kind
domains, err := client.ListDomainsByTypeAndKind(ctx, "deny", "regex")

// Update a domain entry
entry, err := client.UpdateDomain(ctx, "deny", "exact", "ads.example.com", pihole.DomainUpdateRequest{
    Type:    "deny",
    Kind:    "exact",
    Comment: "Updated comment",
    Groups:  []int{0, 1},
    Enabled: true,
})

// Delete a domain entry
err := client.DeleteDomain(ctx, "deny", "exact", "ads.example.com")
```

| Method | Signature |
|--------|-----------|
| `ListDomains` | `(ctx context.Context) ([]DomainEntry, error)` |
| `ListDomainsByTypeAndKind` | `(ctx context.Context, domainType, kind string) ([]DomainEntry, error)` |
| `GetDomain` | `(ctx context.Context, domainType, kind, domain string) (*DomainEntry, error)` |
| `CreateDomain` | `(ctx context.Context, req DomainCreateRequest) (*DomainEntry, error)` |
| `UpdateDomain` | `(ctx context.Context, domainType, kind, domain string, req DomainUpdateRequest) (*DomainEntry, error)` |
| `DeleteDomain` | `(ctx context.Context, domainType, kind, domain string) error` |

### Clients

Clients represent devices (by IP, CIDR, or MAC address) that can be assigned to groups.

```go
// Register a client
c, err := client.CreateClient(ctx, pihole.ClientCreateRequest{
    Client:  "192.168.1.100",
    Comment: "Living room TV",
    Groups:  []int{0, 2},
})

// Get a client by its identifier
c, err := client.GetClient(ctx, "192.168.1.100")
// c.ID, c.Client, c.Comment, c.Groups, c.Enabled

// Update a client
c, err := client.UpdateClient(ctx, "192.168.1.100", pihole.ClientUpdateRequest{
    Comment: "Updated comment",
    Groups:  []int{0},
})

// List all clients
clients, err := client.ListClients(ctx)

// Delete a client
err := client.DeleteClient(ctx, "192.168.1.100")
```

| Method | Signature |
|--------|-----------|
| `ListClients` | `(ctx context.Context) ([]PiholeClient, error)` |
| `GetClient` | `(ctx context.Context, clientID string) (*PiholeClient, error)` |
| `CreateClient` | `(ctx context.Context, req ClientCreateRequest) (*PiholeClient, error)` |
| `UpdateClient` | `(ctx context.Context, clientID string, req ClientUpdateRequest) (*PiholeClient, error)` |
| `DeleteClient` | `(ctx context.Context, clientID string) error` |

### DNS Blocking

DNS blocking is Pi-hole's master on/off switch for ad/DNS filtering. It can be toggled permanently or temporarily with an automatic revert timer.

```go
// Check the current blocking state
status, err := client.GetDNSBlocking(ctx)
// status.Enabled is true when blocking is active.
// status.Timer is the number of seconds until blocking reverts to its
// previous state, or nil when no timer is active.

// Enable blocking permanently
err := client.SetDNSBlocking(ctx, true, nil)

// Disable blocking for 60 seconds, then revert automatically
seconds := 60
err := client.SetDNSBlocking(ctx, false, &seconds)
```

| Method | Signature |
|--------|-----------|
| `GetDNSBlocking` | `(ctx context.Context) (*DNSBlockingStatus, error)` |
| `SetDNSBlocking` | `(ctx context.Context, enabled bool, timerSeconds *int) error` |

`DNSBlockingStatus` has fields `Enabled bool` and `Timer *float64`.

### DNS Upstreams

Upstream DNS servers are the resolvers Pi-hole forwards queries to (e.g. `"1.1.1.1"`, `"8.8.8.8#5353"`). The setter replaces the entire ordered list.

```go
// Read the current upstream list
upstreams, err := client.GetDNSUpstreams(ctx)
// e.g. []string{"1.1.1.1", "8.8.8.8"}

// Replace the upstream list
err := client.SetDNSUpstreams(ctx, []string{"9.9.9.9", "149.112.112.112"})
```

| Method | Signature |
|--------|-----------|
| `GetDNSUpstreams` | `(ctx context.Context) ([]string, error)` |
| `SetDNSUpstreams` | `(ctx context.Context, upstreams []string) error` |

## Error handling

The library defines three error types. Use type assertions or `errors.As` to inspect them.

### APIError

Returned when the Pi-hole API responds with a non-success status code. Contains the HTTP status code and structured error details from the API response.

```go
var apiErr *pihole.APIError
if errors.As(err, &apiErr) {
    fmt.Println(apiErr.StatusCode) // e.g. 400
    fmt.Println(apiErr.Key)        // e.g. "database_busy"
    fmt.Println(apiErr.Message)    // human-readable message
    fmt.Println(apiErr.Hint)       // optional hint from the API
}
```

### ErrNotFound

Returned when a requested resource does not exist (HTTP 404 or an empty result set).

```go
var notFound *pihole.ErrNotFound
if errors.As(err, &notFound) {
    fmt.Println(notFound.Resource) // e.g. "DNS record"
    fmt.Println(notFound.ID)       // e.g. "myserver.lan"
}
```

### ErrAuth

Returned when authentication fails -- either the password is wrong, the session expired and re-authentication failed, or the session is invalid.

```go
var authErr *pihole.ErrAuth
if errors.As(err, &authErr) {
    fmt.Println(authErr.Message)
}
```

## Requirements

- **Go 1.25+**
- **Pi-hole v6** with the HTTP API enabled
- An **app-password** configured in the Pi-hole web UI (Settings > API)
- **app_sudo** enabled for write operations (enabled by default when using app-passwords)

## License

This project is licensed under the [Mozilla Public License 2.0](LICENSE).

## Related

- [terraform-provider-pihole-v6](https://github.com/barryw/terraform-provider-pihole-v6) -- OpenTofu / Terraform provider built on this library
