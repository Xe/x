# x

Go monorepo for within.website services. JS tooling via npm.

## Critical Rules

- Always ask the user for intent before writing code.
- `internal.HandleStartup()` calls `flag.Parse()` — binaries must **not** call `flag.Parse()` themselves.
- All git commits require `--signoff`.

## Commit Messages

Follow **Conventional Commits**:

```text
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Types:** `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`

- Add `!` after type/scope for breaking changes or include `BREAKING CHANGE:` in the footer.
- Descriptions: concise, imperative, lowercase, no trailing period.
- Reference issues/PRs in the footer when applicable.
- All commits require `--signoff`.

## Guidelines

- [Code Style](.claude/code-style.md)
- [Git Workflow](.claude/git-workflow.md)
- [Project Info](.claude/project-info.md)
