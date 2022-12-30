package main

import (
	"errors"
	"time"

	"github.com/perlin-network/life/compiler"
	"github.com/perlin-network/life/exec"
)

type Process struct {
	Output []byte
}

// ResolveGlobal does nothing, currently.
func (p *Process) ResolveGlobal(module, field string) int64 { return 0 }

// ResolveFunc resolves h's ABI and importable function.
func (p *Process) ResolveFunc(module, field string) exec.FunctionImport {
	switch module {
	case "h":
		switch field {
		case "h":
			return func(vm *exec.VirtualMachine) int64 {
				frame := vm.GetCurrentFrame()
				data := frame.Locals[0]
				p.Output = append(p.Output, byte(data))

				return 0
			}

		default:
			panic("impossible state")
		}

	default:
		panic("impossible state")
	}
}

type ExecResult struct {
	Output   string        `json:"out"`
	GasUsed  uint64        `json:"gas"`
	ExecTime time.Duration `json:"exec_duration"`
}

func run(bin []byte) (*ExecResult, error) {
	p := &Process{}

	var cfg exec.VMConfig
	gp := &compiler.SimpleGasPolicy{GasPerInstruction: 1}
	vm, err := exec.NewVirtualMachine(bin, cfg, p, gp)
	if err != nil {
		return nil, err
	}

	mainFunc, ok := vm.GetFunctionExport("h")
	if !ok {
		return nil, errors.New("impossible state: no h function exposed")
	}

	t0 := time.Now()
	_, err = vm.Run(mainFunc)
	if err != nil {
		return nil, err
	}

	return &ExecResult{
		Output:   string(p.Output),
		GasUsed:  vm.Gas,
		ExecTime: time.Since(t0),
	}, nil
}
