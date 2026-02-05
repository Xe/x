package config

import (
	"errors"
	"strings"
	"testing"
	"time"
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

func TestTimeoutsParse(t *testing.T) {
	for _, tt := range []struct {
		name               string
		input              Timeouts
		wantDial           time.Duration
		wantResponseHeader time.Duration
		wantIdle           time.Duration
		wantErr            bool
	}{
		{
			name: "all timeouts specified",
			input: Timeouts{
				Dial:           "5s",
				ResponseHeader: "10s",
				Idle:           "90s",
			},
			wantDial:           5 * time.Second,
			wantResponseHeader: 10 * time.Second,
			wantIdle:           90 * time.Second,
			wantErr:            false,
		},
		{
			name:               "no timeouts specified - uses defaults",
			input:              Timeouts{},
			wantDial:           5 * time.Second,
			wantResponseHeader: 10 * time.Second,
			wantIdle:           90 * time.Second,
			wantErr:            false,
		},
		{
			name: "partial timeouts specified",
			input: Timeouts{
				Dial: "3s",
			},
			wantDial:           3 * time.Second,
			wantResponseHeader: 0, // not specified
			wantIdle:           0, // not specified
			wantErr:            false,
		},
		{
			name: "milliseconds",
			input: Timeouts{
				Dial:           "500ms",
				ResponseHeader: "1000ms",
				Idle:           "30000ms",
			},
			wantDial:           500 * time.Millisecond,
			wantResponseHeader: 1000 * time.Millisecond,
			wantIdle:           30000 * time.Millisecond,
			wantErr:            false,
		},
		{
			name: "mixed duration units",
			input: Timeouts{
				Dial:           "100ms",
				ResponseHeader: "5s",
				Idle:           "2m",
			},
			wantDial:           100 * time.Millisecond,
			wantResponseHeader: 5 * time.Second,
			wantIdle:           2 * time.Minute,
			wantErr:            false,
		},
		{
			name: "invalid dial timeout",
			input: Timeouts{
				Dial: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid response header timeout",
			input: Timeouts{
				ResponseHeader: "foo",
			},
			wantErr: true,
		},
		{
			name: "invalid idle timeout",
			input: Timeouts{
				Idle: "bar",
			},
			wantErr: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dial, responseHeader, idle, err := tt.input.Parse()

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if dial != tt.wantDial {
				t.Errorf("Dial timeout = %v, want %v", dial, tt.wantDial)
			}

			if responseHeader != tt.wantResponseHeader {
				t.Errorf("ResponseHeader timeout = %v, want %v", responseHeader, tt.wantResponseHeader)
			}

			if idle != tt.wantIdle {
				t.Errorf("Idle timeout = %v, want %v", idle, tt.wantIdle)
			}
		})
	}
}
