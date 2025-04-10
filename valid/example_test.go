package valid

import (
	"errors"
	"log/slog"
)

type Example struct {
	Name string
}

func (e *Example) Valid() error {
	var errs []error
	if e.Name == "" {
		errs = append(errs, errors.New("name is empty"))
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Join(errs...)
}

func ExampleInterface_Valid() {
	e := Example{Name: ""}

	if err := e.Valid(); err != nil {
		slog.Error("validation failed", "err", err)
	}
}
