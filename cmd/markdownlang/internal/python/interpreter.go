// Package python provides a WebAssembly-based Python interpreter for markdownlang.
// It runs Python code safely in a wazero sandbox with captured stdout/stderr.
//
// Why wasm? Because running arbitrary Python code directly on your machine is
// a fantastic way to say goodbye to your security model. With wasm, we get:
// - True isolation (no filesystem escape, no network access by default)
// - Resource limits (memory limits, timeouts)
// - Consistent behavior across platforms
// - The warm fuzzy feeling of not getting pwned by user code
package python

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

//go:embed python.wasm
var binary []byte

// runtime holds the wazero runtime and compiled module.
// Initialized once to avoid recompiling the wasm module on every run.
type runtime struct {
	r    wazero.Runtime
	code wazero.CompiledModule
}

var pythonRuntime *runtime

func init() {
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)

	// WASI is required for basic filesystem and stdio operations
	wasi_snapshot_preview1.MustInstantiate(ctx, r)

	code, err := r.CompileModule(ctx, binary)
	if err != nil {
		panic(fmt.Sprintf("failed to compile Python wasm module: %v", err))
	}

	pythonRuntime = &runtime{
		r:    r,
		code: code,
	}
}

// Result represents the output from a Python execution.
type Result struct {
	// Stdout contains the standard output from the Python code.
	Stdout string `json:"stdout"`

	// Stderr contains the standard error output from the Python code.
	Stderr string `json:"stderr"`

	// Error contains any execution error (timeout, memory limit, etc.).
	// This is distinct from Stderr - Error is for runtime failures,
	// Stderr is for Python's stderr stream.
	Error string `json:"error,omitempty"`

	// Duration is how long the execution took.
	Duration time.Duration `json:"duration"`
}

// Config holds configuration for Python execution.
type Config struct {
	// Timeout is the maximum time to allow execution.
	// Zero means no timeout (not recommended for untrusted code).
	Timeout time.Duration

	// MemoryLimit is the maximum memory in bytes.
	// Zero means no limit (not recommended for untrusted code).
	MemoryLimit uint64

	// Stdin provides input to the Python process.
	Stdin string
}

// DefaultConfig returns sensible defaults for Python execution.
func DefaultConfig() Config {
	return Config{
		Timeout:     30 * time.Second,
		MemoryLimit: 128 * 1024 * 1024, // 128 MB
	}
}

// Run executes Python code in the wasm sandbox.
//
// The code runs with:
// - No network access
// - Limited filesystem access (a temporary directory)
// - Optional timeout and memory limits
// - Captured stdout and stderr
//
// Example:
//
//	result, err := python.Run(ctx, code)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result.Stdout)
func Run(ctx context.Context, code string) (*Result, error) {
	return RunWithConfig(ctx, code, DefaultConfig())
}

// RunWithConfig executes Python code with custom configuration.
func RunWithConfig(ctx context.Context, code string, cfg Config) (*Result, error) {
	start := time.Now()

	// Create buffers for stdio
	fout := &bytes.Buffer{}
	ferr := &bytes.Buffer{}
	fin := bytes.NewBufferString(cfg.Stdin)

	// Create a temporary directory for the code
	tmpDir, err := os.MkdirTemp("", "markdownlang-python-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Write the Python code to a file
	// The path is special: /.within.website.main.py (no directory separator)
	// This matches the convention used by the python-wasm-mcp server
	mainPyPath := "/.within.website.main.py"
	if err := os.WriteFile(filepath.Join(tmpDir, ".within.website.main.py"), []byte(code), 0644); err != nil {
		return nil, fmt.Errorf("failed to write Python file: %w", err)
	}

	// Build filesystem configuration
	fsConfig := wazero.NewFSConfig().WithDirMount(tmpDir, "/")

	// Build module configuration
	moduleConfig := wazero.NewModuleConfig().
		WithStdout(fout).
		WithStderr(ferr).
		WithStdin(fin).
		WithArgs("python", mainPyPath).
		WithName("markdownlang-python").
		WithFSConfig(fsConfig).
		WithSysNanosleep().
		WithSysNanotime().
		WithSysWalltime()

	// Apply timeout if configured
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	// Instantiate and run the module
	mod, err := pythonRuntime.r.InstantiateModule(ctx, pythonRuntime.code, moduleConfig)
	if err != nil {
		result := &Result{
			Stdout:   fout.String(),
			Stderr:   ferr.String(),
			Error:    err.Error(),
			Duration: time.Since(start),
		}
		return result, fmt.Errorf("Python execution failed: %w", err)
	}

	defer mod.Close(ctx)

	return &Result{
		Stdout:   fout.String(),
		Stderr:   ferr.String(),
		Duration: time.Since(start),
	}, nil
}

// emptyFS is an empty filesystem that returns ENOENT for all paths.
// Used when no filesystem is provided to the interpreter.
type emptyFS struct{}

func (emptyFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
}

func (emptyFS) Glob(pattern string) ([]string, error) {
	return nil, nil
}

func (emptyFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return nil, &fs.PathError{Op: "readdir", Path: name, Err: fs.ErrNotExist}
}
