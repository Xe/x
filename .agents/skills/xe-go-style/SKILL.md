---
name: xe-go-style
description: Write Go code following the conventions and patterns used in the within.website/x repository, including CLI patterns with internal.HandleStartup(), error handling, logging with slog, HTTP handlers, and testing.
---

# Xe/x Go Style Guide

Write Go code following the conventions and patterns used in the `within.website/x` repository (Go 1.25.4).

## Project Structure

```
├── cmd/            # Main applications (each subdirectory is a binary)
│   ├── x/          # Main CLI with subcommands
│   │   ├── main.go
│   │   └── cmd/    # Subcommand packages
│   └── sakurajima/ # Service binaries
│       ├── main.go
│       └── internal/  # Command-specific internal packages
├── internal/       # Private packages shared across commands
├── web/            # Web-related services and API clients
├── writer/         # Utility libraries for io.Writer middleware
└── gen/            # Generated code (protobuf)
```

Package naming: lowercase, single words preferred (slog, flagenv, kahless).

## CLI Pattern (CRITICAL)

All command-line tools MUST call `internal.HandleStartup()` at the start of `main()`. This function handles configuration loading and calls `flag.Parse()`. Individual binaries MUST NOT call `flag.Parse()` themselves.

```go
package main

import (
    "flag"
    "within.website/x/internal"
)

var (
    bind  = flag.String("bind", ":8080", "HTTP bind address")
    dbLoc = flag.String("db-loc", "", "Database location")
)

func main() {
    internal.HandleStartup()  // NO flag.Parse() call!
    // flags are now parsed, use *bind and *dbLoc here
}
```

Configuration loading order: flags → env vars (flagenv) → /run/secrets (flagfolder) → config file (flagconfyg) → flags (final override).

### Subcommands

Use `github.com/google/subcommands` for subcommand-based CLIs:

```go
package main

import (
    "context"
    "flag"
    "os"

    "github.com/google/subcommands"
    "within.website/x/internal"
)

func main() {
    internal.HandleStartup()

    subcommands.Register(subcommands.HelpCommand(), "")
    subcommands.Register(subcommands.FlagsCommand(), "")
    subcommands.Register(subcommands.CommandsCommand(), "")
    subcommands.Register(&serveCmd{}, "")
    subcommands.Register(&versionCmd{}, "")

    os.Exit(int(subcommands.Execute(context.Background())))
}

type serveCmd struct {
    dbLoc string
}

func (c *serveCmd) Name() string     { return "serve" }
func (c *serveCmd) Synopsis() string { return "Start the server" }
func (c *serveCmd) Usage() string {
    return "serve [flags]\nStart the HTTP server.\n"
}

func (c *serveCmd) SetFlags(f *flag.FlagSet) {
    f.StringVar(&c.dbLoc, "db-loc", "", "Database location")
}

func (c *serveCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
    // Implementation
    return subcommands.ExitSuccess
}
```

Flag names use kebab-case: `--db-loc`, `--grpc-bind`, `--slog-level`.

## Error Handling

Define package-level sentinel errors:

```go
var ErrNotFound = errors.New("store: key not found")
var ErrCantDecode = errors.New("store: can't decode value")
```

Wrap errors with context using `%w`:

```go
if err != nil {
    return nil, fmt.Errorf("failed to connect to database: %w", err)
}
return nil, fmt.Errorf("ollama: error encoding request: %w", err)
```

Combine multiple validation errors with `errors.Join`:

```go
func (t *Toplevel) Valid() error {
    var errs []error
    if err := t.Bind.Valid(); err != nil {
        errs = append(errs, fmt.Errorf("invalid bind block:\n%w", err))
    }
    if len(errs) != 0 {
        return fmt.Errorf("invalid configuration file:\n%w", errors.Join(errs...))
    }
    return nil
}
```

Check errors with `errors.Is()` and `errors.As()`:

```go
if errors.Is(err, ErrNotFound) {
    // handle not found
}
```

Custom error types implement `Error()` and optionally `slog.LogValue()`:

```go
type Error struct {
    WantStatus, GotStatus int
    URL                   *url.URL
    Method                string
    ResponseBody          string
}

func (e Error) Error() string {
    return fmt.Sprintf("%s %s: wanted status code %d, got: %d: %v",
        e.Method, e.URL, e.WantStatus, e.GotStatus, e.ResponseBody)
}

func (e Error) LogValue() slog.Value {
    return slog.GroupValue(
        slog.Int("want_status", e.WantStatus),
        slog.Int("got_status", e.GotStatus),
        slog.String("url", e.URL.String()),
        slog.String("method", e.Method),
        slog.String("body", e.ResponseBody),
    )
}
```

## Logging

Use `log/slog` (never the plain `log` package). Output JSON to stderr. Use `-slog-level` flag for runtime control.

```go
slog.Info("starting up",
    "bind", *bind,
    "db-loc", *dbLoc,
)
slog.Error("failed to create dao", "err", err)
slog.Debug("processing", "count", len(items))
```

Always use `"err"` as the key for errors. Implement `LogValue()` for complex types:

```go
func (s *Show) LogValue() slog.Value {
    return slog.GroupValue(
        slog.String("title", s.GetTitle()),
        slog.String("disk_path", s.GetDiskPath()),
    )
}
```

## HTTP Patterns

Middleware pattern:

```go
func PasswordMiddleware(username, password string, next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        user, pass, ok := r.BasicAuth()
        if !ok || user != username || pass != password {
            http.Error(w, "Unauthorized", http.StatusUnauthorized)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

Chaining middleware:

```go
var h http.Handler = topLevel
h = xffMW.Handler(h)
h = cors.Default().Handler(h)
h = FlyRegionAnnotation(h)
```

HTTP client with context and error handling:

```go
func (c *Client) doRequest(ctx context.Context, method, path string, wantCode int, body io.Reader) (*http.Response, error) {
    req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    resp, err := c.http.Do(req)
    if err != nil {
        return nil, fmt.Errorf("request failed: %w", err)
    }

    if resp.StatusCode != wantCode {
        return nil, web.NewError(wantCode, resp)
    }
    return resp, nil
}
```

Server lifecycle with errgroup:

```go
g, gCtx := errgroup.WithContext(ctx)

g.Go(func() error {
    ln, err := net.Listen("tcp", cfg.Bind.HTTP)
    if err != nil {
        return fmt.Errorf("failed to listen: %w", err)
    }
    return rtr.HandleHTTP(gCtx, ln)
})

return g.Wait()
```

## Testing

Tests co-located with source (`*_test.go`). Use table-driven tests:

```go
func TestDomainValid(t *testing.T) {
    t.Parallel()

    for _, tt := range []struct {
        name        string
        input       Domain
        err         error
        errContains string
    }{
        {name: "simple happy path", input: Domain{Name: "example.com"}},
        {name: "invalid domain", input: Domain{Name: "\uFFFD.com"}, err: ErrInvalidDomainName},
    } {
        t.Run(tt.name, func(t *testing.T) {
            err := tt.input.Valid()
            if tt.err != nil {
                if !errors.Is(err, tt.err) {
                    t.Logf("want: %v", tt.err)
                    t.Logf("got:  %v", err)
                    t.Error("got wrong error")
                }
            }
        })
    }
}
```

Testing best practices:

- Use `tt` as the loop variable in table-driven tests
- Always include `name` field for subtests
- Use `t.Helper()` for helper functions
- Use `t.Parallel()` for independent tests
- Use `t.TempDir()` for temporary directories
- Use `httptest.NewServer()` for HTTP mocking
- Use `errors.Is()` for error comparison
- Include `t.Logf("want: %v", x)` and `t.Logf("got: %v", y)` for debugging

```go
func loadConfig(t *testing.T, fname string) config.Toplevel {
    t.Helper()
    // ...
}
```

## Code Style

- Use `go fmt`/`goimports` for formatting
- Tabs for indentation
- `camelCase` for variables
- `PascalCase` for exported identifiers
- Files use snake_case
- Packages use lower-case module names
- Run `npm run format` before committing
