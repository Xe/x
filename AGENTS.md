# x

Go monorepo for within.website services. JS tooling via npm.

## Critical Rules

- Always ask the user for intent before writing code.
- `internal.HandleStartup()` calls `flag.Parse()` — binaries must **not** call `flag.Parse()` themselves.
- All git commits require `--signoff`.

## Commit Messages

Follow **Conventional Commits** — see the `conventional-commits` skill.

## Guidelines

- [Code Style](.claude/code-style.md)
- [Git Workflow](.claude/git-workflow.md)
- [Project Info](.claude/project-info.md)
