# x

Go monorepo for within.website services. JS tooling via npm.

## Critical Rules

- Always ask the user for intent before writing code.
- `internal.HandleStartup()` calls `flag.Parse()` — binaries must **not** call `flag.Parse()` themselves.
- All git commits require `--signoff`.

## Guidelines

- [Code Style](.agents/code-style.md) -- `xe-go-style` and `go-table-driven-tests` skills.
- [Git Workflow](.agents/git-workflow.md) -- `conventional-commits` skill
- [Project Info](.agents/project-info.md)
