package config

import (
	"errors"
	"strings"
	"testing"
)

func TestDomainValid(t *testing.T) {
	for _, tt := range []struct {
		name        string
		input       Domain
		err         error
		errContains string
	}{
		{
			name: "simple happy path",
			input: Domain{
				Name: "anubis.techaro.lol",
				TLS: TLS{
					Cert: "./testdata/tls/selfsigned.crt",
					Key:  "./testdata/tls/selfsigned.key",
				},
				Target:       "http://localhost:3000",
				HealthTarget: "http://localhost:9091/healthz",
			},
		},
		{
			name: "invalid domain name",
			input: Domain{
				Name: "\uFFFD.techaro.lol",
				TLS: TLS{
					Cert: "./testdata/tls/selfsigned.crt",
					Key:  "./testdata/tls/selfsigned.key",
				},
				Target:       "http://localhost:3000",
				HealthTarget: "http://localhost:9091/healthz",
			},
			err: ErrInvalidDomainName,
		},
		{
			name: "invalid tls config",
			input: Domain{
				Name: "anubis.techaro.lol",
				TLS: TLS{
					Cert: "./testdata/tls/invalid.crt",
					Key:  "./testdata/tls/invalid.key",
				},
				Target:       "http://localhost:3000",
				HealthTarget: "http://localhost:9091/healthz",
			},
			err: ErrInvalidDomainTLSConfig,
		},
		{
			name: "invalid URL",
			input: Domain{
				Name: "anubis.techaro.lol",
				TLS: TLS{
					Cert: "./testdata/tls/selfsigned.crt",
					Key:  "./testdata/tls/selfsigned.key",
				},
				Target:       "file://[::1:3000",
				HealthTarget: "file://[::1:9091/healthz",
			},
			err: ErrInvalidURL,
		},
		{
			name: "wrong URL scheme",
			input: Domain{
				Name: "anubis.techaro.lol",
				TLS: TLS{
					Cert: "./testdata/tls/selfsigned.crt",
					Key:  "./testdata/tls/selfsigned.key",
				},
				Target:       "file://localhost:3000",
				HealthTarget: "file://localhost:9091/healthz",
			},
			err: ErrInvalidURLScheme,
		},
		{
			name: "insecure_skip_verify with https target",
			input: Domain{
				Name:               "anubis.techaro.lol",
				InsecureSkipVerify: true,
				TLS: TLS{
					Cert: "./testdata/tls/selfsigned.crt",
					Key:  "./testdata/tls/selfsigned.key",
				},
				Target:       "https://localhost:3000",
				HealthTarget: "http://localhost:9091/healthz",
			},
		},
		{
			name: "insecure_skip_verify with http target",
			input: Domain{
				Name:               "anubis.techaro.lol",
				InsecureSkipVerify: true,
				TLS: TLS{
					Cert: "./testdata/tls/selfsigned.crt",
					Key:  "./testdata/tls/selfsigned.key",
				},
				Target:       "http://localhost:3000",
				HealthTarget: "http://localhost:9091/healthz",
			},
			errContains: "insecure_skip_verify is only valid for https:// targets",
		},
		{
			name: "insecure_skip_verify with h2c target",
			input: Domain{
				Name:               "anubis.techaro.lol",
				InsecureSkipVerify: true,
				TLS: TLS{
					Cert: "./testdata/tls/selfsigned.crt",
					Key:  "./testdata/tls/selfsigned.key",
				},
				Target:       "h2c://localhost:3000",
				HealthTarget: "http://localhost:9091/healthz",
			},
			errContains: "insecure_skip_verify is only valid for https:// targets",
		},
		{
			name: "insecure_skip_verify with unix target",
			input: Domain{
				Name:               "anubis.techaro.lol",
				InsecureSkipVerify: true,
				TLS: TLS{
					Cert: "./testdata/tls/selfsigned.crt",
					Key:  "./testdata/tls/selfsigned.key",
				},
				Target:       "unix://.sock",
				HealthTarget: "http://localhost:9091/healthz",
			},
			errContains: "insecure_skip_verify is only valid for https:// targets",
		},
		{
			name: "unix socket path traversal attack",
			input: Domain{
				Name: "anubis.techaro.lol",
				TLS: TLS{
					Cert: "./testdata/tls/selfsigned.crt",
					Key:  "./testdata/tls/selfsigned.key",
				},
				Target:       "unix://../../../etc/passwd",
				HealthTarget: "http://localhost:9091/healthz",
			},
			err: ErrInvalidURLScheme,
		},
		{
			name: "unix socket empty path",
			input: Domain{
				Name: "anubis.techaro.lol",
				TLS: TLS{
					Cert: "./testdata/tls/selfsigned.crt",
					Key:  "./testdata/tls/selfsigned.key",
				},
				Target:       "unix://",
				HealthTarget: "http://localhost:9091/healthz",
			},
			err: ErrInvalidURLScheme,
		},
		{
			name: "unix socket valid path",
			input: Domain{
				Name: "anubis.techaro.lol",
				TLS: TLS{
					Cert: "./testdata/tls/selfsigned.crt",
					Key:  "./testdata/tls/selfsigned.key",
				},
				Target:       "unix:///var/run/app.sock",
				HealthTarget: "http://localhost:9091/healthz",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Valid()
			if tt.err != nil {
				if !errors.Is(err, tt.err) {
					t.Logf("want: %v", tt.err)
					t.Logf("got:  %v", err)
					t.Error("got wrong error from validation function")
				} else {
					t.Log(err)
				}
			} else if tt.errContains != "" {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Logf("want error containing: %v", tt.errContains)
					t.Logf("got:  %v", err)
					t.Error("got wrong error from validation function")
				} else {
					t.Log(err)
				}
			} else {
				if err != nil {
					t.Logf("want: nil")
					t.Logf("got:  %v", err)
					t.Error("got unexpected error from validation function")
				}
			}
		})
	}
}
