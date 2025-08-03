package config

import (
	"errors"
	"fmt"

	"within.website/x/cmd/sakurajima/internal/logging/expressions"
)

var (
	ErrWrongValue          = errors.New("wrong value")
	ErrFilterDoesntCompile = errors.New("config: filter does not compile")
)

type Logging struct {
	AccessLog  string `hcl:"access_log"`
	MaxSizeMB  int    `hcl:"max_size_mb,optional"`
	MaxAgeDays int    `hcl:"max_age_days,optional"`
	MaxBackups int    `hcl:"max_backups,optional"`
	Compress   bool   `hcl:"compress,optional"`

	Filters []Filter `hcl:"filter,block"`
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

	for _, filter := range l.Filters {
		if err := filter.Valid(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return errors.Join(errs...)
	}

	return nil
}

type Filter struct {
	Name       string `hcl:"name,label"`
	Expression string `hcl:"expression"`
}

func (f Filter) Valid() error {
	var errs []error

	if err := expressions.TryCompile(f.Expression); err != nil {
		errs = append(errs, fmt.Errorf("%w: %s Compile(%q): %w", ErrFilterDoesntCompile, f.Name, f.Expression, err))
	}

	if len(errs) != 0 {
		return errors.Join(errs...)
	}

	return nil
}
