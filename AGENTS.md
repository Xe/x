# Repository Guidelines

A Go repository with JavaScript tooling using the standard `cmd/` binary structure.

## Quick Reference

| Command              | Description                                        |
| -------------------- | -------------------------------------------------- |
| `npm run generate`   | Regenerates protobuf, Go code, and runs formatters |
| `npm test`           | Runs generate then executes `go test ./...`        |
| `go build ./...`     | Compiles all Go packages                           |
| `go run ./cmd/<app>` | Runs a specific binary from `cmd/`                 |
| `npm run format`     | Formats Go (goimports) and JS/HTML (prettier)      |

## Detailed Guidelines

- [Go Style](.claude/go-style.md) - Formatting, naming conventions, code organization
- [Testing](.claude/testing.md) - Test framework, file placement, coverage guidelines
- [Git Workflow](.claude/git-workflow.md) - Commit conventions, AI attribution, PR requirements
- [Security](.claude/security.md) - Secrets management, dependency security
- [AI Instructions](.claude/ai-instructions.md) - Task execution guidance for AI assistants

## Implementation Details

### Flag Parsing Convention

All command‑line tools invoke `internal.HandleStartup()` at the start of `main()`. `internal.HandleStartup()` already calls `flag.Parse()`, so individual binaries must **not** call `flag.Parse()` themselves. Doing so can cause flags to be parsed twice and lead to unexpected behavior.

_This file is consulted by the repository's tooling. Keep it up‑to‑date as the project evolves._
