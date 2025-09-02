package config

import (
	"errors"
	"fmt"
	"net/http"
)

type Toplevel struct {
	Bind     Bind      `hcl:"bind,block"`
	Domains  []Domain  `hcl:"domain,block"`
	Logging  Logging   `hcl:"logging,block"`
	Autocert *Autocert `hcl:"autocert,block"`
}

type Autocert struct {
	S3Bucket         string `hcl:"s3_bucket"`
	S3Prefix         string `hcl:"s3_prefix,optional"`
	Email            string `hcl:"email,optional"`
	DirectoryURL     string `hcl:"directory_url,optional"`
	HTTPRedirectCode int    `hcl:"http_redirect_code,optional"`
}

func (a *Autocert) Valid() error {
	var errs []error

	if a.HTTPRedirectCode != 0 {
		switch a.HTTPRedirectCode {
		case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
			// valid
		default:
			errs = append(errs, fmt.Errorf("autocert: invalid http_redirect_code %d", a.HTTPRedirectCode))
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func (t *Toplevel) Valid() error {
	var errs []error

	if err := t.Bind.Valid(); err != nil {
		errs = append(errs, fmt.Errorf("invalid bind block:\n%w", err))
	}

	var needsAutocert bool
	for _, d := range t.Domains {
		if err := d.Valid(); err != nil {
			errs = append(errs, fmt.Errorf("when parsing domain %s: %w", d.Name, err))
		}
		if d.TLS.Autocert {
			needsAutocert = true
		}
	}

	if err := t.Logging.Valid(); err != nil {
		errs = append(errs, fmt.Errorf("invalid logging block:\n%w", err))
	}

	if needsAutocert {
		if t.Autocert.S3Bucket == "" {
			errs = append(errs, fmt.Errorf("autocert: s3_bucket is required when any domain has tls.autocert = true"))
		}
		if err := t.Autocert.Valid(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return fmt.Errorf("invalid configuration file:\n%w", errors.Join(errs...))
	}

	return nil
}
