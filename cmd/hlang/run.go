package main

import (
	"context"
	"time"

	"github.com/tetratelabs/wazero"
)

type Process struct {
	Output []byte
}

func (p *Process) Putchar(char int32) {
	p.Output = append(p.Output, byte(char))
}

func run(bin []byte) (*ExecResult, error) {
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	p := &Process{}

	env, err := r.NewHostModuleBuilder("h").NewFunctionBuilder().WithFunc(func(char int32) { p.Putchar(char) }).Export("h").Instantiate(ctx, r)
	if err != nil {
		return nil, err
	}
	defer env.Close(ctx)

	code, err := r.CompileModule(ctx, bin)
	if err != nil {
		return nil, err
	}

	mod, err := r.InstantiateModule(ctx, code, wazero.NewModuleConfig())
	if err != nil {
		return nil, err
	}
	defer mod.Close(ctx)

	t0 := time.Now()
	if _, err = mod.ExportedFunction("h").Call(ctx); err != nil {
		return nil, err
	}
	runTime := time.Since(t0)

	return &ExecResult{
		Output:   string(p.Output),
		ExecTime: runTime,
	}, nil
}

type ExecResult struct {
	Output   string        `json:"out"`
	ExecTime time.Duration `json:"exec_duration"`
}
