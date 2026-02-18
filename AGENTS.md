# x

Go monorepo for within.website services. JS tooling via npm.

## Commands

| Command              | Description                   |
| -------------------- | ----------------------------- |
| `npm test`           | Generate + `go test ./...`    |
| `npm run format`     | goimports + Prettier          |
| `npm run generate`   | Protobuf, codegen, formatters |
| `go build ./...`     | Compile all packages          |
| `go run ./cmd/<app>` | Run a specific binary         |

## Critical Rules

- Always ask the user for intent before writing code.
- `internal.HandleStartup()` calls `flag.Parse()` — binaries must **not** call `flag.Parse()` themselves.
- All git commits require `--signoff`.

## Guidelines

- [Code Style](.claude/code-style.md)
- [Git Workflow](.claude/git-workflow.md)
- [Project Info](.claude/project-info.md)
