package config

import (
	"errors"
	"net"
	"testing"
)

func TestValidateURLForSSRF(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr error
	}{
		// Public IPs - should pass
		{
			name:    "public IPv4",
			url:     "http://1.1.1.1:8080",
			wantErr: nil,
		},
		{
			name:    "public IPv6",
			url:     "http://[2606:4700:4700::1111]:8080",
			wantErr: nil,
		},
		{
			name:    "public domain",
			url:     "http://example.com:8080",
			wantErr: nil,
		},
		{
			name:    "public domain with https",
			url:     "https://example.com:8080",
			wantErr: nil,
		},
		{
			name:    "h2c scheme",
			url:     "h2c://example.com:8080",
			wantErr: nil,
		},

		// Loopback addresses - should fail
		{
			name:    "localhost IPv4",
			url:     "http://127.0.0.1:8080",
			wantErr: ErrPrivateIP,
		},
		{
			name:    "localhost IPv6",
			url:     "http://[::1]:8080",
			wantErr: ErrPrivateIP,
		},
		{
			name:    "localhost domain",
			url:     "http://localhost:8080",
			wantErr: nil, // DNS name allowed
		},

		// Private IPv4 ranges - should fail
		{
			name:    "10.0.0.0/8 private network",
			url:     "http://10.0.0.1:8080",
			wantErr: ErrPrivateIP,
		},
		{
			name:    "172.16.0.0/12 private network",
			url:     "http://172.16.0.1:8080",
			wantErr: ErrPrivateIP,
		},
		{
			name:    "192.168.0.0/16 private network",
			url:     "http://192.168.1.1:8080",
			wantErr: ErrPrivateIP,
		},

		// Link-local - should fail
		{
			name:    "169.254.0.0/16 link-local",
			url:     "http://169.254.1.1:8080",
			wantErr: ErrPrivateIP,
		},
		{
			name:    "AWS metadata",
			url:     "http://169.254.169.254:80",
			wantErr: ErrPrivateIP,
		},

		// CGNAT - should fail
		{
			name:    "100.64.0.0/10 CGNAT",
			url:     "http://100.64.0.1:8080",
			wantErr: ErrPrivateIP,
		},

		// Unix socket - should pass (exempt)
		{
			name:    "unix socket",
			url:     "unix:///tmp/app.sock",
			wantErr: nil,
		},

		// Other schemes - should pass
		{
			name:    "file scheme",
			url:     "file:///tmp/test",
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURLForSSRF(tt.url)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("ValidateURLForSSRF() error = %v, wantErr %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("ValidateURLForSSRF() unexpected error = %v", err)
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		// Loopback
		{"127.0.0.1", "127.0.0.1", true},
		{"127.0.0.2", "127.0.0.2", true},
		{"::1", "::1", true},

		// Private networks
		{"10.0.0.1", "10.0.0.1", true},
		{"172.16.0.1", "172.16.0.1", true},
		{"192.168.1.1", "192.168.1.1", true},

		// Link-local
		{"169.254.1.1", "169.254.1.1", true},
		{"169.254.169.254", "169.254.169.254", true},

		// CGNAT
		{"100.64.0.1", "100.64.0.1", true},

		// Public IPs
		{"8.8.8.8", "8.8.8.8", false},
		{"1.1.1.1", "1.1.1.1", false},
		{"93.184.216.34", "93.184.216.34", false}, // example.com

		// IPv6 public
		{"2606:4700:4700::1111", "2606:4700:4700::1111", false},
		{"2001:4860:4860::8888", "2001:4860:4860::8888", false}, // Google DNS
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIPNoZone(tt.ip)
			if got := isPrivateIP(ip); got != tt.want {
				t.Errorf("isPrivateIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDomainValidWithSSRF(t *testing.T) {
	tests := []struct {
		name  string
		input Domain
		want  error
	}{
		{
			name: "public IP target",
			input: Domain{
				Name: "example.com",
				TLS: TLS{
					Cert: "./testdata/tls/selfsigned.crt",
					Key:  "./testdata/tls/selfsigned.key",
				},
				Target:       "http://1.1.1.1:8080",
				HealthTarget: "http://1.1.1.1:9091/healthz",
			},
			want: nil,
		},
		{
			name: "localhost target without opt-out",
			input: Domain{
				Name: "example.com",
				TLS: TLS{
					Cert: "./testdata/tls/selfsigned.crt",
					Key:  "./testdata/tls/selfsigned.key",
				},
				Target:       "http://127.0.0.1:8080",
				HealthTarget: "http://127.0.0.1:9091/healthz",
			},
			want: ErrPrivateIP,
		},
		{
			name: "private IP target without opt-out",
			input: Domain{
				Name: "example.com",
				TLS: TLS{
					Cert: "./testdata/tls/selfsigned.crt",
					Key:  "./testdata/tls/selfsigned.key",
				},
				Target:       "http://10.0.0.1:8080",
				HealthTarget: "http://10.0.0.1:9091/healthz",
			},
			want: ErrPrivateIP,
		},
		{
			name: "localhost target with opt-out",
			input: Domain{
				Name:               "example.com",
				AllowPrivateTarget: true,
				TLS: TLS{
					Cert: "./testdata/tls/selfsigned.crt",
					Key:  "./testdata/tls/selfsigned.key",
				},
				Target:       "http://127.0.0.1:8080",
				HealthTarget: "http://127.0.0.1:9091/healthz",
			},
			want: nil,
		},
		{
			name: "private IP target with opt-out",
			input: Domain{
				Name:               "example.com",
				AllowPrivateTarget: true,
				TLS: TLS{
					Cert: "./testdata/tls/selfsigned.crt",
					Key:  "./testdata/tls/selfsigned.key",
				},
				Target:       "http://192.168.1.1:8080",
				HealthTarget: "http://192.168.1.1:9091/healthz",
			},
			want: nil,
		},
		{
			name: "domain name target (DNS rebinding risk accepted)",
			input: Domain{
				Name: "example.com",
				TLS: TLS{
					Cert: "./testdata/tls/selfsigned.crt",
					Key:  "./testdata/tls/selfsigned.key",
				},
				Target:       "http://localhost:8080",
				HealthTarget: "http://localhost:9091/healthz",
			},
			want: nil, // DNS names are allowed
		},
		{
			name: "unix socket target",
			input: Domain{
				Name: "example.com",
				TLS: TLS{
					Cert: "./testdata/tls/selfsigned.crt",
					Key:  "./testdata/tls/selfsigned.key",
				},
				Target:       "unix:///tmp/app.sock",
				HealthTarget: "unix:///tmp/health.sock",
			},
			want: nil,
		},
		{
			name: "public domain target",
			input: Domain{
				Name: "example.com",
				TLS: TLS{
					Cert: "./testdata/tls/selfsigned.crt",
					Key:  "./testdata/tls/selfsigned.key",
				},
				Target:       "http://example.com:8080",
				HealthTarget: "http://example.com:9091/healthz",
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Valid()
			if tt.want != nil {
				if err == nil {
					t.Errorf("Expected error containing %v, got nil", tt.want)
				} else if !errors.Is(err, tt.want) {
					t.Errorf("Expected error containing %v, got %v", tt.want, err)
				}
			} else if err != nil {
				t.Errorf("Expected nil error, got %v", err)
			}
		})
	}
}

// parseIPNoZone parses an IP string without zone identifier
func parseIPNoZone(s string) net.IP {
	ip := net.ParseIP(s)
	if ip == nil {
		return nil
	}
	// Return IP without zone
	return ip
}
