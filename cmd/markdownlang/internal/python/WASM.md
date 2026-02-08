# Python WebAssembly Binary

This directory contains `python.wasm`, a WebAssembly build of Python that runs in the wazero runtime.

## Obtaining python.wasm

The `python.wasm` file is copied from `llm/codeinterpreter/python/python.wasm` in this repository.

To rebuild or obtain a new version:

1. **From the existing location:**

   ```bash
   cp llm/codeinterpreter/python/python.wasm cmd/markdownlang/internal/python/python.wasm
   ```

2. **Building from source (if needed):**
   The python.wasm file can be built using the official Python WASI SDK.
   See: https://github.com/python/cpython/tree/main/Tools/wasm

## File Details

- **Size:** ~26 MB
- **Format:** WebAssembly (WASM)
- **Runtime:** wazero
- **Python Version:** Compatible with CPython built for WASI

## Why WebAssembly?

Using Python in WebAssembly provides:

- **Sandboxing:** True isolation from the host system
- **Security:** No filesystem access beyond mounted directories
- **Portability:** Same behavior across all platforms
- **Resource Control:** Memory and execution time limits

## Updating python.wasm

If you need to update the Python wasm binary:

1. Build or obtain a new WASM Python build
2. Test it thoroughly with the test suite
3. Replace the existing file
4. Run `go test ./cmd/markdownlang/internal/python/...` to verify compatibility

## Notes

- The wasm module is embedded in the Go binary using `//go:embed`
- Changes to python.wasm require rebuilding the Go package
- The init() function compiles the wasm module at package load time
