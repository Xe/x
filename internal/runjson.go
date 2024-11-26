package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

// Zilch returns the zero value of a given type.
func Zilch[T any]() T { return *new(T) }

func RunJSON[T any](ctx context.Context, program string, args ...any) (T, error) {
	exePath, err := exec.LookPath(program)
	if err != nil {
		return Zilch[T](), fmt.Errorf("can't find %s: %w", program, err)
	}

	var argStr []string

	for _, arg := range args {
		argStr = append(argStr, fmt.Sprint(arg))
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd := exec.CommandContext(ctx, exePath, argStr...)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		os.Stderr.Write(stderr.Bytes())
		return Zilch[T](), fmt.Errorf("can't run %s: %w", program, err)
	}

	var result T
	if err := json.NewDecoder(&stdout).Decode(&result); err != nil {
		return Zilch[T](), fmt.Errorf("can't decode json: %w", err)
	}

	return result, nil
}
