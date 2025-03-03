//go:build !wasip1

package wasm

func Malloc(size uint32) Buffer {
	panic("don't call this if you're not using WASI")
}
