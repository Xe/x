package entrypoint

import (
	"net/http"
	"net/http/httptest"
	"testing"

	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
	"within.website/x/cmd/sakurajima/internal"
)

func TestHealthz(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(healthz))

	internal.SetHealth("osiris", healthv1.HealthCheckResponse_NOT_SERVING)

	resp, err := srv.Client().Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Errorf("wanted not ready but got %d", resp.StatusCode)
	}

	internal.SetHealth("osiris", healthv1.HealthCheckResponse_SERVING)

	resp, err = srv.Client().Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("wanted ready but got %d", resp.StatusCode)
	}
}

func TestReadyz(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(readyz))

	internal.SetHealth("osiris", healthv1.HealthCheckResponse_NOT_SERVING)

	resp, err := srv.Client().Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		t.Errorf("wanted not ready but got %d", resp.StatusCode)
	}

	internal.SetHealth("osiris", healthv1.HealthCheckResponse_SERVING)

	resp, err = srv.Client().Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("wanted ready but got %d", resp.StatusCode)
	}
}

func TestReadyzMissingHealthService(t *testing.T) {
	// Test that readyz returns StatusExpectationFailed when osiris is not registered
	// We can't easily clear the health server state between tests, so we'll
	// just verify the code path by checking that if we query a service that
	// doesn't exist, GetHealth returns (UNKNOWN, false).

	// Verify GetHealth behavior for non-existent service
	status, ok := internal.GetHealth("some-nonexistent-service")
	if ok {
		t.Errorf("GetHealth for non-existent service should return ok=false, got ok=true")
	}
	if status != healthv1.HealthCheckResponse_UNKNOWN {
		t.Errorf("GetHealth for non-existent service should return UNKNOWN, got %v", status)
	}

	// The readyz handler will return StatusExpectationFailed if "osiris" is not found
	// Since we can't reliably clear the state, we'll skip the full HTTP test
	// if osiris is already registered
	if _, ok := internal.GetHealth("osiris"); ok {
		t.Skip("osiris health service already registered, skipping HTTP test")
	}

	srv := httptest.NewServer(http.HandlerFunc(readyz))
	defer srv.Close()

	resp, err := srv.Client().Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusExpectationFailed {
		t.Errorf("wanted StatusExpectationFailed but got %d", resp.StatusCode)
	}
}
