# go-pihole

[![Go Reference](https://pkg.go.dev/badge/github.com/barryw/go-pihole.svg)](https://pkg.go.dev/github.com/barryw/go-pihole)
[![CI](https://github.com/barryw/go-pihole/actions/workflows/ci.yml/badge.svg)](https://github.com/barryw/go-pihole/actions/workflows/ci.yml)
[![License: MPL-2.0](https://img.shields.io/badge/License-MPL--2.0-blue.svg)](LICENSE)

Go client library for the [Pi-hole](https://pi-hole.net/) v6 HTTP API. Provides typed access to DNS records, CNAME records, groups, adlists, domain allow/deny lists, and clients.

## Installation

```
go get github.com/barryw/go-pihole
```

Requires Go 1.24 or later.

## Quick start

```go
package main

import (
    "fmt"
    "log"

    pihole "github.com/barryw/go-pihole"
)

func main() {
    client, err := pihole.NewClient("http://pihole.local", "your-app-password")
    if err != nil {
        log.Fatal(err)
    }

    // Create a DNS record
    err = client.CreateDNSRecord("192.168.1.50", "myserver.lan")
    if err != nil {
        log.Fatal(err)
    }

    // List all DNS records
    records, err := client.ListDNSRecords()
    if err != nil {
        log.Fatal(err)
    }
    for _, r := range records {
        fmt.Printf("%s -> %s\n", r.Domain, r.IP)
    }
}
```

## Authentication

This library authenticates using Pi-hole v6 **app-passwords**. Generate one in the Pi-hole web UI under Settings > API.

The client sends the password to `/api/auth` and receives a session ID (`SID`), which is then passed as the `X-FTL-SID` header on all subsequent requests. No CSRF token is needed for API access.

**Session handling is automatic.** The client authenticates lazily on the first request and retries once on HTTP 401 (expired session) before returning an error. You never need to call authenticate manually.

Your Pi-hole must have `app_sudo` enabled for write operations (creating, updating, and deleting resources).

## API Reference

### DNS Records

Local DNS records map a domain name to an IP address.

```go
// List all DNS records
records, err := client.ListDNSRecords()

// Get a single record by domain name
record, err := client.GetDNSRecord("myserver.lan")
// record.IP, record.Domain

// Create a record
err := client.CreateDNSRecord("192.168.1.50", "myserver.lan")

// Delete a record (both IP and domain must match)
err := client.DeleteDNSRecord("192.168.1.50", "myserver.lan")
```

| Method | Signature |
|--------|-----------|
| `ListDNSRecords` | `() ([]DNSRecord, error)` |
| `GetDNSRecord` | `(domain string) (*DNSRecord, error)` |
| `CreateDNSRecord` | `(ip, domain string) error` |
| `DeleteDNSRecord` | `(ip, domain string) error` |

### CNAME Records

CNAME records alias one domain to another, with optional TTL.

```go
// Create a CNAME with a 300-second TTL
err := client.CreateCNAMERecord("app.lan", "loadbalancer.lan", 300)

// Create a CNAME with default TTL (pass 0)
err := client.CreateCNAMERecord("app.lan", "loadbalancer.lan", 0)

// Look up a CNAME by its domain
record, err := client.GetCNAMERecord("app.lan")
// record.Domain, record.Target, record.TTL

// List all CNAME records
records, err := client.ListCNAMERecords()

// Delete (all three fields must match)
err := client.DeleteCNAMERecord("app.lan", "loadbalancer.lan", 300)
```

| Method | Signature |
|--------|-----------|
| `ListCNAMERecords` | `() ([]CNAMERecord, error)` |
| `GetCNAMERecord` | `(domain string) (*CNAMERecord, error)` |
| `CreateCNAMERecord` | `(domain, target string, ttl int) error` |
| `DeleteCNAMERecord` | `(domain, target string, ttl int) error` |

### Groups

Groups organize clients, domains, and adlists together.

```go
// Create a group
group, err := client.CreateGroup(pihole.GroupCreateRequest{
    Name:    "iot-devices",
    Comment: "IoT devices with restricted DNS",
    Enabled: true,
})

// Get a group by name
group, err := client.GetGroup("iot-devices")
// group.ID, group.Name, group.Comment, group.Enabled

// Update a group
group, err := client.UpdateGroup("iot-devices", pihole.GroupUpdateRequest{
    Comment: "Updated comment",
    Enabled: false,
})

// List all groups
groups, err := client.ListGroups()

// Delete a group by name
err := client.DeleteGroup("iot-devices")
```

| Method | Signature |
|--------|-----------|
| `ListGroups` | `() ([]Group, error)` |
| `GetGroup` | `(name string) (*Group, error)` |
| `CreateGroup` | `(req GroupCreateRequest) (*Group, error)` |
| `UpdateGroup` | `(name string, req GroupUpdateRequest) (*Group, error)` |
| `DeleteGroup` | `(name string) error` |

### Adlists

Adlists are blocklist or allowlist URLs that Pi-hole imports. The `type` parameter distinguishes between them (typically `"block"` or `"allow"`).

```go
// Add a blocklist
list, err := client.CreateAdlist(pihole.AdlistCreateRequest{
    Address: "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
    Type:    "block",
    Comment: "Steven Black unified hosts",
    Groups:  []int{0},
    Enabled: true,
})

// Get an adlist by address
list, err := client.GetAdlist("https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts")
// list.ID, list.Address, list.Type, list.Enabled, list.Number

// Update an adlist
list, err := client.UpdateAdlist(
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
lists, err := client.ListAdlists()

// Delete an adlist
err := client.DeleteAdlist(
    "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
    "block",
)
```

| Method | Signature |
|--------|-----------|
| `ListAdlists` | `() ([]Adlist, error)` |
| `GetAdlist` | `(address string) (*Adlist, error)` |
| `CreateAdlist` | `(req AdlistCreateRequest) (*Adlist, error)` |
| `UpdateAdlist` | `(address, listType string, req AdlistUpdateRequest) (*Adlist, error)` |
| `DeleteAdlist` | `(address, listType string) error` |

### Domains (Allow/Deny Lists)

Domain entries control per-domain allow and deny behavior. Each entry has a **type** (`"allow"` or `"deny"`) and a **kind** (`"exact"` or `"regex"`).

```go
// Add an exact deny-list entry
entry, err := client.CreateDomain(pihole.DomainCreateRequest{
    Domain:  "ads.example.com",
    Type:    "deny",
    Kind:    "exact",
    Comment: "Block this domain",
    Groups:  []int{0},
    Enabled: true,
})

// Add a regex allow-list entry
entry, err := client.CreateDomain(pihole.DomainCreateRequest{
    Domain:  `(\.|^)example\.com$`,
    Type:    "allow",
    Kind:    "regex",
    Groups:  []int{0},
    Enabled: true,
})

// Get a specific domain entry
entry, err := client.GetDomain("deny", "exact", "ads.example.com")

// List all domains
domains, err := client.ListDomains()

// List domains filtered by type and kind
domains, err := client.ListDomainsByTypeAndKind("deny", "regex")

// Update a domain entry
entry, err := client.UpdateDomain("deny", "exact", "ads.example.com", pihole.DomainUpdateRequest{
    Type:    "deny",
    Kind:    "exact",
    Comment: "Updated comment",
    Groups:  []int{0, 1},
    Enabled: true,
})

// Delete a domain entry
err := client.DeleteDomain("deny", "exact", "ads.example.com")
```

| Method | Signature |
|--------|-----------|
| `ListDomains` | `() ([]DomainEntry, error)` |
| `ListDomainsByTypeAndKind` | `(domainType, kind string) ([]DomainEntry, error)` |
| `GetDomain` | `(domainType, kind, domain string) (*DomainEntry, error)` |
| `CreateDomain` | `(req DomainCreateRequest) (*DomainEntry, error)` |
| `UpdateDomain` | `(domainType, kind, domain string, req DomainUpdateRequest) (*DomainEntry, error)` |
| `DeleteDomain` | `(domainType, kind, domain string) error` |

### Clients

Clients represent devices (by IP, CIDR, or MAC address) that can be assigned to groups.

```go
// Register a client
c, err := client.CreateClient(pihole.ClientCreateRequest{
    Client:  "192.168.1.100",
    Comment: "Living room TV",
    Groups:  []int{0, 2},
})

// Get a client by its identifier
c, err := client.GetClient("192.168.1.100")
// c.ID, c.Client, c.Comment, c.Groups, c.Enabled

// Update a client
c, err := client.UpdateClient("192.168.1.100", pihole.ClientUpdateRequest{
    Comment: "Updated comment",
    Groups:  []int{0},
})

// List all clients
clients, err := client.ListClients()

// Delete a client
err := client.DeleteClient("192.168.1.100")
```

| Method | Signature |
|--------|-----------|
| `ListClients` | `() ([]PiholeClient, error)` |
| `GetClient` | `(clientID string) (*PiholeClient, error)` |
| `CreateClient` | `(req ClientCreateRequest) (*PiholeClient, error)` |
| `UpdateClient` | `(clientID string, req ClientUpdateRequest) (*PiholeClient, error)` |
| `DeleteClient` | `(clientID string) error` |

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

- **Go 1.24+**
- **Pi-hole v6** with the HTTP API enabled
- An **app-password** configured in the Pi-hole web UI (Settings > API)
- **app_sudo** enabled for write operations (enabled by default when using app-passwords)

## License

This project is licensed under the [Mozilla Public License 2.0](LICENSE).

## Related

- [terraform-provider-pihole-v6](https://github.com/barryw/terraform-provider-pihole-v6) -- OpenTofu / Terraform provider built on this library
