package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/idna"
)

var (
	ErrInvalidDomainName      = errors.New("domain: name is invalid")
	ErrInvalidDomainTLSConfig = errors.New("domain: TLS config is invalid")
	ErrInvalidURL             = errors.New("invalid URL")
	ErrInvalidURLScheme       = errors.New("URL has invalid scheme")
	ErrInvalidTimeout         = errors.New("invalid timeout duration")
)

// Timeouts defines timeout durations for HTTP connections to a backend.
type Timeouts struct {
	Dial           string `hcl:"dial,optional"`
	ResponseHeader string `hcl:"response_header,optional"`
	Idle           string `hcl:"idle,optional"`
}

// DefaultTimeouts returns sensible default timeout values.
func DefaultTimeouts() Timeouts {
	return Timeouts{
		Dial:           "5s",
		ResponseHeader: "10s",
		Idle:           "90s",
	}
}

// Parse parses the human-readable timeout strings into time.Duration values.
func (t Timeouts) Parse() (dial, responseHeader, idle time.Duration, err error) {
	if t.Dial == "" && t.ResponseHeader == "" && t.Idle == "" {
		// Use defaults if no timeouts are specified
		t = DefaultTimeouts()
	}

	if t.Dial != "" {
		dial, err = time.ParseDuration(t.Dial)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("dial timeout: %w", err)
		}
	}

	if t.ResponseHeader != "" {
		responseHeader, err = time.ParseDuration(t.ResponseHeader)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("response header timeout: %w", err)
		}
	}

	if t.Idle != "" {
		idle, err = time.ParseDuration(t.Idle)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("idle timeout: %w", err)
		}
	}

	return dial, responseHeader, idle, nil
}

type Domain struct {
	Name               string   `hcl:"name,label"`
	TLS                TLS      `hcl:"tls,block"`
	Target             string   `hcl:"target"`
	InsecureSkipVerify bool     `hcl:"insecure_skip_verify,optional"`
	HealthTarget       string   `hcl:"health_target"`
	AllowPrivateTarget bool     `hcl:"allow_private_target,optional"`
	Timeouts           Timeouts `hcl:"timeouts,block"`
	Limits             *Limits  `hcl:"limits,block"`
}

func (d Domain) Valid() error {
	var errs []error

	if _, err := idna.Lookup.ToASCII(d.Name); err != nil {
		errs = append(errs, fmt.Errorf("%w %q: %w", ErrInvalidDomainName, d.Name, err))
	}

	if err := d.TLS.Valid(); err != nil {
		errs = append(errs, fmt.Errorf("%w: %w", ErrInvalidDomainTLSConfig, err))
	}

	if err := isURLValid(d.Target); err != nil {
		errs = append(errs, fmt.Errorf("target has %w %q: %w", ErrInvalidURL, d.Target, err))
	}

	if err := isURLValid(d.HealthTarget); err != nil {
		errs = append(errs, fmt.Errorf("health_target has %w %q: %w", ErrInvalidURL, d.HealthTarget, err))
	}

	// SSRF protection: validate targets don't point to private IPs
	if !d.AllowPrivateTarget {
		if err := ValidateURLForSSRF(d.Target); err != nil {
			errs = append(errs, fmt.Errorf("target SSRF validation failed %q: %w", d.Target, err))
		}
		if err := ValidateURLForSSRF(d.HealthTarget); err != nil {
			errs = append(errs, fmt.Errorf("health_target SSRF validation failed %q: %w", d.HealthTarget, err))
		}
	}

	// Validate InsecureSkipVerify is only used with HTTPS targets
	if d.InsecureSkipVerify {
		u, err := url.Parse(d.Target)
		if err == nil && u.Scheme != "https" {
			errs = append(errs, fmt.Errorf("insecure_skip_verify is only valid for https:// targets, got %s", u.Scheme))
		}
	}

	if d.Limits != nil {
		if err := d.Limits.Valid(); err != nil {
			errs = append(errs, fmt.Errorf("limits config is invalid: %w", err))
		}
	}

	if _, _, _, err := d.Timeouts.Parse(); err != nil {
		errs = append(errs, fmt.Errorf("%w: %w", ErrInvalidTimeout, err))
	}

	if len(errs) != 0 {
		return errors.Join(errs...)
	}

	return nil
}

func isURLValid(input string) error {
	u, err := url.Parse(input)
	if err != nil {
		return err
	}

	switch u.Scheme {
	case "http", "https", "h2c":
		// do nothing
	case "unix":
		socketPath := strings.TrimPrefix(input, "unix://")
		if strings.Contains(socketPath, "../") {
			return fmt.Errorf("%w unix socket path contains path traversal: %s", ErrInvalidURLScheme, socketPath)
		}
		if socketPath == "" {
			return fmt.Errorf("%w unix socket path is empty", ErrInvalidURLScheme)
		}
	default:
		return fmt.Errorf("%w %s has scheme %s (want http, https, h2c, unix)", ErrInvalidURLScheme, input, u.Scheme)
	}

	return nil
}
