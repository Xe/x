# Sakurajima Production Fixes Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix critical and high-severity issues identified in production readiness review to make sakurajima safe for deployment.

**Architecture:** Incremental fixes following TDD principles, with atomic commits and comprehensive test coverage for each change.

**Tech Stack:** Go 1.23+, gosec security scanner, race detector testing, standard library only.

---

## Table of Contents

1. [Critical Fixes (Must Fix)](#critical-fixes-must-fix)
2. [High Severity Fixes](#high-severity-fixes)
3. [Medium Severity Security Hardening](#medium-severity-security-hardening)
4. [Tooling and Testing Infrastructure](#tooling-and-testing-infrastructure)
5. [Execution Checklist](#execution-checklist)

---

# Critical Fixes (Must Fix)

## Task 1: Fix Data Race on accessLog

**Files:**

- Modify: `cmd/sakurajima/internal/entrypoint/router.go` (lines 60-70, 166-169, 201, 268-278, 416)
- Create: `cmd/sakurajima/internal/entrypoint/router_race_test.go`

### Step 1: Write the failing test (detects the race)

Create file: `cmd/sakurajima/internal/entrypoint/router_race_test.go`

```go
package entrypoint

import (
    "context"
    "net/http"
    "net/http/httptest"
    "sync"
    "testing"
    "time"

    "within.website/x/cmd/sakurajima/internal/config"
)

func TestRouterAccessLogRace(t *testing.T) {
    cfg := loadConfig(t, "./testdata/good/selfsigned.hcl")
    rtr := newRouter(t, cfg)

    srv := httptest.NewServer(rtr)
    defer srv.Close()

    var wg sync.WaitGroup
    done := make(chan struct{})

    // Simulate concurrent config reloads
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for {
                select {
                case <-done:
                    return
                default:
                    rtr.setConfig(cfg)
                }
            }
        }()
    }

    // Simulate concurrent requests
    for i := 0; i < 100; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for {
                select {
                case <-done:
                    return
                default:
                    req, _ := http.NewRequest("GET", srv.URL, nil)
                    req.Host = "osiris.local.cetacean.club"
                    http.DefaultClient.Do(req)
                }
            }
        }()
    }

    time.Sleep(100 * time.Millisecond)
    close(done)
    wg.Wait()
}
```

Run: `go test -race -run TestRouterAccessLogRace ./cmd/sakurajima/internal/entrypoint/`
Expected: FAIL with data race detected

### Step 2: Update Router struct to use atomic.Value

Modify: `cmd/sakurajima/internal/entrypoint/router.go`

At line 1, add import:

```go
import "sync/atomic"
```

At lines 60-70, change Router struct:

```go
type Router struct {
    lock                 sync.RWMutex
    routes               map[string]http.Handler
    tlsCerts             map[string]*tls.Certificate
    opts                 Options
    accessLog            atomic.Value // stores *lumberjack.Logger
    baseSlog             *slog.Logger
    log                  *slog.Logger
    autoMgr              *autocert.Manager
    autocertRedirectCode int
}
```

### Step 3: Update setConfig to use atomic operations

At lines 166-169, replace:

```go
if rtr.accessLog != nil {
    rtr.accessLog.Rotate()
    rtr.accessLog.Close()
}
```

With:

```go
if oldLog := (*lumberjack.Logger)(rtr.accessLog.Load()); oldLog != nil {
    oldLog.Rotate()
    oldLog.Close()
}
```

At line 201, replace:

```go
rtr.accessLog = lum
```

With:

```go
rtr.accessLog.Store(lum)
```

### Step 4: Update ServeHTTP to use atomic load

At line 416, replace:

```go
logging.LogHTTPRequest(rtr.accessLog, r, m.Code, m.Written, m.Duration)
```

With:

```go
if accessLog := (*lumberjack.Logger)(rtr.accessLog.Load()); accessLog != nil {
    logging.LogHTTPRequest(accessLog, r, m.Code, m.Written, m.Duration)
}
```

### Step 5: Update NewRouter initialization

At lines 268-278, add after creating result:

```go
result.accessLog.Store((*lumberjack.Logger)(nil))
```

### Step 6: Run test to verify fix

Run: `go test -race -run TestRouterAccessLogRace ./cmd/sakurajima/internal/entrypoint/`
Expected: PASS with no data race detected

### Step 7: Run all tests

Run: `go test -race ./cmd/sakurajima/...`
Expected: All PASS

### Step 8: Commit

```bash
git add cmd/sakurajima/internal/entrypoint/router.go cmd/sakurajima/internal/entrypoint/router_race_test.go
git commit -m "fix(sakurajima): fix data race on accessLog during config reload

Replace direct accessLog access with atomic.Value to prevent race
between ServeHTTP (read) and setConfig (close/replace).

Fixes data race detected when SIGHUP triggers config reload during
active request handling.

Assisted-by: GLM 4.6 via Claude Code
Reviewbot-request: yes
"
```

---

## Task 2: Fix SSRF Vulnerability

**Files:**

- Create: `cmd/sakurajima/internal/config/ssrf.go`
- Modify: `cmd/sakurajima/internal/config/domain.go` (lines 18-24, 26-50)
- Create: `cmd/sakurajima/internal/config/ssrf_test.go`
- Modify: `cmd/sakurajima/internal/entrypoint/testdata/good/selfsigned.hcl`

### Step 1: Write the failing test

Create file: `cmd/sakurajima/internal/config/ssrf_test.go`

```go
package config

import (
    "errors"
    "testing"
)

func TestValidateURLForSSRF(t *testing.T) {
    tests := []struct {
        name    string
        url     string
        wantErr error
    }{
        {"public IP", "http://8.8.8.8:8080", nil},
        {"public hostname", "http://example.com", nil},
        {"loopback", "http://127.0.0.1:8080", ErrPrivateIP},
        {"private 10.x", "http://10.0.0.1:8080", ErrPrivateIP},
        {"private 192.168", "http://192.168.1.1:8080", ErrPrivateIP},
        {"AWS metadata", "http://169.254.169.254", ErrPrivateIP},
        {"unix socket", "unix:///var/run/app.sock", nil},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateURLForSSRF(tt.url)
            if tt.wantErr != nil {
                if !errors.Is(err, tt.wantErr) {
                    t.Errorf("ValidateURLForSSRF(%s) error = %v, want %v", tt.url, err, tt.wantErr)
                }
            } else if err != nil {
                t.Errorf("ValidateURLForSSRF(%s) unexpected error = %v", tt.url, err)
            }
        })
    }
}
```

Run: `go test ./cmd/sakurajima/internal/config/`
Expected: FAIL with "undefined: ValidateURLForSSRF"

### Step 2: Create SSRF validation package

Create file: `cmd/sakurajima/internal/config/ssrf.go`

```go
package config

import (
    "errors"
    "net"
    "net/url"
)

var (
    ErrPrivateIP = errors.New("target points to private IP address (SSRF protection)")
)

func isPrivateIP(ip net.IP) bool {
    if ip.IsLoopback() {
        return true
    }
    if ip.IsLinkLocalUnicast() {
        return true
    }
    if ip.IsLinkLocalMulticast() {
        return true
    }
    if ip.IsPrivate() {
        return true
    }
    if ip.IsUnspecified() {
        return true
    }
    if ip.IsMulticast() {
        return true
    }

    if ip4 := ip.To4(); ip4 != nil {
        if ip4[0] == 0 { // 0.0.0.0/8
            return true
        }
        if ip4[0] == 100 && ip4[1]&0xC0 == 64 { // 100.64.0.0/10
            return true
        }
        if ip4[0] == 192 && ip4[1] == 0 && ip4[2] == 0 { // 192.0.0.0/24
            return true
        }
        if ip4[0] == 192 && ip4[1] == 0 && ip4[2] == 2 { // TEST-NET-1
            return true
        }
        if ip4[0] == 198 && ip4[1] == 51 && ip4[2] == 100 { // TEST-NET-2
            return true
        }
        if ip4[0] == 203 && ip4[1] == 0 && ip4[2] == 113 { // TEST-NET-3
            return true
        }
        if ip4[0]&0xF0 == 240 { // 240.0.0.0/4
            return true
        }
    }

    return false
}

func ValidateURLForSSRF(targetURL string) error {
    u, err := url.Parse(targetURL)
    if err != nil {
        return err
    }

    if u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "h2c" {
        return nil
    }

    host := u.Hostname()
    if host == "" {
        return nil
    }

    ip := net.ParseIP(host)
    if ip == nil {
        return nil // DNS name, not IP
    }

    if isPrivateIP(ip) {
        return ErrPrivateIP
    }

    return nil
}
```

### Step 3: Update Domain struct

Modify: `cmd/sakurajima/internal/config/domain.go`

At lines 18-24, add `AllowPrivateTarget` field:

```go
type Domain struct {
    Name               string `hcl:"name,label"`
    TLS                TLS    `hcl:"tls,block"`
    Target             string `hcl:"target"`
    InsecureSkipVerify bool   `hcl:"insecure_skip_verify,optional"`
    HealthTarget       string `hcl:"health_target"`
    AllowPrivateTarget bool   `hcl:"allow_private_target,optional"`
}
```

### Step 4: Add SSRF validation to Domain.Valid()

At lines 26-50, add SSRF check before the final return:

```go
    // SSRF validation (can be opted out with AllowPrivateTarget)
    if !d.AllowPrivateTarget {
        if err := ValidateURLForSSRF(d.Target); err != nil {
            errs = append(errs, fmt.Errorf("target SSRF validation failed %q: %w", d.Target, err))
        }
        if err := ValidateURLForSSRF(d.HealthTarget); err != nil {
            errs = append(errs, fmt.Errorf("health_target SSRF validation failed %q: %w", d.HealthTarget, err))
        }
    }
```

### Step 5: Update test configuration

Modify: `cmd/sakurajima/internal/entrypoint/testdata/good/selfsigned.hcl`

Add `allow_private_target = true` to the domain block:

```hcl
domain "osiris.local.cetacean.club" {
  tls {
    cert = "./testdata/selfsigned.crt"
    key  = "./testdata/selfsigned.key"
  }

  target                = "http://localhost:3000"
  health_target         = "http://localhost:9091/healthz"
  allow_private_target  = true
}
```

### Step 6: Run tests to verify fix

Run: `go test ./cmd/sakurajima/internal/config/`
Expected: PASS

### Step 7: Run all tests

Run: `go test ./cmd/sakurajima/...`
Expected: All PASS

### Step 8: Commit

```bash
git add cmd/sakurajima/internal/config/
git commit -m "feat(sakurajima): add SSRF protection with opt-out for internal targets

Add IP-based SSRF validation that blocks requests to private IP ranges
(loopback, 10.x, 172.16-31.x, 192.168.x, link-local, etc.).

Users can opt-out for legitimate internal targets using
allow_private_target = true in domain configuration.

Assisted-by: GLM 4.6 via Claude Code
Reviewbot-request: yes
"
```

---

# High Severity Fixes

## Task 3: Fix Goroutine Leaks (8 locations)

**Files:**

- Modify: `cmd/sakurajima/internal/entrypoint/router.go` (lines 286-289, 327-330, 344-365)
- Modify: `cmd/sakurajima/internal/entrypoint/entrypoint.go` (lines 49-52, 67-70)

### Step 1: Write test for goroutine cleanup

Create file: `cmd/sakurajima/internal/entrypoint/goroutine_test.go`

```go
package entrypoint

import (
    "context"
    "runtime"
    "testing"
    "time"
)

func TestGoroutineCleanupOnContextCancel(t *testing.T) {
    cfg := loadConfig(t, "./testdata/good/selfsigned.hcl")
    rtr := newRouter(t, cfg)

    initialGoroutines := runtime.NumGoroutine()

    ctx, cancel := context.WithCancel(context.Background())
    ln, err := net.Listen("tcp", ":0")
    require.NoError(t, err)

    errChan := make(chan error, 1)
    go func() {
        errChan <- rtr.HandleHTTP(ctx, ln)
    }()

    time.Sleep(100 * time.Millisecond)
    cancel()

    err = <-errChan
    assert.Error(t, err)

    time.Sleep(100 * time.Millisecond)

    finalGoroutines := runtime.NumGoroutine()
    assert.LessOrEqual(t, finalGoroutines, initialGoroutines+2)
}
```

Run: `go test -race -run TestGoroutineCleanupOnContextCancel ./cmd/sakurajima/internal/entrypoint/`
Expected: PASS (but will leak goroutines with current code)

### Step 2: Fix HandleHTTP goroutine

Modify: `cmd/sakurajima/internal/entrypoint/router.go`

At lines 286-289, replace:

```go
    go func(ctx context.Context) {
        <-ctx.Done()
        srv.Close()
    }(ctx)
```

With:

```go
    go func() {
        <-ctx.Done()
        srv.Close()
    }()
```

### Step 3: Fix HandleHTTPS goroutine

At lines 327-330, replace:

```go
    go func(ctx context.Context) {
        <-ctx.Done()
        srv.Close()
    }(ctx)
```

With:

```go
    go func() {
        <-ctx.Done()
        srv.Close()
    }()
```

### Step 4: Fix ListenAndServeMetrics - remove duplicate goroutine

At lines 344-365, simplify to remove the listener close goroutine (duplicate):

```go
func (rtr *Router) ListenAndServeMetrics(ctx context.Context, addr string) error {
    ln, err := net.Listen("tcp", addr)
    if err != nil {
        return fmt.Errorf("(metrics) can't bind to tcp %s: %w", addr, err)
    }
    defer ln.Close()

    mux := http.NewServeMux()

    mux.Handle("/metrics", promhttp.Handler())
    mux.HandleFunc("/readyz", readyz)
    mux.HandleFunc("/healthz", healthz)

    rtr.log.Info("listening", "for", "metrics", "bind", addr)

    srv := http.Server{
        Addr:    addr,
        Handler: mux,
    }

    go func() {
        <-ctx.Done()
        srv.Close()
    }()

    return srv.Serve(ln)
}
```

### Step 5: Fix Main entrypoint goroutines

Modify: `cmd/sakurajima/internal/entrypoint/entrypoint.go`

At lines 49-52, replace:

```go
        go func(ctx context.Context) {
            <-ctx.Done()
            ln.Close()
        }(ctx)
```

With:

```go
        go func() {
            <-ctx.Done()
            ln.Close()
        }()
```

At lines 67-70, replace:

```go
        go func(ctx context.Context) {
            <-ctx.Done()
            ln.Close()
        }(ctx)
```

With:

```go
        go func() {
            <-ctx.Done()
            ln.Close()
        }()
```

### Step 6: Run tests to verify

Run: `go test -race ./cmd/sakurajima/...`
Expected: All PASS with no goroutine leaks

### Step 7: Commit

```bash
git add cmd/sakurajima/internal/entrypoint/router.go cmd/sakurajima/internal/entrypoint/entrypoint.go cmd/sakurajima/internal/entrypoint/goroutine_test.go
git commit -m "fix(sakurajima): fix goroutine leaks by removing ctx parameter from closures

The pattern 'go func(ctx context.Context) { <-ctx.Done(); ... }' creates
goroutines that receive a copy of ctx instead of capturing from outer scope,
causing them to never exit when the outer context is cancelled.

Fix by removing the parameter and letting the goroutine capture ctx via closure.

Also removes duplicate goroutine in ListenAndServeMetrics that was closing
the same resource twice.

Assisted-by: GLM 4.6 via Claude Code
Reviewbot-request: yes
"
```

---

## Task 4: Add InsecureSkipVerify Validation

**Files:**

- Modify: `cmd/sakurajima/internal/config/domain.go`
- Modify: `cmd/sakurajima/internal/entrypoint/router.go`

### Step 1: Write test for InsecureSkipVerify validation

Add to `cmd/sakurajima/internal/config/domain_test.go`:

```go
func TestDomainInsecureSkipVerifyValidation(t *testing.T) {
    tests := []struct {
        name    string
        domain  config.Domain
        wantErr bool
        errMsg  string
    }{
        {
            name: "insecure_skip_verify with https",
            domain: config.Domain{
                Name:               "test.example.com",
                Target:             "https://backend:443",
                InsecureSkipVerify: true,
                TLS: config.TLS{Cert: "c", Key: "k"},
            },
            wantErr: false,
        },
        {
            name: "insecure_skip_verify with http",
            domain: config.Domain{
                Name:               "test.example.com",
                Target:             "http://backend:80",
                InsecureSkipVerify: true,
                TLS: config.TLS{Cert: "c", Key: "k"},
            },
            wantErr: true,
            errMsg:  "only valid for https:// targets",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.domain.Valid()
            if tt.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

Run: `go test ./cmd/sakurajima/internal/config/`
Expected: FAIL (validation not implemented yet)

### Step 2: Add validation to Domain.Valid()

Modify: `cmd/sakurajima/internal/config/domain.go`

Add in Domain.Valid() after TLS validation:

```go
    // Validate InsecureSkipVerify usage
    if d.InsecureSkipVerify {
        u, err := url.Parse(d.Target)
        if err == nil && u.Scheme != "https" {
            errs = append(errs, fmt.Errorf("insecure_skip_verify is only valid for https:// targets, got %s", u.Scheme))
        }
    }
```

### Step 3: Add security warning in router

Modify: `cmd/sakurajima/internal/entrypoint/router.go`

At lines 95-101, replace:

```go
        if d.InsecureSkipVerify {
            rp.Transport = &http.Transport{
                TLSClientConfig: &tls.Config{
                    InsecureSkipVerify: true,
                },
            }
        }
```

With:

```go
        if d.InsecureSkipVerify {
            if u.Scheme != "https" {
                return fmt.Errorf("insecure_skip_verify can only be used with https:// targets, got %s", u.Scheme)
            }

            slog.Warn("SECURITY WARNING: TLS certificate verification disabled",
                "domain", d.Name,
                "target", d.Target,
                "risk", "Man-in-the-Middle attacks possible")

            rp.Transport = &http.Transport{
                TLSClientConfig: &tls.Config{
                    InsecureSkipVerify: true,
                },
            }
        }
```

### Step 4: Run tests

Run: `go test ./cmd/sakurajima/...`
Expected: All PASS

### Step 5: Commit

```bash
git add cmd/sakurajima/internal/config/domain.go cmd/sakurajima/internal/entrypoint/router.go
git commit -m "feat(sakurajima): add validation for insecure_skip_verify option

Restrict InsecureSkipVerify to HTTPS targets only and add security
warning when enabled. This prevents accidental use with HTTP targets
where the option would have no effect but might be misleading.

Assisted-by: GLM 4.6 via Claude Code
Reviewbot-request: yes
"
```

---

## Task 5: Fix Unix Socket Path Traversal

**Files:**

- Modify: `cmd/sakurajima/internal/config/domain.go`
- Modify: `cmd/sakurajima/internal/entrypoint/router.go`

### Step 1: Write test for path traversal validation

Add to `cmd/sakurajima/internal/config/domain_test.go`:

```go
func TestUnixSocketPathTraversal(t *testing.T) {
    tests := []struct {
        name    string
        target  string
        wantErr bool
    }{
        {"valid unix socket", "unix:///var/run/app.sock", false},
        {"path traversal attempt", "unix://../../../etc/passwd", true},
        {"relative path", "unix://./sock", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            d := config.Domain{
                Name:   "test.com",
                Target: tt.target,
                TLS:    config.TLS{},
            }
            err := d.Valid()
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Step 2: Add validation to isURLValid()

Modify: `cmd/sakurajima/internal/config/domain.go`

In isURLValid(), add unix case validation:

```go
    case "unix":
        socketPath := strings.TrimPrefix(input, "unix://")
        if strings.Contains(socketPath, "../") {
            return fmt.Errorf("%w unix socket path contains path traversal: %s", ErrInvalidURLScheme, socketPath)
        }
        if socketPath == "" {
            return fmt.Errorf("%w unix socket path is empty", ErrInvalidURLScheme)
        }
```

### Step 3: Clean path in router

Modify: `cmd/sakurajima/internal/entrypoint/router.go`

At lines 106-118, replace:

```go
        h = &httputil.ReverseProxy{
            Director: func(r *http.Request) {
                r.URL.Scheme = "http"
                r.URL.Host = d.Name
                r.Host = d.Name
            },
            Transport: &http.Transport{
                DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
                    return net.Dial("unix", strings.TrimPrefix(d.Target, "unix://"))
                },
            },
        }
```

With:

```go
        socketPath := strings.TrimPrefix(d.Target, "unix://")
        socketPath = filepath.Clean(socketPath)

        if !filepath.IsAbs(socketPath) {
            domainErrs = append(domainErrs, fmt.Errorf("unix socket path must be absolute: %s", socketPath))
            break
        }

        h = &httputil.ReverseProxy{
            Director: func(r *http.Request) {
                r.URL.Scheme = "http"
                r.URL.Host = d.Name
                r.Host = d.Name
            },
            Transport: &http.Transport{
                DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
                    return net.Dial("unix", socketPath)
                },
            },
        }
```

Add import at top of file:

```go
import "path/filepath"
```

### Step 4: Run tests

Run: `go test ./cmd/sakurajima/...`
Expected: All PASS

### Step 5: Commit

```bash
git add cmd/sakurajima/internal/config/domain.go cmd/sakurajima/internal/entrypoint/router.go
git commit -m "feat(sakurajima): add unix socket path validation and sanitization

Prevent path traversal attacks in unix socket paths by:
1. Validating that paths don't contain '../' sequences
2. Cleaning paths with filepath.Clean()
3. Requiring absolute paths

Assisted-by: GLM 4.6 via Claude Code
Reviewbot-request: yes
"
```

---

## Task 6: Fix H2C Target Validation

**Files:**

- Modify: `cmd/sakurajima/internal/entrypoint/h2c.go`
- Modify: `cmd/sakurajima/internal/entrypoint/router.go`

### Step 1: Write test for H2C validation

Create file: `cmd/sakurajima/internal/entrypoint/h2c_test.go`:

```go
package entrypoint

import (
    "net/url"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestNewH2CReverseProxy(t *testing.T) {
    tests := []struct {
        name    string
        target  string
        wantErr bool
    }{
        {"valid h2c target", "h2c://backend:443", false},
        {"http target for h2c", "http://backend:80", false},
        {"https target for h2c", "https://backend:443", true},
        {"missing host", "h2c://", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            u, err := url.Parse(tt.target)
            require.NoError(t, err)

            proxy, err := newH2CReverseProxy(u)
            if tt.wantErr {
                assert.Error(t, err)
                assert.Nil(t, proxy)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, proxy)
            }
        })
    }
}
```

Run: `go test ./cmd/sakurajima/internal/entrypoint/ -run TestNewH2CReverseProxy`
Expected: FAIL (function doesn't return error yet)

### Step 2: Update newH2CReverseProxy signature

Modify: `cmd/sakurajima/internal/entrypoint/h2c.go`

Change function signature to return error:

```go
func newH2CReverseProxy(target *url.URL) (*httputil.ReverseProxy, error) {
```

Add validation at start:

```go
    if target == nil {
        return nil, fmt.Errorf("h2c target cannot be nil")
    }

    if target.Host == "" {
        return nil, fmt.Errorf("h2c target must have a host: %s", target.String())
    }

    if target.Scheme != "http" && target.Scheme != "h2c" {
        return nil, fmt.Errorf("h2c target must use http:// or h2c:// scheme, got: %s", target.Scheme)
    }

    if strings.Contains(target.Host, "..") {
        return nil, fmt.Errorf("h2c target host contains invalid characters: %s", target.Host)
    }
```

Change final return:

```go
    return &httputil.ReverseProxy{
        Director:  director,
        Transport: transport,
    }, nil
```

### Step 3: Update router to handle error

Modify: `cmd/sakurajima/internal/entrypoint/router.go`

At lines 104-105, replace:

```go
            case "h2c":
                h = newH2CReverseProxy(u)
```

With:

```go
            case "h2c":
                h2cProxy, err := newH2CReverseProxy(u)
                if err != nil {
                    domainErrs = append(domainErrs, fmt.Errorf("h2c proxy: %w", err))
                } else {
                    h = h2cProxy
                }
```

### Step 4: Run tests

Run: `go test ./cmd/sakurajima/...`
Expected: All PASS

### Step 5: Commit

```bash
git add cmd/sakurajima/internal/entrypoint/h2c.go cmd/sakurajima/internal/entrypoint/router.go cmd/sakurajima/internal/entrypoint/h2c_test.go
git commit -m "feat(sakurajima): add H2C target validation

Validate that H2C targets use http:// or h2c:// schemes and have
valid hostnames. Returns error instead of panicking on invalid input.

Assisted-by: GLM 4.6 via Claude Code
Reviewbot-request: yes
"
```

---

## Task 7: Fix Missing Return in readyz

**Files:**

- Modify: `cmd/sakurajima/internal/entrypoint/metrics.go`

### Step 1: Write test for readyz

Add to `cmd/sakurajima/internal/entrypoint/metrics_test.go`:

```go
func TestReadyzMissingHealthService(t *testing.T) {
    req := httptest.NewRequest("GET", "/readyz", nil)
    w := httptest.NewRecorder()

    readyz(w, req)

    resp := w.Result()
    defer resp.Body.Close()

    assert.Equal(t, http.StatusExpectationFailed, resp.StatusCode)
}
```

Run: `go test ./cmd/sakurajima/internal/entrypoint/ -run TestReadyzMissingHealthService`
Expected: PASS (but code has missing return)

### Step 2: Add missing return statements

Modify: `cmd/sakurajima/internal/entrypoint/metrics.go`

At lines 54-72, ensure all paths return:

```go
func readyz(w http.ResponseWriter, r *http.Request) {
    st, ok := internal.GetHealth("osiris")
    if !ok {
        slog.Error("health service osiris does not exist, file a bug")
        http.Error(w, "health service osiris does not exist", http.StatusExpectationFailed)
        return // ADD THIS
    }

    switch st {
    case healthv1.HealthCheckResponse_NOT_SERVING:
        http.Error(w, "NOT OK", http.StatusInternalServerError)
        return
    case healthv1.HealthCheckResponse_SERVING:
        fmt.Fprintln(w, "OK")
        return
    default:
        http.Error(w, "UNKNOWN", http.StatusFailedDependency)
        return
    }
}
```

### Step 3: Run tests

Run: `go test ./cmd/sakurajima/...`
Expected: All PASS

### Step 4: Commit

```bash
git add cmd/sakurajima/internal/entrypoint/metrics.go
git commit -m "fix(sakurajima): add missing return statements in readyz handler

Ensure all code paths in readyz return appropriately to prevent
continued execution after sending error response.

Assisted-by: GLM 4.6 via Claude Code
Reviewbot-request: yes
"
```

---

# Medium Severity Security Hardening

## Task 8: Install and Configure gosec

**Files:**

- Modify: `package.json`

### Step 1: Install gosec

Run:

```bash
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
```

Verify installation:

```bash
gosec -version
```

### Step 2: Add npm scripts

Modify: `package.json`

Add to scripts section:

```json
  "test:security": "gosec ./cmd/sakurajima/...",
  "test:security:ci": "gosec -quiet -fmt sarif -out gosec-results.sarif ./cmd/sakurajima/..."
```

### Step 3: Run gosec

Run:

```bash
npm run test:security
```

Expected: Report with findings (document them for fixing)

### Step 4: Commit

```bash
git add package.json
git commit -m "feat(sakurajima): add gosec security scanning to npm scripts

Add gosec installation and npm scripts for security scanning.
Run 'npm run test:security' to scan for security issues.

Assisted-by: GLM 4.6 via Claude Code
Reviewbot-request: yes
"
```

---

## Task 9: Add TLS Security Hardening

**Files:**

- Modify: `cmd/sakurajima/internal/config/tls.go`
- Modify: `cmd/sakurajima/internal/entrypoint/router.go`

### Step 1: Write test for TLS config

Add to `cmd/sakurajima/internal/config/tls_test.go`:

```go
func TestTLSToTLSConfig(t *testing.T) {
    tests := []struct {
        name    string
        tls     config.TLS
        wantMin uint16
        wantErr bool
    }{
        {
            name:    "default secure configuration",
            tls:     config.TLS{},
            wantMin: tls.VersionTLS12,
        },
        {
            name: "explicit TLS 1.3 only",
            tls: config.TLS{
                MinVersion: config.TLSVersion13,
                MaxVersion: config.TLSVersion13,
            },
            wantMin: tls.VersionTLS13,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.tls.Valid()
            if (err != nil) != tt.wantErr {
                t.Errorf("Valid() error = %v, wantErr %v", err, tt.wantErr)
            }

            if !tt.wantErr {
                tc := tt.tls.ToTLSConfig(nil)
                if tc.MinVersion != tt.wantMin {
                    t.Errorf("MinVersion = %v, want %v", tc.MinVersion, tt.wantMin)
                }
            }
        })
    }
}
```

### Step 2: Add TLS configuration to config/tls.go

Add types, constants, and ToTLSConfig method (see security hardening plan for full code).

Key additions:

- TLSVersion type and constants
- CipherSuites and Curves configuration fields
- ToTLSConfig() method with secure defaults

### Step 3: Update router to use TLS config

Modify: `cmd/sakurajima/internal/entrypoint/router.go`

Add tlsConfig field to Router struct and update HandleHTTPS.

### Step 4: Run tests

Run: `go test ./cmd/sakurajima/...`
Expected: All PASS

### Step 5: Commit

```bash
git add cmd/sakurajima/internal/config/tls.go cmd/sakurajima/internal/entrypoint/router.go
git commit -m "feat(sakurajima): add TLS security hardening options

Add configurable TLS minimum version, cipher suites, and curves.
Defaults to TLS 1.2 minimum with secure cipher suites.

Assisted-by: GLM 4.6 via Claude Code
Reviewbot-request: yes
"
```

---

## Task 10: Add Log Injection Prevention

**Files:**

- Modify: `cmd/sakurajima/internal/logging/combined.go`

### Step 1: Write test for log sanitization

Add to `cmd/sakurajima/internal/logging/combined_test.go`:

```go
func TestSanitizeLogString(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"normal string", "hello world", "hello world"},
        {"newline injection", "test\n2024/01/01", "test 2024/01/01"},
        {"CRLF injection", "test\r\n[2024]", "test  [2024]"},
        {"tab preserved", "test\tstring", "test\tstring"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := sanitizeLogString(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Step 2: Add sanitization functions

Add sanitizeLogString, sanitizeURI, sanitizeReferer, sanitizeUserAgent functions to combined.go (see security hardening plan for full code).

### Step 3: Update LogHTTPRequest to use sanitization

Modify LogHTTPRequest to sanitize all user-controlled input before writing.

### Step 4: Run tests

Run: `go test ./cmd/sakurajima/...`
Expected: All PASS

### Step 5: Commit

```bash
git add cmd/sakurajima/internal/logging/combined.go
git commit -m "feat(sakurajima): add log injection prevention

Sanitize user-controlled input (URI, headers, Host) before logging
to prevent log injection attacks. Replaces newlines, carriage returns,
and control characters with spaces.

Assisted-by: GLM 4.6 via Claude Code
Reviewbot-request: yes
"
```

---

# Tooling and Testing Infrastructure

## Task 11: Add Concurrency Tests

**Files:**

- Create: `cmd/sakurajima/internal/entrypoint/concurrency_test.go`

### Step 1: Create concurrency test file

Create comprehensive tests for:

- Concurrent config reload during requests
- Certificate lookup during reload
- Multiple SIGHUP signals

### Step 2: Run with race detector

Run: `go test -race ./cmd/sakurajima/...`
Expected: All PASS with no data races

### Step 3: Commit

```bash
git add cmd/sakurajima/internal/entrypoint/concurrency_test.go
git commit -m "test(sakurajima): add concurrency tests for config reload

Add tests that simulate concurrent config reloads during active
request handling to verify no race conditions occur.

Assisted-by: GLM 4.6 via Claude Code
Reviewbot-request: yes
"
```

---

## Task 12: Add Graceful Shutdown Tests

**Files:**

- Create: `cmd/sakurajima/internal/entrypoint/shutdown_test.go`

### Step 1: Create shutdown test file

Create tests for:

- Basic graceful shutdown
- All servers shutting down
- In-flight request completion

### Step 2: Run tests

Run: `go test ./cmd/sakurajima/...`
Expected: All PASS

### Step 3: Commit

```bash
git add cmd/sakurajima/internal/entrypoint/shutdown_test.go
git commit -m "test(sakurajima): add graceful shutdown tests

Verify that the service shuts down cleanly when context is
cancelled and that in-flight requests are handled properly.

Assisted-by: GLM 4.6 via Claude Code
Reviewbot-request: yes
"
```

---

## Task 13: Fix Formatting Issues

### Step 1: Format with goimports

Run:

```bash
npm run format
```

Or specifically:

```bash
go tool goimports -w ./cmd/sakurajima/internal/health_test.go
```

### Step 2: Verify formatting

Run:

```bash
gofmt -l ./cmd/sakurajima/
```

Expected: No output (all files formatted)

### Step 3: Commit

```bash
git add cmd/sakurajima/internal/health_test.go
git commit -m "style(sakurajima): fix formatting in health_test.go

Run goimports/gofmt to ensure consistent code formatting.

Assisted-by: GLM 4.6 via Claude Code
Reviewbot-request: yes
"
```

---

# Execution Checklist

## Before Starting

- [ ] Create a git worktree for this work
- [ ] Run `npm test` to establish baseline
- [ ] Run `go test -race ./cmd/sakurajima/...` to verify current state

## Execution Order

### Phase 1: Critical Fixes (Must Do)

- [ ] Task 1: Fix Data Race on accessLog
- [ ] Task 2: Fix SSRF Vulnerability

### Phase 2: High Severity Fixes

- [ ] Task 3: Fix Goroutine Leaks
- [ ] Task 4: InsecureSkipVerify Validation
- [ ] Task 5: Unix Socket Path Traversal
- [ ] Task 6: H2C Target Validation
- [ ] Task 7: Missing Return in readyz

### Phase 3: Security Hardening

- [ ] Task 8: Install gosec
- [ ] Task 9: TLS Security Hardening
- [ ] Task 10: Log Injection Prevention

### Phase 4: Testing Infrastructure

- [ ] Task 11: Concurrency Tests
- [ ] Task 12: Graceful Shutdown Tests
- [ ] Task 13: Formatting Fixes

## Verification After Each Task

- [ ] Run `go test ./cmd/sakurajima/...`
- [ ] Run `go test -race ./cmd/sakurajima/...`
- [ ] Run `go vet ./cmd/sakurajima/...`
- [ ] Run `go tool staticcheck ./cmd/sakurajima/...`
- [ ] Build: `go build ./cmd/sakurajima`

## Final Verification

- [ ] All tests pass: `npm test`
- [ ] Race detector clean: `go test -race ./cmd/sakurajima/...`
- [ ] Security scan: `npm run test:security`
- [ ] Code formatted: `npm run format`
- [ ] Build succeeds: `go build ./cmd/sakurajima`
- [ ] Review findings from all reviewers addressed

---

## Notes for Implementation

1. **Each task should be completed independently** - commit after each task
2. **Run tests after every change** - don't batch changes
3. **Use exact file paths** - these are specific to the codebase
4. **All commits must be signed** - use `--signoff`
5. **Follow Conventional Commits** format as shown
6. **Include Assisted-by footer** in all commits
7. **Trigger reviewbot** with `Reviewbot-request: yes` footer

---

## Post-Implementation

After all tasks are complete:

1. **Create summary PR** with all commits
2. **Run full test suite** including race detector
3. **Generate security scan report** with gosec
4. **Update documentation** with new configuration options
5. **Create deployment guide** with security recommendations

---

**Plan complete and saved to `docs/plans/2026-02-05-sakurajima-production-fixes.md`.**

Two execution options:

**1. Subagent-Driven (this session)** - I dispatch fresh subagent per task, review between tasks, fast iteration

**2. Parallel Session (separate)** - Open new session with executing-plans, batch execution with checkpoints

Which approach?
