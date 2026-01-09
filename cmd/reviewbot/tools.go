package main

import (
	"errors"
	"log/slog"
)

type SubmitReviewParams struct {
	Approved bool   `json:"approved"`
	Message  string `json:"message"`
}

func (srp SubmitReviewParams) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Bool("approved", srp.Approved),
		slog.String("message", srp.Message),
	)
}

func (srp SubmitReviewParams) Valid() error {
	if srp.Message == "" {
		return errors.New("message is required")
	}

	return nil
}
