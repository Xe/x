package wasm

//go:wasmexport malloc
func Malloc(size uint32) Buffer {
	return FromSlice(make([]byte, size))
}
