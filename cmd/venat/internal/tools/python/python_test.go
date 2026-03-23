package python

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"within.website/x/cmd/venat/internal/agentloop"
	cipython "within.website/x/llm/codeinterpreter/python"
)

// Compile-time check that Impl satisfies agentloop.Tool.
var _ agentloop.Tool = Impl{}

func TestInputValid(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name  string
		input Input
		err   error
	}{
		{name: "valid code", input: Input{Code: "print('hello')"}},
		{name: "empty code", input: Input{Code: ""}, err: ErrNoCode},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Valid()
			if tt.err != nil {
				if !errors.Is(err, tt.err) {
					t.Logf("want: %v", tt.err)
					t.Logf("got:  %v", err)
					t.Error("got wrong error")
				}
			} else if err != nil {
				t.Logf("unexpected error: %v", err)
				t.Error("expected no error")
			}
		})
	}
}

func TestImplValid(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name        string
		data        []byte
		wantErr     bool
		errContains string
	}{
		{
			name: "valid input",
			data: []byte(`{"code":"print('hello')"}`),
		},
		{
			name:        "empty code field",
			data:        []byte(`{"code":""}`),
			wantErr:     true,
			errContains: "no code provided",
		},
		{
			name:        "missing code field",
			data:        []byte(`{}`),
			wantErr:     true,
			errContains: "no code provided",
		},
		{
			name:        "invalid json",
			data:        []byte(`not json`),
			wantErr:     true,
			errContains: "can't parse json",
		},
		{
			name:        "empty input",
			data:        []byte(``),
			wantErr:     true,
			errContains: "can't parse json",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var impl Impl
			err := impl.Valid(tt.data)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errContains != "" {
					if got := err.Error(); !strings.Contains(got, tt.errContains) {
						t.Logf("want substring: %q", tt.errContains)
						t.Logf("got:           %q", got)
						t.Error("error message mismatch")
					}
				}
			} else if err != nil {
				t.Logf("unexpected error: %v", err)
				t.Error("expected no error")
			}
		})
	}
}

func TestImplRun(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		name       string
		data       []byte
		wantErr    bool
		wantStdout string
	}{
		{
			name:       "print hello",
			data:       []byte(`{"code":"print('hello')"}`),
			wantStdout: "hello\n",
		},
		{
			name:       "arithmetic",
			data:       []byte(`{"code":"print(2 + 2)"}`),
			wantStdout: "4\n",
		},
		{
			name:    "invalid json",
			data:    []byte(`not json`),
			wantErr: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var impl Impl
			out, err := impl.Run(context.Background(), tt.data)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var result cipython.Result
			if err := json.Unmarshal(out, &result); err != nil {
				t.Fatalf("can't unmarshal result: %v", err)
			}

			if result.Stdout != tt.wantStdout {
				t.Logf("want: %q", tt.wantStdout)
				t.Logf("got:  %q", result.Stdout)
				t.Error("unexpected stdout")
			}
		})
	}
}
