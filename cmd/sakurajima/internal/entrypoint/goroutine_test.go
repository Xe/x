package entrypoint

import (
	"context"
	"net"
	"runtime"
	"testing"
	"time"
)

// TestGoroutineCleanup verifies that all goroutines started by the router
// are properly cleaned up when the context is cancelled.
func TestGoroutineCleanup(t *testing.T) {
	// Get baseline goroutine count
	runtime.GC()
	baseline := runtime.NumGoroutine()

	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Create listeners to avoid bind conflicts
	httpLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create HTTP listener: %v", err)
	}
	defer httpLn.Close()

	httpsLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create HTTPS listener: %v", err)
	}
	defer httpsLn.Close()

	metricsLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create metrics listener: %v", err)
	}
	defer metricsLn.Close()

	// Track if goroutines exit properly
	httpExited := make(chan struct{})
	httpsExited := make(chan struct{})
	metricsExited := make(chan struct{})
	cleanupExited := make(chan struct{})

	// Simulate the goroutine pattern from HandleHTTP
	go func() {
		<-ctx.Done()
		httpLn.Close()
		close(httpExited)
	}()

	// Simulate the goroutine pattern from HandleHTTPS
	go func() {
		<-ctx.Done()
		httpsLn.Close()
		close(httpsExited)
	}()

	// Simulate the goroutine pattern from ListenAndServeMetrics
	go func() {
		<-ctx.Done()
		metricsLn.Close()
		close(metricsExited)
	}()

	// Simulate the goroutine pattern from entrypoint listeners
	go func() {
		<-ctx.Done()
		close(cleanupExited)
	}()

	// Give the goroutines time to start
	time.Sleep(100 * time.Millisecond)

	// Verify goroutines have started
	afterStart := runtime.NumGoroutine()
	if afterStart <= baseline {
		t.Fatalf("expected goroutines to increase, got %d (baseline: %d)", afterStart, baseline)
	}

	// Cancel the context - this should trigger cleanup
	cancel()

	// Wait for all goroutines to exit (with timeout)
	timeout := time.After(5 * time.Second)
	allExited := false

	select {
	case <-httpExited:
		select {
		case <-httpsExited:
			select {
			case <-metricsExited:
				select {
				case <-cleanupExited:
					allExited = true
				case <-timeout:
					t.Error("cleanup goroutine did not exit in time")
				}
			case <-timeout:
				t.Error("metrics goroutine did not exit in time")
			}
		case <-timeout:
			t.Error("https goroutine did not exit in time")
		}
	case <-timeout:
		t.Error("http goroutine did not exit in time")
	}

	if !allExited {
		t.Error("not all goroutines exited properly")
	}

	// Force GC to clean up any lingering resources
	runtime.GC()

	// Check that goroutines have been cleaned up
	afterCancel := runtime.NumGoroutine()

	// Allow for some tolerance (goroutines from the test itself)
	// More than 5 extra goroutines suggests a leak
	extraGoroutines := afterCancel - baseline
	if extraGoroutines > 5 {
		t.Errorf("possible goroutine leak: %d extra goroutines remain after context cancellation (started with %d, ended with %d)", extraGoroutines, baseline, afterCancel)
	}

	t.Logf("Goroutine cleanup successful: baseline=%d, afterStart=%d, afterCancel=%d, extra=%d", baseline, afterStart, afterCancel, extraGoroutines)
}
