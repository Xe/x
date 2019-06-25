package main

import (
	"errors"
	"time"

	"github.com/perlin-network/life/compiler"
	"github.com/perlin-network/life/exec"
)

type Process struct {
	Source CompiledProgram
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
	Output   string
	GasUsed  uint64
	ExecTime time.Duration
}

func run(cp CompiledProgram) (*ExecResult, error) {
	p := &Process{
		Source: cp,
	}

	var cfg exec.VMConfig
	gp := &compiler.SimpleGasPolicy{GasPerInstruction: 1}
	vm, err := exec.NewVirtualMachine(cp.Binary, cfg, p, gp)
	if err != nil {
		return nil, err
	}

	mainFunc, ok := vm.GetFunctionExport("main")
	if !ok {
		return nil, errors.New("impossible state: no main function exposed")
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
