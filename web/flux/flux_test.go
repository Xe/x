package flux

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Mock server for testing
func mockServer() *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/health-check", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	return httptest.NewServer(handler)
}

func TestHealthCheck(t *testing.T) {
	server := mockServer()
	defer server.Close()

	client := NewClient(server.URL)
	healthResp, err := client.HealthCheck()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if healthResp.Status != "ok" {
		t.Fatalf("expected status 'ok', got %v", healthResp.Status)
	}
}
