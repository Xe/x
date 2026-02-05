# Sakurajima - Project Memory

This document contains patterns, conventions, and insights specific to the Sakurajima reverse proxy.

## Project Overview

Sakurajima is a reverse proxy service that uses HCL for configuration, supports HTTP/2 Cleartext (H2C), and includes comprehensive timeout, limit, and SSRF protection.

## Coding Patterns & Conventions

### HCL Configuration

The project uses HCL for configuration with clear separation of concerns:

- **Domain blocks**: `domain` blocks define reverse proxy targets
- **Nested blocks**: Use nested blocks for complex configurations:
  - `tls`: TLS/certificate settings
  - `timeouts`: Connection and request timeout settings
  - `limits`: Size and rate limiting settings

Example structure:
```hcl
domain "example.com" {
  tls {
    # TLS settings
  }
  timeouts {
    # Timeout settings
  }
  limits {
    # Limit settings
  }
}
```

### Reverse Proxy Pattern

Use `net/http/httputil.ReverseProxy` for proxying requests to backend services:

```go
rp := &httputil.ReverseProxy{
    Director: func(req *http.Request) {
        // Modify request for backend
    },
    Transport: customTransport,
}
```

Apply custom `Director` and `Transport` as needed for each domain.

### HTTP Transport Configuration

**For HTTP/1.1 and HTTPS:**

```go
transport := &http.Transport{
    DialContext: (&net.Dialer{
        Timeout:   dialTimeout,
        KeepAlive: keepAlive,
    }).DialContext,
    ResponseHeaderTimeout: responseHeaderTimeout,
    IdleConnTimeout:       idleConnTimeout,
    TLSClientConfig:       tlsConfig,
}
```

**For HTTP/2 Cleartext (H2C):**

```go
transport := &http2.Transport{
    DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
        // H2C dialing logic
    },
    IdleConnTimeout: idleConnTimeout,
}
```

Key timeout fields to understand:
- `DialContext`: Initial connection timeout
- `ResponseHeaderTimeout`: Time to wait for response headers
- `IdleConnTimeout`: How long to keep idle connections open

### Context for Timeouts

Use `context.WithTimeout` with `Dialer.DialContext` to implement request timeouts:

```go
ctx, cancel := context.WithTimeout(context.Background(), timeout)
defer cancel()
conn, err := dialer.DialContext(ctx, network, addr)
```

### atomic.Value for Hot-Reloading

Use `atomic.Value` to safely update logger instances during configuration reload:

```go
var log atomic.Value
log.Store(newLogger)

// Later
logger := log.Load().(*slog.Logger)
```

This allows `*lumberjack.Logger` and `*slog.Logger` to be updated without restarting the service.

### Table-Driven Tests

Follow Go's table-driven test pattern for thorough testing:

```go
func TestTimeoutsParse(t *testing.T) {
    tests := []struct {
        name     string
        input    map[string]any
        want     time.Duration
        wantErr  bool
    }{
        {"valid second", map[string]any{"read": "5s"}, 5 * time.Second, false},
        {"invalid", map[string]any{"read": "invalid"}, 0, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test logic
        })
    }
}
```

Tests are co-located with source files (`*_test.go`).

## User Preferences

### Human-Readable Durations

Timeout values in HCL should be human-readable strings like `"5s"`, `"100ms"`, `"1h"`, parsed using `time.ParseDuration`.

### Sensible Defaults

When timeout values are not explicitly configured, use sensible defaults (e.g., 30s for reads, 10s for dial).

### Clear Task Management

When implementing multi-step features, use numbered tasks and status updates to track progress.

## Technical Insights

### time.ParseDuration

The standard `time.ParseDuration` function parses human-readable duration strings:
- `"5s"` → 5 seconds
- `"100ms"` → 100 milliseconds
- `"1h30m"` → 1 hour 30 minutes

### H2C with http2.Transport

The `golang.org/x/net/http2` package provides `http2.Transport` for H2C connections. Configure `DialTLS` even for cleartext connections.

### Autocert Integration

`autocert.Manager` is used for automatic TLS certificate acquisition:
```go
manager := &autocert.Manager{
    Cache:      autocert.DirCache("cache-dir"),
    Prompt:     autocert.AcceptTOS,
    HostPolicy: autocert.HostWhitelist("example.com"),
}
```

Can be configured with custom cache backends (like S3).

### SSRF Protection

The project implements SSRF protection by validating target URLs against private IP ranges. Users can override with `allow_private_target` in the domain configuration.

## Best Practices

### Validation of Configuration

Use a `Valid()` method to validate all aspects of a domain's configuration early in the process:
```go
func (d *Domain) Valid() error {
    if err := d.NameValid(); err != nil {
        return err
    }
    if err := d.Timeouts.Parse(); err != nil {
        return err
    }
    // ... other validations
    return nil
}
```

### Helper Functions

Create helper functions to promote code reusability:
```go
func newTransport(d *config.Domain) (*http.Transport, error) {
    // Encapsulate complex transport setup
}
```

### Comprehensive Testing

- Unit tests for parsing logic (e.g., `TestTimeoutsParse`, `TestDomainValid`)
- Integration tests for core functionality (e.g., `TestH2CReverseProxy`, `TestRouterSetConfig`)
- Race condition tests with `-race` tag where appropriate (e.g., `router_race_test.go`)

### Clear Commit Messages

Follow Conventional Commits format:
```
feat(sakurajima): add HTTP request timeouts to prevent hanging connections

- Add DialTimeout, ResponseHeaderTimeout, IdleConnTimeout to http.Transport
- Add DialTLS timeout to http2.Transport for H2C
- Parse human-readable duration strings from HCL config

Fixes XE-28
```

## Common Pitfalls

### Ignoring Errors

Do not ignore errors from configuration parsing. Invalid timeout configurations should fail explicitly:
```go
if err := d.Timeouts.Parse(); err != nil {
    return fmt.Errorf("timeout parsing failed: %w", err)
}
```

### Unused Imports

Always remove unused imports. Failing to do so will cause build failures:
```go
// Bad: unused "time" import after removing time.Duration usage
import "time"
```

### Validation Timing

Validate configurations early (at parse/load time), not when the configuration is applied. This prevents invalid configs from being partially applied.
