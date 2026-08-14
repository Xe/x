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

### HTTP Transport Configuration

Configure `http.Transport` with explicit `DialContext`, `ResponseHeaderTimeout`, and `IdleConnTimeout` values from the parsed HCL config; `IdleConnTimeout` is frequently overlooked but essential for connection reuse.

For HTTP/2 Cleartext (H2C), use `http2.Transport` with a custom `DialTLS` that ignores the TLS config and dials plain TCP.

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

## User Preferences

### Human-Readable Durations

Timeout values in HCL should be human-readable strings like `"5s"`, `"100ms"`, `"1h"`, parsed using `time.ParseDuration`.

### Sensible Defaults

When timeout values are not explicitly configured, use sensible defaults (e.g., 30s for reads, 10s for dial).

### Clear Task Management

When implementing multi-step features, use numbered tasks and status updates to track progress.

## Technical Insights

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

## Common Pitfalls

### Ignoring Errors

Do not ignore errors from configuration parsing. Invalid timeout configurations should fail explicitly:

```go
if err := d.Timeouts.Parse(); err != nil {
    return fmt.Errorf("timeout parsing failed: %w", err)
}
```

### Validation Timing

Validate configurations early (at parse/load time), not when the configuration is applied. This prevents invalid configs from being partially applied.
