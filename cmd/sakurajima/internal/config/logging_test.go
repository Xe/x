package config

import (
	"errors"
	"testing"
)

func TestLoggingValid(t *testing.T) {
	for _, tt := range []struct {
		name  string
		input Logging
		err   error
	}{
		{
			name: "valid configuration",
			input: Logging{
				AccessLog:  "/var/log/access.log",
				MaxSizeMB:  100,
				MaxAgeDays: 30,
				MaxBackups: 5,
				Compress:   true,
			},
		},
		{
			name: "zero values are valid",
			input: Logging{
				AccessLog:  "/var/log/access.log",
				MaxSizeMB:  0,
				MaxAgeDays: 0,
				MaxBackups: 0,
				Compress:   false,
			},
		},
		{
			name: "negative max_size_mb",
			input: Logging{
				AccessLog:  "/var/log/access.log",
				MaxSizeMB:  -1,
				MaxAgeDays: 30,
				MaxBackups: 5,
				Compress:   true,
			},
			err: ErrWrongValue,
		},
		{
			name: "negative max_age_days",
			input: Logging{
				AccessLog:  "/var/log/access.log",
				MaxSizeMB:  100,
				MaxAgeDays: -1,
				MaxBackups: 5,
				Compress:   true,
			},
			err: ErrWrongValue,
		},
		{
			name: "negative max_backups",
			input: Logging{
				AccessLog:  "/var/log/access.log",
				MaxSizeMB:  100,
				MaxAgeDays: 30,
				MaxBackups: -1,
				Compress:   true,
			},
			err: ErrWrongValue,
		},
		{
			name: "max_size_mb too large",
			input: Logging{
				AccessLog:  "/var/log/access.log",
				MaxSizeMB:  513,
				MaxAgeDays: 30,
				MaxBackups: 5,
				Compress:   true,
			},
			err: ErrWrongValue,
		},
		{
			name: "max_size_mb exactly 512 is valid",
			input: Logging{
				AccessLog:  "/var/log/access.log",
				MaxSizeMB:  512,
				MaxAgeDays: 30,
				MaxBackups: 5,
				Compress:   true,
			},
		},
		{
			name: "multiple negative values",
			input: Logging{
				AccessLog:  "/var/log/access.log",
				MaxSizeMB:  -1,
				MaxAgeDays: -1,
				MaxBackups: -1,
				Compress:   true,
			},
			err: ErrWrongValue,
		},
		{
			name: "filter with only expression",
			input: Logging{
				AccessLog:  "/var/log/access.log",
				MaxSizeMB:  100,
				MaxAgeDays: 30,
				MaxBackups: 5,
				Compress:   true,
				Filters: []Filter{
					{Expression: "true"},
				},
			},
		},
		{
			name: "uncompilable filter",
			input: Logging{
				AccessLog:  "/var/log/access.log",
				MaxSizeMB:  100,
				MaxAgeDays: 30,
				MaxBackups: 5,
				Compress:   true,
				Filters: []Filter{
					{Expression: "taco"},
				},
			},
			err: ErrFilterDoesntCompile,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.input.Valid(); !errors.Is(err, tt.err) {
				t.Logf("want: %v", tt.err)
				t.Logf("got:  %v", err)
				t.Error("got wrong error from validation function")
			} else {
				t.Log(err)
			}
		})
	}
}
