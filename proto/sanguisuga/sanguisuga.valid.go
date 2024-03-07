package sanguisuga

import (
	"errors"
	"fmt"
)

func (s *Show) Valid() error {
	var errs []error

	if s.Title == "" {
		errs = append(errs, errors.New("title is empty"))
	}

	if s.DiskPath == "" {
		errs = append(errs, errors.New("disk path is empty"))
	}

	if s.Quality != "1080p" {
		errs = append(errs, errors.New("quality is not 1080p"))
	}

	if len(errs) == 0 {
		return nil
	}

	return fmt.Errorf("show is invalid: %w", errors.Join(errs...))
}
