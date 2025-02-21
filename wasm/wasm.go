package wasm

import (
	"unsafe"

	"github.com/tetratelabs/wazero/api"
)

type String uint64

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

func FromString(value string) String {
	position := uint32(uintptr(unsafe.Pointer(unsafe.StringData(value))))
	bytes := uint32(len(value))
	return String(uint64(position)<<32 | uint64(bytes))
}

type Buffer uint64

func (buffer Buffer) Address() uint32 {
	return uint32(buffer >> 32)
}

func (buffer Buffer) Length() uint32 {
	return uint32((buffer << 32) >> 32)
}

func FromSlice(value []byte) Buffer {
	if len(value) == 0 {
		return 0
	}
	ptr := uint64(uintptr(unsafe.Pointer(&value[0])))
	return Buffer(ptr<<32 | uint64(len(value)))
}

func (buffer Buffer) Slice() []byte {
	return unsafe.Slice((*byte)(unsafe.Pointer(uintptr(buffer.Address()))), buffer.Length())
}

func (buffer Buffer) String() string {
	return unsafe.String((*byte)(unsafe.Pointer(uintptr(buffer.Address()))), buffer.Length())
}
