---
name: go-table-driven-tests
description: Write Go table-driven tests following Go community best practices and this repository's conventions. Use when writing or refactoring Go tests, especially when you notice repeated test patterns or copy-pasted test code.
---

# Go Table-Driven Tests

## Overview

Table-driven tests are a Go testing idiom that reduces code duplication and makes tests more maintainable. Instead of writing separate test functions for each case, you define a table of test cases and iterate over it.

## When to Use Table-Driven Tests

Use table-driven tests when:

- You find yourself copying and pasting test code
- You're testing the same function/behavior with multiple inputs
- You want to add more test cases without writing more test functions
- Edge cases and boundary conditions need systematic coverage

**Do NOT use for**: Completely unrelated test scenarios, or when each test requires substantially different setup/teardown logic.

## Basic Template (Slice Pattern)

This is the most common pattern in this codebase:

```go
func TestFunctionName(t *testing.T) {
    cases := []struct {
        name  string
        input string
        want  string
        err   error
    }{
        {
            name:  "simple case",
            input: "a/b/c",
            want:  "a,b,c",
        },
        {
            name:  "empty input",
            input: "",
            want:  "",
        },
        {
            name:  "invalid input",
            input: "!!!",
            want:  "",
            err:   ErrInvalid,
        },
    }

    for _, tt := range cases {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            if !errors.Is(err, tt.err) {
                t.Errorf("FunctionName(%q) error = %v, want %v", tt.input, err, tt.err)
            }
            if got != tt.want {
                t.Errorf("FunctionName(%q) = %q, want %q", tt.input, got, tt.want)
            }
        })
    }
}
```

## Map Pattern (For Non-Deterministic Test Ordering)

Use a map when you want to ensure test independence:

```go
func TestFunctionName(t *testing.T) {
    tests := map[string]struct {
        input string
        want  string
    }{
        "simple case":   {input: "a/b/c", want: "a,b,c"},
        "empty input":   {input: "", want: ""},
        "trailing sep":  {input: "a/b/c/", want: "a,b,c"},
    }

    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            got := FunctionName(tc.input)
            if got != tc.want {
                t.Fatalf("%s: expected %q, got %q", name, tc.want, got)
            }
        })
    }
}
```

## Repository-Specific Conventions

### Variable Naming

This codebase consistently uses these variable names:

| Purpose          | Variable Name        | Example                    |
| ---------------- | -------------------- | -------------------------- |
| Test cases slice | `cases`, `tt`, `cs`  | `cases := []struct{...}`   |
| Loop variable    | `tt`, `cs`, `tc`     | `for _, tt := range cases` |
| Input field      | `input`, `in`, `inp` | `input: "test"`            |
| Expected output  | `want`, `expected`   | `want: "result"`           |
| Actual output    | `got`, `output`      | `got := Function()`        |
| Error field      | `err`, `wantErr`     | `err: ErrInvalid`          |
| Name field       | `name`               | `name: "descriptive name"` |

### Struct Field Guidelines

```go
// Always use named fields (not anonymous structs in this codebase)
cases := []struct {
    name      string  // Descriptive test name (REQUIRED when using slice pattern)
    input     Type    // Input to function under test
    want      Type    // Expected output
    err       error   // Expected error (use errors.Is for comparison)
    wantErr   bool    // Alternative: true if error is expected
    precondition func(*testing.T)  // Optional setup function
}{ ... }
```

### Error Reporting

This repository uses these patterns:

```go
// For error checking (preferred)
if !errors.Is(err, tt.err) {
    t.Errorf("error = %v, want %v", err, tt.err)
}

// For simple comparisons
if got != tt.want {
    t.Errorf("got %q, want %q", got, tt.want)
}

// Use t.Fatalf only when continuing doesn't make sense
// Use t.Errorf to see all test failures before stopping
```

## Advanced Patterns

### With Precondition Functions

When tests need specific setup:

```go
for _, tt := range []struct {
    name         string
    precondition func(*testing.T)
    input        string
    err          error
}{
    {
        name: "with listener",
        precondition: func(t *testing.T) {
            ln, err := net.Listen("tcp", ":8081")
            if err != nil {
                t.Fatal(err)
            }
            t.Cleanup(func() { ln.Close() })
        },
        input: "test",
        err:   nil,
    },
} {
    t.Run(tt.name, func(t *testing.T) {
        if tt.precondition != nil {
            tt.precondition(t)
        }
        // test logic
    })
}
```

### Parallel Tests

For independent tests that can run in parallel:

```go
func TestFunctionName(t *testing.T) {
    cases := []struct {
        name  string
        input string
        want  string
    }{
        // ... test cases
    }

    for _, tt := range cases {
        tt := tt // Capture range variable (Go < 1.22)
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel() // Marks this subtest as parallel

            got := FunctionName(tt.input)
            if got != tt.want {
                t.Errorf("got %q, want %q", got, tt.want)
            }
        })
    }
}
```

## Best Practices

1. **Always use `t.Run()` for subtests** - This is 100% consistent in this codebase
2. **Use descriptive test names** - The `name` field should clearly describe what is being tested
3. **Test one thing per case** - Each table entry should test one specific behavior
4. **Include edge cases** - Empty strings, nil values, maximum values, etc.
5. **Use `errors.Is` for error comparison** - Not `==` or `reflect.DeepEqual`
6. **Prefer `t.Errorf` over `t.Fatalf`** - See all failures before stopping
7. **Keep test data inline** - External files only for large golden test sets

## Common Pitfalls to Avoid

1. **Forgetting `t.Run()`** - Without subtests, all failures appear at the same line
2. **Using `Fatalf` immediately** - You won't see other test failures
3. **Not capturing range variable** - In Go < 1.22, add `tt := tt` before `t.Run`
4. **Anonymous structs** - This codebase prefers named structs for clarity
5. **Inconsistent naming** - Stick to the conventions (`cases`, `tt`, `want`, `got`)

## Comparison with Traditional Tests

| Traditional                          | Table-Driven                                                 |
| ------------------------------------ | ------------------------------------------------------------ |
| `func TestFoo(t *testing.T) { ... }` | `func TestFoo(t *testing.T) { cases := []struct{...}{...} }` |
| One test function per case           | Single function, many cases                                  |
| Hard to add new cases                | Just add a row to the table                                  |
| Verbose boilerplate                  | Concise, DRY code                                            |
| `go test -run TestFoo_SpecificCase`  | `go test -run TestFoo/name`                                  |

## Running Specific Tests

```bash
# Run all tests in a function
go test -run TestFunctionName

# Run a specific subtest
go test -run TestFunctionName/descriptive_name

# Run all tests matching a pattern
go test -run TestFunctionName/.*/empty

# Verbose mode (see all test output)
go test -v

# Run with race detector
go test -race
```

## References

- [Go Wiki: TableDrivenTests](https://go.dev/wiki/TableDrivenTests)
- [Go Testing Package](https://pkg.go.dev/testing/)
- [Prefer Table Driven Tests - Dave Cheney](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
