//go:build !wasip1

package wasm

import (
	"context"
	"fmt"

	"github.com/tetratelabs/wazero/api"
)

func (value String) Load(module api.Module) string {
	return string(value.LoadBytes(module))
}

func (value String) LoadBytes(module api.Module) []byte {
	data, ok := module.Memory().Read(uint32(value>>32), uint32(value))
	if !ok {
		panic("memory read out of bounds")
	}
	return data
}

func (value String) Store(ctx context.Context, module api.Module, data string) (String, error) {
	buffer, err := Store(ctx, module, []byte(data))
	if err != nil {
		return 0, err
	}

	return String(buffer), nil
}

func (buffer Buffer) Load(module api.Module) []byte {
	data, ok := module.Memory().Read(buffer.Address(), buffer.Length())
	if !ok {
		panic("memory read out of bounds")
	}
	return data
}

func Store(ctx context.Context, module api.Module, data []byte) (Buffer, error) {
	results, err := module.ExportedFunction("malloc").Call(ctx, uint64(len(data)))
	if err != nil {
		return 0, fmt.Errorf("wasm: failed to call malloc: %w", err)
	}

	buffer := Buffer(results[0])
	module.Memory().Write(buffer.Address(), data)
	return buffer, nil
}
