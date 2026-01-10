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

type ReadFileParams struct {
	Path      string `json:"path"`
	StartLine *int   `json:"start_line,omitempty"`
	EndLine   *int   `json:"end_line,omitempty"`
}

func (rfp ReadFileParams) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("path", rfp.Path),
		slog.Any("start_line", rfp.StartLine),
		slog.Any("end_line", rfp.EndLine),
	)
}

func (rfp ReadFileParams) Valid() error {
	if rfp.Path == "" {
		return errors.New("path is required")
	}
	if rfp.StartLine != nil && *rfp.StartLine < 1 {
		return errors.New("start_line must be >= 1")
	}
	if rfp.EndLine != nil && *rfp.EndLine < 1 {
		return errors.New("end_line must be >= 1")
	}
	if rfp.StartLine != nil && rfp.EndLine != nil && *rfp.StartLine > *rfp.EndLine {
		return errors.New("start_line must be <= end_line")
	}
	return nil
}
