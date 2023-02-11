package run

import (
	"context"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

type Process struct {
	Output []byte
}

func (p *Process) Putchar(ctx context.Context, stack []uint64) {
	x := api.DecodeI32(stack[0])
	p.putchar(x)
}

func (p *Process) putchar(char int32) {
	p.Output = append(p.Output, byte(char))
}

func Run(bin []byte) (*ExecResult, error) {
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)
	defer r.Close(ctx)

	p := &Process{}

	env, err := r.NewHostModuleBuilder("h").
		NewFunctionBuilder().
		WithGoFunction(api.GoFunc(p.Putchar), []api.ValueType{api.ValueTypeI32}, nil).
		Export("h").
		Instantiate(ctx)
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
