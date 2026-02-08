# Python Interpreter

WebAssembly-based Python interpreter for markdownlang. Runs Python code safely in a wazero sandbox with captured stdout/stderr.

## Why WebAssembly?

Running arbitrary Python code directly on your machine is a fantastic way to say goodbye to your security model. With wasm, we get:

- **True isolation** - No filesystem escape, no network access by default
- **Resource limits** - Memory limits, timeouts
- **Consistent behavior** - Same execution across platforms
- **Security** - The warm fuzzy feeling of not getting pwned by user code

## Usage

### Basic Execution

```go
import "within.website/x/cmd/markdownlang/internal/python"

ctx := context.Background()
result, err := python.Run(ctx, `print("Hello, World!")`)
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Stdout)
```

### Custom Configuration

```go
cfg := python.Config{
    Timeout:     10 * time.Second,
    MemoryLimit: 64 * 1024 * 1024, // 64 MB
    Stdin:       "input data\n",
}

result, err := python.RunWithConfig(ctx, code, cfg)
```

### Result Structure

```go
type Result struct {
    Stdout   string        // Standard output
    Stderr   string        // Standard error
    Error    string        // Execution errors (if any)
    Duration time.Duration // Execution time
}
```

## MCP Integration

The interpreter can be exposed as an MCP tool:

```go
import "within.website/x/cmd/markdownlang/internal/python"

// Add the tool to your MCP server
tool := python.Tool()
handler := python.Execute
```

The tool provides an instruction for the LLM:

```go
instruction := python.Instruction()
```

This instructs the LLM to prefer the python-interpreter for:

- Calculations
- Data processing
- Algorithms
- Mathematical operations
- JSON/data transformations

## Limitations

The WebAssembly Python runtime has some limitations:

- **No time.sleep()** - Not supported in wasm (OSError 58)
- **Sequential execution only** - Concurrent execution of the same module is not supported
- **No network access** - By default for security
- **Limited filesystem** - Only a temporary directory is mounted

## Supported Libraries

The interpreter supports standard Python libraries including:

- `math` - Mathematical functions
- `json` - JSON serialization/deserialization
- `re` - Regular expressions
- `datetime` - Date and time manipulation
- `collections` - Specialized container datatypes
- `itertools` - Functions creating iterators
- And most other standard library modules

## Performance

Benchmark results (Apple M3 Max):

- ~35-37ms per execution (simple code)
- ~67ms per execution (complex code)
- Includes wasm runtime overhead
- Suitable for interactive use

## Security

The interpreter runs in a WebAssembly sandbox with:

- No network access
- Limited filesystem access
- Optional timeout and memory limits
- Process isolation via wazero

This makes it safe for executing untrusted Python code.

## Testing

The interpreter has comprehensive test coverage:

- **Coverage:** 87.3% of statements
- **Test files:** 7 (5 test files + 2 source files)
- **Test cases:** 126+ individual tests
- **Benchmarks:** 2 performance benchmarks

Test coverage includes:

- Error handling (NameError, TypeError, ValueError, etc.)
- Edge cases (empty code, unicode, special characters)
- Output capture (stdout, stderr)
- Configuration options (timeout, memory limits, stdin)
- MCP integration (Execute, ExecuteWithConfig, Tool, Instruction)
- Sequential execution patterns

## Python WASM Binary

The `python.wasm` file (26MB) is embedded in the Go binary using `//go:embed`. See `WASM.md` for details on obtaining or rebuilding the WebAssembly Python runtime.

## Examples

See `example_test.go` for usage examples including:

- Basic execution
- Mathematical calculations
- JSON manipulation
- Data processing
- Error handling
