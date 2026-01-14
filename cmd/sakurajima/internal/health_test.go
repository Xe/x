package internal

import (
	"testing"

	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

func TestSetHealthGetHealth(t *testing.T) {
	cases := []struct {
		name          string
		service       string
		setStatus     healthv1.HealthCheckResponse_ServingStatus
		wantStatus    healthv1.HealthCheckResponse_ServingStatus
		wantFound     bool
	}{
		{
			name:       "service is serving",
			service:    "api",
			setStatus:  healthv1.HealthCheckResponse_SERVING,
			wantStatus: healthv1.HealthCheckResponse_SERVING,
			wantFound:  true,
		},
		{
			name:       "service is not serving",
			service:    "api",
			setStatus:  healthv1.HealthCheckResponse_NOT_SERVING,
			wantStatus: healthv1.HealthCheckResponse_NOT_SERVING,
			wantFound:  true,
		},
		{
			name:       "service status unknown",
			service:    "api",
			setStatus:  healthv1.HealthCheckResponse_UNKNOWN,
			wantStatus: healthv1.HealthCheckResponse_UNKNOWN,
			wantFound:  true,
		},
		{
			name:       "multiple services - first",
			service:    "service-a",
			setStatus:  healthv1.HealthCheckResponse_SERVING,
			wantStatus: healthv1.HealthCheckResponse_SERVING,
			wantFound:  true,
		},
		{
			name:       "multiple services - second",
			service:    "service-b",
			setStatus:  healthv1.HealthCheckResponse_NOT_SERVING,
			wantStatus: healthv1.HealthCheckResponse_NOT_SERVING,
			wantFound:  true,
		},
		{
			name:       "service with hyphens",
			service:    "my-grpc-service",
			setStatus:  healthv1.HealthCheckResponse_SERVING,
			wantStatus: healthv1.HealthCheckResponse_SERVING,
			wantFound:  true,
		},
		{
			name:       "service with dots",
			service:    "api.v1.service",
			setStatus:  healthv1.HealthCheckResponse_SERVING,
			wantStatus: healthv1.HealthCheckResponse_SERVING,
			wantFound:  true,
		},
		{
			name:       "empty service name",
			service:    "",
			setStatus:  healthv1.HealthCheckResponse_SERVING,
			wantStatus: healthv1.HealthCheckResponse_SERVING,
			wantFound:  true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			SetHealth(tt.service, tt.setStatus)

			gotStatus, gotFound := GetHealth(tt.service)
			if gotStatus != tt.wantStatus {
				t.Errorf("GetHealth(%q) status = %v, want %v", tt.service, gotStatus, tt.wantStatus)
			}
			if gotFound != tt.wantFound {
				t.Errorf("GetHealth(%q) found = %v, want %v", tt.service, gotFound, tt.wantFound)
			}
		})
	}
}

func TestGetHealthNotFound(t *testing.T) {
	cases := []struct {
		name    string
		service string
	}{
		{
			name:    "non-existent service",
			service: "does-not-exist",
		},
		{
			name:    "never set service",
			service: "uninitialized-service",
		},
		{
			name:    "service with special chars",
			service: "service/with/slashes",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			gotStatus, gotFound := GetHealth(tt.service)
			if gotFound {
				t.Errorf("GetHealth(%q) found = %v, want false", tt.service, gotFound)
			}
			if gotStatus != healthv1.HealthCheckResponse_UNKNOWN {
				t.Errorf("GetHealth(%q) status = %v, want UNKNOWN", tt.service, gotStatus)
			}
		})
	}
}

func TestSetHealthOverwrite(t *testing.T) {
	cases := []struct {
		name       string
		service    string
		firstStatus  healthv1.HealthCheckResponse_ServingStatus
		secondStatus healthv1.HealthCheckResponse_ServingStatus
		wantStatus healthv1.HealthCheckResponse_ServingStatus
	}{
		{
			name:         "serving to not serving",
			service:      "api",
			firstStatus:  healthv1.HealthCheckResponse_SERVING,
			secondStatus: healthv1.HealthCheckResponse_NOT_SERVING,
			wantStatus:   healthv1.HealthCheckResponse_NOT_SERVING,
		},
		{
			name:         "not serving to serving",
			service:      "api",
			firstStatus:  healthv1.HealthCheckResponse_NOT_SERVING,
			secondStatus: healthv1.HealthCheckResponse_SERVING,
			wantStatus:   healthv1.HealthCheckResponse_SERVING,
		},
		{
			name:         "unknown to serving",
			service:      "api",
			firstStatus:  healthv1.HealthCheckResponse_UNKNOWN,
			secondStatus: healthv1.HealthCheckResponse_SERVING,
			wantStatus:   healthv1.HealthCheckResponse_SERVING,
		},
		{
			name:         "serving to unknown",
			service:      "api",
			firstStatus:  healthv1.HealthCheckResponse_SERVING,
			secondStatus: healthv1.HealthCheckResponse_UNKNOWN,
			wantStatus:   healthv1.HealthCheckResponse_UNKNOWN,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			SetHealth(tt.service, tt.firstStatus)
			SetHealth(tt.service, tt.secondStatus)

			gotStatus, gotFound := GetHealth(tt.service)
			if !gotFound {
				t.Errorf("GetHealth(%q) found = false, want true", tt.service)
			}
			if gotStatus != tt.wantStatus {
				t.Errorf("GetHealth(%q) status = %v, want %v", tt.service, gotStatus, tt.wantStatus)
			}
		})
	}
}
