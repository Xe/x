# Tooling and Testing Infrastructure Plan for cmd/sakurajima

## Overview

This document outlines the implementation of security scanning, concurrent operation testing, graceful shutdown verification, and formatting fixes for the sakurajima application.

## 1. Security Scanning with gosec

### 1.1 Installation

Install gosec security scanner:

```bash
# Install gosec locally
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest

# Verify installation
gosec version
```

### 1.2 Running gosec

```bash
# Scan the sakurajima package
gosec ./cmd/sakurajima/...

# Scan with specific severity level
gosec -severity=medium ./cmd/sakurajima/...

# Generate reports in multiple formats
gosec -fmt json -out gosec-report.json ./cmd/sakurajima/...
gosec -fmt sarif -out gosec-report.sarif ./cmd/sakurajima/...

# Run with confidence score filtering
gosec -confidence=0.5 ./cmd/sakurajima/...
```

### 1.3 Integration with npm test workflow

Update `/Users/cadey/Code/Xe/x/package.json` to include security scanning:

```json
{
  "scripts": {
    "test": "npm run generate && go test ./...",
    "test:security": "gosec -no-fail -fmt json -out gosec-report.json ./cmd/sakurajima/...",
    "test:security:ci": "gosec -severity=medium ./cmd/sakurajima/...",
    "format": "go tool goimports -w . && npx prettier -w . 2>&1 >/dev/null && buf format -w",
    "generate": "npm run generate:buf && npm run generate:go && npm run format",
    "generate:buf": "buf generate && npm run generate:buf:falin",
    "generate:buf:falin": "cd migroserbices/falin && npm ci && npm run generate",
    "generate:go": "go generate ./...",
    "prepare": "husky"
  }
}
```

### 1.4 CI/CD Integration

Add security scanning step to `.github/workflows/go.yml`:

```yaml
- name: Run gosec security scanner
  run: |
    go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
    gosec -severity=medium -fmt sarif -out gosec-report.sarif ./cmd/sakurajima/...

- name: Upload gosec results
  if: always()
  uses: github/codeql-action/upload-sarif@main
  with:
    sarif_file: gosec-report.sarif
```

### 1.5 Common Security Issues to Address

Based on the codebase analysis, gosec will likely flag:

1. **G104 (Errors not checked)**: Error handling in goroutines
   - Location: `/Users/cadey/Code/Xe/x/cmd/sakurajima/internal/entrypoint/router.go:49-51`
   - Current: `go func(ctx context.Context) { <-ctx.Done(); ln.Close() }(ctx)`
   - Fix: Log the close error or use a deferred error handler

2. **G307 (Deferring file close method)**: Resource cleanup
   - Location: Multiple locations in router.go
   - Current: `defer ln.Close()`
   - Fix: Add error checking or use a wrapper

3. **G101 (Look for hard coded credentials)**: S3 bucket configuration
   - Location: config.go
   - Current: Autocert S3 bucket configuration
   - Fix: Ensure credentials are from environment, not hardcoded

## 2. Concurrent Operations Testing (Config Reload During Requests)

### 2.1 Test Design

Create file: `/Users/cadey/Code/Xe/x/cmd/sakurajima/internal/entrypoint/concurrency_test.go`

```go
package entrypoint

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"within.website/x/cmd/sakurajima/internal/config"
)

// TestConcurrentConfigReload verifies that config reloads don't interfere with active requests
func TestConcurrentConfigReload(t *testing.T) {
	// Create initial configuration
	initialCfg := createTestConfig(t, "example.com", "http://localhost:8080")

	rtr, err := NewRouter(initialCfg, "INFO")
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	// Track concurrent requests
	var requestCount atomic.Int64
	var reloadCount atomic.Int64
	var errors atomic.Int64

	// Start HTTP server for testing
	server := httptest.NewServer(rtr)
	defer server.Close()

	// Simulate concurrent requests and reloads
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})

	// Goroutine to continuously reload config
	go func() {
		for {
			select {
			case <-ctx.Done():
				close(done)
				return
			default:
				time.Sleep(100 * time.Millisecond)

				// Trigger SIGHUP to reload config
				if err := rtr.loadConfig(); err != nil {
					t.Logf("Config reload error: %v", err)
				}
				reloadCount.Add(1)
			}
		}
	}()

	// Goroutine to make continuous requests
	go func() {
		client := &http.Client{Timeout: 1 * time.Second}
		for {
			select {
			case <-ctx.Done():
				return
			default:
				req, err := http.NewRequest("GET", server.URL, nil)
				if err != nil {
					errors.Add(1)
					continue
				}
				req.Host = "example.com"

				resp, err := client.Do(req)
				if err != nil {
					// Some errors are expected during reload
					errors.Add(1)
				} else {
					resp.Body.Close()
				}
				requestCount.Add(1)

				time.Sleep(50 * time.Millisecond)
			}
		}
	}()

	<-done

	totalRequests := requestCount.Load()
	totalReloads := reloadCount.Load()
	totalErrors := errors.Load()

	t.Logf("Completed %d requests during %d config reloads with %d errors",
		totalRequests, totalReloads, totalErrors)

	// Verify that errors are within acceptable threshold
	errorRate := float64(totalErrors) / float64(totalRequests)
	if errorRate > 0.1 { // Allow up to 10% error rate
		t.Errorf("Error rate too high: %.2f%% (%d/%d)", errorRate*100, totalErrors, totalRequests)
	}
}

// TestConcurrentConfigReloadWithSignal tests config reload via SIGHUP signal
func TestConcurrentConfigReloadWithSignal(t *testing.T) {
	cfg := createTestConfig(t, "example.com", "http://localhost:8080")

	rtr, err := NewRouter(cfg, "INFO")
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background config reload goroutine
	go rtr.backgroundReloadConfig(ctx)

	// Create test server
	server := httptest.NewServer(rtr)
	defer server.Close()

	// Send SIGHUP to trigger reload
	time.Sleep(100 * time.Millisecond)
	selfProcess, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("Failed to find process: %v", err)
	}

	// Make request during reload
	client := &http.Client{Timeout: 2 * time.Second}
	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Host = "example.com"

	done := make(chan bool)
	go func() {
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			done <- true
		} else {
			done <- false
		}
	}()

	// Trigger reload
	selfProcess.Signal(syscall.SIGHUP)

	// Wait for request to complete
	success := <-done
	if !success {
		t.Error("Request failed during config reload")
	}
}

// TestRouterConcurrentAccess tests concurrent read/write access to router
func TestRouterConcurrentAccess(t *testing.T) {
	cfg := createTestConfig(t, "example.com", "http://localhost:8080")

	rtr, err := NewRouter(cfg, "INFO")
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan struct{})

	// Concurrent readers (simulating requests)
	for i := 0; i < 10; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					rtr.lock.RLock()
					_ = len(rtr.routes)
					_ = len(rtr.tlsCerts)
					rtr.lock.RUnlock()
				}
			}
		}()
	}

	// Concurrent writers (simulating config reloads)
	for i := 0; i < 3; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				default:
					newCfg := createTestConfig(t, "example.com", "http://localhost:8080")
					_ = rtr.setConfig(newCfg)
					time.Sleep(100 * time.Millisecond)
				}
			}
		}()
	}

	<-ctx.Done()
	close(done)
}

// Helper function to create test configuration
func createTestConfig(t *testing.T, domain, target string) config.Toplevel {
	t.Helper()
	return config.Toplevel{
		Bind: config.Bind{
			HTTP:    ":8080",
			HTTPS:   ":8443",
			Metrics: ":9090",
		},
		Domains: []config.Domain{
			{
				Name: domain,
				Target: target,
				TLS: config.TLS{
					Cert:       "/path/to/cert.pem",
					Key:        "/path/to/key.pem",
					Autocert:   false,
				},
			},
		},
		Logging: config.Logging{
			AccessLog:  "/tmp/access.log",
			MaxSizeMB:  100,
			MaxAgeDays: 7,
			MaxBackups: 3,
			Compress:   true,
			Filters:    []config.Filter{},
		},
	}
}
```

### 2.2 Test Cases for Race Conditions

Key race condition scenarios to test:

1. **Config reload during active request**
   - Start HTTP request
   - Trigger config reload mid-request
   - Verify request completes successfully

2. **Certificate lookup during config reload**
   - Initiate TLS handshake
   - Reload config simultaneously
   - Verify no panic or deadlock

3. **Metrics update during config reload**
   - Make request that updates metrics
   - Reload config
   - Verify metrics are not corrupted

4. **Concurrent SIGHUP signals**
   - Send multiple SIGHUP signals rapidly
   - Verify system remains stable

## 3. Graceful Shutdown Testing

### 3.1 Test Design

Create file: `/Users/cadey/Code/Xe/x/cmd/sakurajima/internal/entrypoint/shutdown_test.go`

```go
package entrypoint

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"
)

// TestGracefulShutdown verifies graceful shutdown behavior
func TestGracefulShutdown(t *testing.T) {
	cfg := createTestConfig(t, "example.com", "http://localhost:8080")

	rtr, err := NewRouter(cfg, "INFO")
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Create test server
	server := httptest.NewServer(rtr)
	defer server.Close()

	// Track in-flight requests
	inFlight := make(chan struct{})
	requestComplete := make(chan struct{})

	// Create a slow handler to simulate in-flight requests
	slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inFlight <- struct{}{}
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		<-requestComplete
	})

	slowServer := &http.Server{
		Addr:    ":0",
		Handler: slowHandler,
	}

	// Start slow server
	go func() {
		slowServer.ListenAndServe()
	}()

	// Start a request
	go func() {
		client := &http.Client{Timeout: 2 * time.Second}
		req, _ := http.NewRequest("GET", server.URL, nil)
		req.Host = "example.com"
		client.Do(req)
	}()

	// Wait for request to be in-flight
	<-inFlight

	// Simulate graceful shutdown
	startTime := time.Now()
	cancel()

	// Signal completion
	close(requestComplete)

	// Verify shutdown completes
	select {
	case <-time.After(3 * time.Second):
		t.Error("Shutdown did not complete in time")
	case <-time.After(500 * time.Millisecond):
		// Expected shutdown time
	}

	shutdownDuration := time.Since(startTime)
	t.Logf("Shutdown completed in %v", shutdownDuration)

	// Verify server stopped
	slowServer.Close()
}

// TestGracefulShutdownWithMultipleServers tests shutdown of all servers
func TestGracefulShutdownWithMultipleServers(t *testing.T) {
	cfg := createTestConfig(t, "example.com", "http://localhost:8080")

	rtr, err := NewRouter(cfg, "INFO")
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create test servers
	httpServer := httptest.NewServer(rtr)
	defer httpServer.Close()

	// Track which servers stopped
	serversStopped := make([]bool, 3)

	// Simulate main function behavior
	g, gCtx := context.WithCancel(ctx)

	// HTTP server goroutine
	go func() {
		<-gCtx.Done()
		serversStopped[0] = true
	}()

	// HTTPS server goroutine
	go func() {
		<-gCtx.Done()
		serversStopped[1] = true
	}()

	// Metrics server goroutine
	go func() {
		<-gCtx.Done()
		serversStopped[2] = true
	}()

	// Make a request
	client := &http.Client{Timeout: 1 * time.Second}
	req, _ := http.NewRequest("GET", httpServer.URL, nil)
	req.Host = "example.com"
	resp, err := client.Do(req)
	if err != nil {
		t.Logf("Request error (may be expected): %v", err)
	} else {
		resp.Body.Close()
	}

	// Cancel context to trigger shutdown
	cancel()

	// Verify all servers stopped
	time.Sleep(100 * time.Millisecond)
	for i, stopped := range serversStopped {
		if !stopped {
			t.Errorf("Server %d did not stop gracefully", i)
		}
	}
}

// TestGracefulShutdownWithInterruptSignal tests SIGINT handling
func TestGracefulShutdownWithInterruptSignal(t *testing.T) {
	cfg := createTestConfig(t, "example.com", "http://localhost:8080")

	rtr, err := NewRouter(cfg, "INFO")
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create test server
	server := httptest.NewServer(rtr)
	defer server.Close()

	// Send interrupt signal
	go func() {
		time.Sleep(100 * time.Millisecond)
		selfProcess, err := os.FindProcess(os.Getpid())
		if err != nil {
			t.Errorf("Failed to find process: %v", err)
			return
		}
		selfProcess.Signal(syscall.SIGINT)
	}()

	// Verify context is cancelled
	select {
	case <-ctx.Done():
		t.Log("Context cancelled successfully on SIGINT")
	case <-time.After(2 * time.Second):
		t.Error("Context did not cancel on SIGINT")
	}

	// Verify server is still responsive for graceful shutdown
	client := &http.Client{Timeout: 500 * time.Millisecond}
	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Host = "example.com"

	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("Server not responsive during graceful shutdown: %v", err)
	} else {
		resp.Body.Close()
	}
}

// TestGracefulShutdownInFlightRequests verifies in-flight requests complete
func TestGracefulShutdownInFlightRequests(t *testing.T) {
	cfg := createTestConfig(t, "example.com", "http://localhost:8080")

	rtr, err := NewRouter(cfg, "INFO")
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	server := httptest.NewServer(rtr)
	defer server.Close()

	// Track request completion
	completedRequests := make(chan int, 10)

	// Start multiple requests
	for i := 0; i < 5; i++ {
		go func(reqNum int) {
			client := &http.Client{Timeout: 2 * time.Second}
			req, _ := http.NewRequest("GET", server.URL, nil)
			req.Host = "example.com"

			resp, err := client.Do(req)
			if err == nil {
				resp.Body.Close()
				completedRequests <- reqNum
			} else {
				completedRequests <- -1
			}
		}(i)
	}

	// Wait a bit then trigger shutdown
	time.Sleep(200 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Trigger graceful shutdown
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	// Count completed requests
	completed := 0
	timeout := time.After(3 * time.Second)

	for {
		select {
		case reqNum := <-completedRequests:
			if reqNum != -1 {
				completed++
				t.Logf("Request %d completed", reqNum)
			}
		case <-timeout:
			t.Logf("Shutdown: %d/5 requests completed", completed)
			if completed < 3 {
				t.Errorf("Too few requests completed during graceful shutdown: %d/5", completed)
			}
			return
		}
	}
}
```

### 3.2 Test Cases for Graceful Shutdown

Key scenarios to test:

1. **In-flight requests complete**
   - Start long-running request
   - Trigger shutdown
   - Verify request completes

2. **All servers stop gracefully**
   - HTTP, HTTPS, and Metrics servers
   - Verify all listeners close

3. **Signal handling**
   - SIGINT and SIGTERM
   - Verify context cancellation

4. **Health status during shutdown**
   - Verify health endpoint reflects shutting down state
   - Check `SetHealth` is called appropriately

## 4. Formatting Issues Fix

### 4.1 Current Formatting Issue

The file `/Users/cadey/Code/Xe/x/cmd/sakurajima/internal/health_test.go` may have formatting issues.

### 4.2 Fix Commands

```bash
# Run goimports to fix imports and formatting
go tool goimports -w /Users/cadey/Code/Xe/x/cmd/sakurajima/internal/health_test.go

# Run gofmt as backup
go fmt /Users/cadey/Code/Xe/x/cmd/sakurajima/internal/health_test.go

# Run prettier for any embedded markdown or JSON
npx prettier -w /Users/cadey/Code/Xe/x/cmd/sakurajima/internal/health_test.go

# Verify formatting
goimports -l /Users/cadey/Code/Xe/x/cmd/sakurajima/internal/health_test.go
```

### 4.3 Fix All Formatting Issues

```bash
# Fix formatting for entire sakurajima package
go tool goimports -w ./cmd/sakurajima/...
go fmt ./cmd/sakurajima/...

# Or use the existing npm script
npm run format
```

## 5. Implementation Steps

### Phase 1: Security Scanning (Week 1)

1. Install gosec
2. Run initial security scan
3. Address critical security findings
4. Add gosec to CI/CD pipeline
5. Document security scanning procedures

### Phase 2: Concurrent Testing (Week 2)

1. Create concurrency_test.go
2. Implement TestConcurrentConfigReload
3. Implement TestRouterConcurrentAccess
4. Add race detector to CI: `go test -race ./...`
5. Fix any race conditions discovered

### Phase 3: Graceful Shutdown Testing (Week 2-3)

1. Create shutdown_test.go
2. Implement TestGracefulShutdown
3. Implement TestGracefulShutdownWithMultipleServers
4. Implement TestGracefulShutdownWithInterruptSignal
5. Verify graceful shutdown with production-like load

### Phase 4: Formatting and Documentation (Week 3)

1. Fix formatting issues in health_test.go
2. Run comprehensive formatting check
3. Update AGENTS.md with new testing procedures
4. Create this infrastructure plan document
5. Document all new test cases

## 6. Validation

### 6.1 Security Validation

```bash
# Run security scan
gosec -severity=medium ./cmd/sakurajima/...

# Verify no critical issues
# Output should show: "No issues found"
```

### 6.2 Concurrency Validation

```bash
# Run tests with race detector
go test -race -v ./cmd/sakurajima/internal/entrypoint/

# Should pass without race warnings
```

### 6.3 Graceful Shutdown Validation

```bash
# Run shutdown tests
go test -v -run TestGracefulShutdown ./cmd/sakurajima/internal/entrypoint/

# Should complete within timeout
```

### 6.4 Formatting Validation

```bash
# Check formatting
goimports -l ./cmd/sakurajima/...

# Should output nothing (all files formatted)
```

## 7. Success Criteria

- [x] gosec installed and integrated into CI/CD
- [ ] gosec runs without critical findings
- [ ] Concurrent operations tests pass with `-race` flag
- [ ] Graceful shutdown tests pass consistently
- [ ] All formatting issues resolved
- [ ] CI/CD pipeline includes all new tests
- [ ] Documentation updated with new procedures

## 8. Monitoring and Maintenance

### 8.1 Ongoing Security Monitoring

- Run gosec weekly: `npm run test:security`
- Review and address new findings
- Update gosec regularly: `go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest`

### 8.2 Test Maintenance

- Update tests as new features are added
- Maintain test coverage above 80%
- Review and update test cases quarterly

### 8.3 CI/CD Maintenance

- Monitor CI/CD pipeline performance
- Update test timeouts as needed
- Review and optimize test execution time

## 9. Troubleshooting

### 9.1 Common Issues

**gosec not found**

```bash
# Ensure GOPATH is set
echo $GOPATH
export PATH=$PATH:$(go env GOPATH)/bin
```

**Race detector false positives**

```bash
# Use -race=false for specific tests if needed
go test -race=false ./cmd/sakurajima/internal/entrypoint/
```

**Formatting fails**

```bash
# Check for conflicting formatters
which goimports gofmt prettier
# Ensure versions are compatible
```

## 10. References

- [gosec Documentation](https://github.com/securecodewarrior/gosec)
- [Go Race Detector](https://go.dev/doc/articles/race_detector)
- [Graceful Shutdown in Go](https://go.dev/doc/effective_go#closure)
- [Project AGENTS.md](/Users/cadey/Code/Xe/x/AGENTS.md)
