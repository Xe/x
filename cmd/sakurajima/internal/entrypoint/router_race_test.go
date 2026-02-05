package entrypoint

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"within.website/x/cmd/sakurajima/internal/config"
)

// createTestConfig creates a minimal valid config for testing
func createTestConfig(accessLogPath string) config.Toplevel {
	return config.Toplevel{
		Bind: config.Bind{
			HTTP:    ":0",
			HTTPS:   ":0",
			Metrics: ":0",
		},
		Logging: config.Logging{
			AccessLog:  accessLogPath,
			MaxSizeMB:  100,
			MaxAgeDays: 7,
			MaxBackups: 3,
			Compress:   true,
			Filters:    []config.Filter{},
		},
		Domains: []config.Domain{
			{
				Name:   "test.local",
				Target: "http://localhost:9999",
				TLS: config.TLS{
					Cert: "./testdata/selfsigned.crt",
					Key:  "./testdata/selfsigned.key",
				},
			},
		},
	}
}

// TestRouterServeHTTPConcurrentReloads tests for data races between
// ServeHTTP (which reads accessLog) and setConfig (which closes/replaces accessLog)
func TestRouterServeHTTPConcurrentReloads(t *testing.T) {
	// Create a temporary file for access logging
	accessLogFile := fmt.Sprintf("%s/access.log", t.TempDir())

	cfg := createTestConfig(accessLogFile)
	rtr := newRouter(t, cfg)

	// Create a test backend server
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	}))
	defer backend.Close()

	// Update config to point to test backend
	cfg.Domains[0].Target = backend.URL
	if err := rtr.setConfig(cfg); err != nil {
		t.Fatalf("failed to set config: %v", err)
	}

	// Use sync.WaitGroup to coordinate concurrent operations
	var wg sync.WaitGroup

	// Start goroutines that continuously reload config
	const numReloaders = 10
	for i := 0; i < numReloaders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				// Create a new access log file for each reload to trigger Rotate/Close
				newAccessLog := fmt.Sprintf("%s/access-%d-%d.log", t.TempDir(), i, j)
				newCfg := createTestConfig(newAccessLog)
				newCfg.Domains[0].Target = backend.URL

				if err := rtr.setConfig(newCfg); err != nil {
					t.Errorf("config reload failed: %v", err)
				}
			}
		}()
	}

	// Start goroutines that make HTTP requests (triggers ServeHTTP -> accessLog read)
	const numRequesters = 100
	for i := 0; i < numRequesters; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				// Create a request that will trigger ServeHTTP
				req, err := http.NewRequest(http.MethodGet, "http://test.local/path", nil)
				if err != nil {
					t.Errorf("failed to create request: %v", err)
					continue
				}
				req.Host = "test.local"

				// Use ResponseRecorder to capture response
				w := httptest.NewRecorder()

				// This triggers the data race: ServeHTTP reads accessLog
				// while setConfig may be closing it
				rtr.ServeHTTP(w, req)
			}
		}()
	}

	// Wait for all goroutines to complete
	wg.Wait()
}
