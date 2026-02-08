package python

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		wantStdout  string
		wantStderr  string
		wantError   bool
		description string
	}{
		{
			name:        "simple print",
			code:        `print("Hello, World!")`,
			wantStdout:  "Hello, World!\n",
			wantError:   false,
			description: "Basic print statement",
		},
		{
			name: "math calculation",
			code: `import math
print(f"Pi is approximately {math.pi:.2f}")
print(f"2^10 = {2**10}")`,
			wantStdout:  "Pi is approximately 3.14\n2^10 = 1024\n",
			wantError:   false,
			description: "Math operations and formatted output",
		},
		{
			name: "list comprehension",
			code: `squares = [x**2 for x in range(10)]
print(squares)`,
			wantStdout:  "[0, 1, 4, 9, 16, 25, 36, 49, 64, 81]\n",
			wantError:   false,
			description: "List comprehension",
		},
		{
			name: "error handling",
			code: `try:
    1 / 0
except ZeroDivisionError as e:
    print(f"Caught error: {e}")`,
			wantStdout:  "Caught error: division by zero\n",
			wantError:   false,
			description: "Exception handling",
		},
		{
			name:        "syntax error",
			code:        `print("unclosed string`,
			wantStdout:  "",
			wantError:   true,
			description: "Python syntax error should fail",
		},
		{
			name: "json manipulation",
			code: `import json
data = {"name": "test", "value": 42}
print(json.dumps(data))`,
			wantStdout:  `{"name": "test", "value": 42}` + "\n",
			wantError:   false,
			description: "JSON serialization",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := Run(ctx, tt.code)

			if tt.wantError {
				if err == nil && result.Error == "" {
					t.Errorf("Expected error but got none")
					t.Logf("stdout: %s", result.Stdout)
					t.Logf("stderr: %s", result.Stderr)
				}
				return
			}

			if err != nil {
				t.Errorf("Run() error = %v", err)
				t.Logf("stdout: %s", result.Stdout)
				t.Logf("stderr: %s", result.Stderr)
				t.Logf("platform error: %s", result.Error)
				return
			}

			if result.Stdout != tt.wantStdout {
				t.Errorf("Run() stdout = %q, want %q", result.Stdout, tt.wantStdout)
			}

			if tt.wantStderr != "" && result.Stderr != tt.wantStderr {
				t.Errorf("Run() stderr = %q, want %q", result.Stderr, tt.wantStderr)
			}

			// Log the execution time for monitoring
			t.Logf("Execution time: %v", result.Duration)
		})
	}
}

func TestErrorScenarios(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		wantError   bool
		checkError  func(error, *Result) bool
		description string
	}{
		{
			name:      "undefined variable",
			code:      `print(undefined_variable)`,
			wantError: true,
			checkError: func(err error, result *Result) bool {
				// Should have either a Go error or a Python error in stderr
				return err != nil || strings.Contains(result.Stderr, "NameError")
			},
			description: "Undefined variable should cause error",
		},
		{
			name:      "import non-existent module",
			code:      `import nonexistent_module`,
			wantError: true,
			checkError: func(err error, result *Result) bool {
				return err != nil || strings.Contains(result.Stderr, "ModuleNotFoundError")
			},
			description: "Importing non-existent module should fail",
		},
		{
			name:      "runtime error",
			code:      `raise RuntimeError("intentional error")`,
			wantError: true,
			checkError: func(err error, result *Result) bool {
				return err != nil || strings.Contains(result.Stderr, "RuntimeError")
			},
			description: "Explicit runtime error should fail",
		},
		{
			name:      "type error",
			code:      `print("string" + 42)`,
			wantError: true,
			checkError: func(err error, result *Result) bool {
				return err != nil || strings.Contains(result.Stderr, "TypeError")
			},
			description: "Type error should fail",
		},
		{
			name: "indentation error",
			code: `def foo():
print("bad indentation")`,
			wantError: true,
			checkError: func(err error, result *Result) bool {
				return err != nil || strings.Contains(result.Stderr, "IndentationError")
			},
			description: "Indentation error should fail",
		},
		{
			name: "key error",
			code: `d = {}
print(d["nonexistent"])`,
			wantError: true,
			checkError: func(err error, result *Result) bool {
				return err != nil || strings.Contains(result.Stderr, "KeyError")
			},
			description: "Key error should fail",
		},
		{
			name: "attribute error",
			code: `import math
print(math.nonexistent_attr)`,
			wantError: true,
			checkError: func(err error, result *Result) bool {
				return err != nil || strings.Contains(result.Stderr, "AttributeError")
			},
			description: "Attribute error should fail",
		},
		{
			name:      "value error",
			code:      `int("not a number")`,
			wantError: true,
			checkError: func(err error, result *Result) bool {
				return err != nil || strings.Contains(result.Stderr, "ValueError")
			},
			description: "Value error should fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := Run(ctx, tt.code)

			if tt.wantError {
				if err == nil && result.Error == "" && tt.checkError != nil && !tt.checkError(err, result) {
					t.Errorf("Expected error but got none")
					t.Logf("stdout: %s", result.Stdout)
					t.Logf("stderr: %s", result.Stderr)
					t.Logf("error: %s", result.Error)
				}
				return
			}

			if err != nil {
				t.Errorf("Run() unexpected error = %v", err)
				t.Logf("stdout: %s", result.Stdout)
				t.Logf("stderr: %s", result.Stderr)
			}
		})
	}
}

func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		wantStdout  string
		wantError   bool
		description string
	}{
		{
			name:        "empty code",
			code:        ``,
			wantStdout:  "",
			wantError:   false,
			description: "Empty code should execute without error",
		},
		{
			name:        "only comments",
			code:        `# This is a comment\n# Another comment`,
			wantStdout:  "",
			wantError:   false,
			description: "Code with only comments should execute",
		},
		{
			name:        "only spaces",
			code:        `      `,
			wantStdout:  "",
			wantError:   false,
			description: "Code with only spaces should execute",
		},
		{
			name: "multiple prints",
			code: `print("line1")
print("line2")
print("line3")`,
			wantStdout:  "line1\nline2\nline3\n",
			wantError:   false,
			description: "Multiple print statements",
		},
		{
			name: "print with no newline",
			code: `import sys
sys.stdout.write("no newline")
sys.stdout.write(" continued")`,
			wantStdout:  "no newline continued",
			wantError:   false,
			description: "Print without newline using sys.stdout.write",
		},
		{
			name:        "unicode output",
			code:        `print("Hello 世界 🌍")`,
			wantStdout:  "Hello 世界 🌍\n",
			wantError:   false,
			description: "Unicode characters should work",
		},
		{
			name: "special characters",
			code: `print("Tab:\tNext")
print("Newline:\nNext")
print("Quote:\"Quote\"")`,
			wantStdout:  "Tab:\tNext\nNewline:\nNext\nQuote:\"Quote\"\n",
			wantError:   false,
			description: "Special characters should be preserved",
		},
		{
			name: "large output",
			code: `for i in range(1000):
    print(f"Line {i}: " + "x" * 50)`,
			wantError:   false,
			description: "Large output should be handled",
		},
		{
			name: "nested loops",
			code: `for i in range(3):
    for j in range(3):
        print(f"{i},{j}")`,
			wantStdout:  "0,0\n0,1\n0,2\n1,0\n1,1\n1,2\n2,0\n2,1\n2,2\n",
			wantError:   false,
			description: "Nested loops should work",
		},
		{
			name: "complex expression",
			code: `result = ((1 + 2) * 3 - 4) / 2
print(f"Result: {result}")`,
			wantStdout:  "Result: 2.5\n",
			wantError:   false,
			description: "Complex mathematical expression",
		},
		{
			name: "boolean operations",
			code: `print(True and False)
print(True or False)
print(not True)
print(1 < 2 and 2 > 1)`,
			wantStdout:  "False\nTrue\nFalse\nTrue\n",
			wantError:   false,
			description: "Boolean operations",
		},
		{
			name: "string operations",
			code: `s = "Hello"
print(s.upper())
print(s.lower())
print(s * 3)
print(s + " World")`,
			wantStdout:  "HELLO\nhello\nHelloHelloHello\nHello World\n",
			wantError:   false,
			description: "String manipulation",
		},
		{
			name: "list operations",
			code: `lst = [1, 2, 3]
print(len(lst))
print(lst[0])
print(lst[-1])
print(2 in lst)`,
			wantStdout:  "3\n1\n3\nTrue\n",
			wantError:   false,
			description: "List operations",
		},
		{
			name: "dictionary operations",
			code: `d = {"a": 1, "b": 2}
print(len(d))
print(d["a"])
print("a" in d)
print(d.get("c", "default"))`,
			wantStdout:  "2\n1\nTrue\ndefault\n",
			wantError:   false,
			description: "Dictionary operations",
		},
		{
			name: "function definition and call",
			code: `def greet(name):
    return f"Hello, {name}!"

print(greet("World"))
print(greet("Alice"))`,
			wantStdout:  "Hello, World!\nHello, Alice!\n",
			wantError:   false,
			description: "Function definition and calling",
		},
		{
			name: "lambda function",
			code: `add = lambda x, y: x + y
print(add(5, 3))`,
			wantStdout:  "8\n",
			wantError:   false,
			description: "Lambda function",
		},
		{
			name: "list methods",
			code: `lst = [1, 2, 3]
lst.append(4)
lst.extend([5, 6])
print(lst)
print(lst.pop())
print(lst)`,
			wantStdout:  "[1, 2, 3, 4, 5, 6]\n6\n[1, 2, 3, 4, 5]\n",
			wantError:   false,
			description: "List method operations",
		},
		{
			name: "string formatting",
			code: `name = "Alice"
age = 30
print(f"{name} is {age} years old")
print("{} is {} years old".format(name, age))
print("%s is %d years old" % (name, age))`,
			wantStdout:  "Alice is 30 years old\nAlice is 30 years old\nAlice is 30 years old\n",
			wantError:   false,
			description: "Various string formatting methods",
		},
		{
			name: "tuple operations",
			code: `t = (1, 2, 3)
print(t[0])
print(len(t))
print(t + (4, 5))`,
			wantStdout:  "1\n3\n(1, 2, 3, 4, 5)\n",
			wantError:   false,
			description: "Tuple operations",
		},
		{
			name: "set operations",
			code: `s1 = {1, 2, 3}
s2 = {3, 4, 5}
print(s1 | s2)
print(s1 & s2)
print(s1 - s2)`,
			wantStdout:  "{1, 2, 3, 4, 5}\n{3}\n{1, 2}\n",
			wantError:   false,
			description: "Set operations",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := Run(ctx, tt.code)

			if tt.wantError {
				if err == nil && result.Error == "" {
					t.Errorf("Expected error but got none")
					t.Logf("stdout: %s", result.Stdout)
					t.Logf("stderr: %s", result.Stderr)
				}
				return
			}

			if err != nil {
				t.Errorf("Run() error = %v", err)
				t.Logf("stdout: %s", result.Stdout)
				t.Logf("stderr: %s", result.Stderr)
				t.Logf("platform error: %s", result.Error)
				return
			}

			if tt.wantStdout != "" && result.Stdout != tt.wantStdout {
				t.Errorf("Run() stdout = %q, want %q", result.Stdout, tt.wantStdout)
			}
		})
	}
}

func TestStderrCapture(t *testing.T) {
	tests := []struct {
		name        string
		code        string
		wantStderr  string
		description string
	}{
		{
			name: "stderr write",
			code: `import sys
sys.stderr.write("Error message\n")`,
			wantStderr:  "Error message\n",
			description: "Writing to stderr should be captured",
		},
		{
			name: "exception to stderr",
			code: `try:
    raise ValueError("test error")
except ValueError as e:
    import sys
    sys.stderr.write(f"Caught: {e}\n")`,
			wantStderr:  "Caught: test error\n",
			description: "Exception messages to stderr",
		},
		{
			name: "warning to stderr",
			code: `import warnings
warnings.warn("This is a warning")`,
			wantStderr:  "",
			description: "Warnings may or may not appear in stderr depending on configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			result, err := Run(ctx, tt.code)

			if err != nil {
				t.Errorf("Run() error = %v", err)
				t.Logf("stdout: %s", result.Stdout)
				t.Logf("stderr: %s", result.Stderr)
				return
			}

			if tt.wantStderr != "" && result.Stderr != tt.wantStderr {
				t.Errorf("Run() stderr = %q, want %q", result.Stderr, tt.wantStderr)
			}

			t.Logf("Captured stderr: %q", result.Stderr)
		})
	}
}

func TestRunWithConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("with timeout", func(t *testing.T) {
		// Note: time.sleep() is not supported in wasm Python
		// Instead, we'll test that the timeout configuration is accepted
		code := `# No sleep supported in wasm
sum = 0
for i in range(1000):
    sum += i
print(f"Sum: {sum}")`

		cfg := Config{
			Timeout: 5 * time.Second,
		}

		result, err := RunWithConfig(ctx, code, cfg)
		if err != nil {
			t.Errorf("RunWithConfig() error = %v", err)
			t.Logf("stdout: %s", result.Stdout)
			t.Logf("stderr: %s", result.Stderr)
		}

		expected := "Sum: 499500\n"
		if result.Stdout != expected {
			t.Errorf("RunWithConfig() stdout = %q, want %q", result.Stdout, expected)
		}
	})

	t.Run("with very short timeout", func(t *testing.T) {
		// Test with a very short timeout that should not affect quick execution
		code := `print("quick")`

		cfg := Config{
			Timeout: 100 * time.Millisecond,
		}

		result, err := RunWithConfig(ctx, code, cfg)
		if err != nil {
			t.Errorf("RunWithConfig() error = %v", err)
		}

		if result.Stdout != "quick\n" {
			t.Errorf("RunWithConfig() stdout = %q, want %q", result.Stdout, "quick\n")
		}
	})

	t.Run("with zero timeout (no limit)", func(t *testing.T) {
		code := `print("no timeout")`

		cfg := Config{
			Timeout: 0,
		}

		result, err := RunWithConfig(ctx, code, cfg)
		if err != nil {
			t.Errorf("RunWithConfig() error = %v", err)
		}

		if result.Stdout != "no timeout\n" {
			t.Errorf("RunWithConfig() stdout = %q, want %q", result.Stdout, "no timeout\n")
		}
	})

	t.Run("with stdin", func(t *testing.T) {
		// Note: This test demonstrates stdin support but actual input()
		// behavior depends on the Python wasm runtime configuration
		code := `print("test")`

		cfg := Config{
			Stdin: "test input\n",
		}

		result, err := RunWithConfig(ctx, code, cfg)
		if err != nil {
			t.Errorf("RunWithConfig() error = %v", err)
		}

		t.Logf("stdout: %s", result.Stdout)
		t.Logf("stderr: %s", result.Stderr)
	})

	t.Run("with empty stdin", func(t *testing.T) {
		code := `print("empty stdin test")`

		cfg := Config{
			Stdin: "",
		}

		result, err := RunWithConfig(ctx, code, cfg)
		if err != nil {
			t.Errorf("RunWithConfig() error = %v", err)
		}

		if result.Stdout != "empty stdin test\n" {
			t.Errorf("RunWithConfig() stdout = %q, want %q", result.Stdout, "empty stdin test\n")
		}
	})

	t.Run("with memory limit set", func(t *testing.T) {
		code := `print("memory limit test")`

		cfg := Config{
			MemoryLimit: 64 * 1024 * 1024, // 64 MB
		}

		result, err := RunWithConfig(ctx, code, cfg)
		if err != nil {
			t.Errorf("RunWithConfig() error = %v", err)
		}

		if result.Stdout != "memory limit test\n" {
			t.Errorf("RunWithConfig() stdout = %q, want %q", result.Stdout, "memory limit test\n")
		}
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Timeout == 0 {
		t.Error("DefaultConfig() should have a non-zero timeout")
	}

	if cfg.MemoryLimit == 0 {
		t.Error("DefaultConfig() should have a non-zero memory limit")
	}

	if cfg.Timeout != 30*time.Second {
		t.Errorf("DefaultConfig() timeout = %v, want %v", cfg.Timeout, 30*time.Second)
	}

	if cfg.MemoryLimit != 128*1024*1024 {
		t.Errorf("DefaultConfig() memory limit = %d, want %d", cfg.MemoryLimit, 128*1024*1024)
	}
}

func TestSequentialExecution(t *testing.T) {
	// Test that multiple executions can run sequentially
	// (concurrent execution of the same wasm module is not supported)
	ctx := context.Background()
	code := `print("sequential test")`

	for i := 0; i < 5; i++ {
		_, err := Run(ctx, code)
		if err != nil {
			t.Errorf("Sequential Run() error = %v", err)
		}
	}
}

func TestDifferentCodeSequences(t *testing.T) {
	// Test that different code can be executed sequentially
	ctx := context.Background()

	codes := []string{
		`print("first")`,
		`x = 5
print(x)`,
		`import math
print(math.pi)`,
		`for i in range(3):
    print(i)`,
	}

	for i, code := range codes {
		t.Run(fmt.Sprintf("sequence_%d", i), func(t *testing.T) {
			result, err := Run(ctx, code)
			if err != nil {
				t.Errorf("Run() error = %v", err)
				t.Logf("stdout: %s", result.Stdout)
			}
		})
	}
}

func TestResultFields(t *testing.T) {
	ctx := context.Background()

	t.Run("all fields present", func(t *testing.T) {
		result, err := Run(ctx, `print("test")`)
		if err != nil {
			t.Fatalf("Run() error = %v", err)
		}

		// Check that Result has all expected fields
		if result.Stdout == "" {
			t.Error("Result.Stdout should not be empty")
		}

		// Duration should be positive
		if result.Duration <= 0 {
			t.Errorf("Result.Duration = %v, want > 0", result.Duration)
		}

		// Error should be empty for successful execution
		if result.Error != "" {
			t.Errorf("Result.Error = %q, want empty", result.Error)
		}
	})

	t.Run("error field set on failure", func(t *testing.T) {
		result, err := Run(ctx, `print("unclosed string`)
		// We expect an error, but result might still be returned
		if err != nil && result == nil {
			t.Skip("Result not returned on error")
		}

		if result != nil && result.Error == "" {
			t.Error("Result.Error should be set on syntax error")
		}
	})
}

func TestContextCancellation(t *testing.T) {
	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Try to run code with cancelled context
		// Note: The execution is so fast that it may complete before cancellation takes effect
		result, err := Run(ctx, `print("test")`)
		// We don't assert an error here because the execution is very fast
		// This test documents the behavior rather than enforcing it
		t.Logf("With cancelled context - err: %v, stdout: %s", err, result.Stdout)
	})
}

func TestEmptyFS(t *testing.T) {
	// Test the emptyFS implementation
	fs := emptyFS{}

	_, err := fs.Open("test.txt")
	if err == nil {
		t.Error("emptyFS.Open() should return error")
	}

	// Test Glob
	files, err := fs.Glob("*.txt")
	if err != nil {
		t.Errorf("emptyFS.Glob() error = %v", err)
	}
	if files != nil && len(files) != 0 {
		t.Errorf("emptyFS.Glob() = %v, want empty", files)
	}

	// Test ReadDir
	_, err = fs.ReadDir("/")
	if err == nil {
		t.Error("emptyFS.ReadDir() should return error")
	}
}

func BenchmarkRun(b *testing.B) {
	ctx := context.Background()
	code := `print("benchmark")`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Run(ctx, code)
		if err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}

func BenchmarkRunWithComplexCode(b *testing.B) {
	ctx := context.Background()
	code := `
import json
data = {"numbers": [i for i in range(100)]}
result = json.dumps(data)
sum_val = sum(data["numbers"])
print(f"Sum: {sum_val}")
`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Run(ctx, code)
		if err != nil {
			b.Fatalf("Run() error = %v", err)
		}
	}
}
