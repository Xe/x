package config

import (
	"errors"
	"fmt"
)

var (
	ErrWrongValue = errors.New("wrong value")
)

type Logging struct {
	AccessLog  string `hcl:"access_log"`
	MaxSizeMB  int    `hcl:"max_size_mb,optional"`
	MaxAgeDays int    `hcl:"max_age_days,optional"`
	MaxBackups int    `hcl:"max_backups,optional"`
	Compress   bool   `hcl:"compress,optional"`
}

func (l *Logging) Valid() error {
	var errs []error

	if l.MaxSizeMB < 0 {
		errs = append(errs, fmt.Errorf("%w: max_size_mb cannot be negative", ErrWrongValue))
	}

	if l.MaxSizeMB > 512 {
		errs = append(errs, fmt.Errorf("%w: max_size_mb cannot be greater than 512 MB as it doesn't make sense to allow a 512 MB log file", ErrWrongValue))
	}

	if l.MaxAgeDays < 0 {
		errs = append(errs, fmt.Errorf("%w: max_age_days cannot be negative", ErrWrongValue))
	}

	if l.MaxBackups < 0 {
		errs = append(errs, fmt.Errorf("%w: max_backups cannot be negative", ErrWrongValue))
	}

	if len(errs) != 0 {
		return errors.Join(errs...)
	}

	return nil
}
