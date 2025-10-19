# Repository Guidelines

## Project Structure & Module Organization

```text
├─ cmd/            # Main applications (each sub‑directory is a binary)
├─ internal/       # Private packages used by the repo
├─ web/            # Web‑related services and helpers
├─ writer/         # Utility libraries for writing output
├─ docs/           # Documentation assets
├─ test files      # Go test files live alongside source (`*_test.go`)
├─ package.json    # npm scripts and JS tooling
└─ go.mod          # Go module definition
```

Source code is primarily Go; JavaScript tooling lives under `node_modules` and the root `package.json`.

## Build, Test & Development Commands

| Command                      | Description                                         |
| ---------------------------- | --------------------------------------------------- |
| `npm run generate`           | Regenerates protobuf, Go code, and runs formatters. |
| `npm test` or `npm run test` | Runs `generate` then executes `go test ./...`.      |
| `go build ./...`             | Compiles all Go packages.                           |
| `go run ./cmd/<app>`         | Runs a specific binary from `cmd/`.                 |
| `npm run format`             | Formats Go (`goimports`) and JS/HTML (`prettier`).  |

## Coding Style & Naming Conventions

- **Go** – use `go fmt`/`goimports`; tabs for indentation, `camelCase` for variables, `PascalCase` for exported identifiers.
- **JavaScript/HTML/CSS** – formatted with Prettier (2‑space tabs, trailing commas, double quotes).
- Files are snake_case; packages use lower‑case module names.
- Run `npm run format` before committing.

## Testing Guidelines

- Tests are written in Go using the standard `testing` package (`*_test.go`).
- Keep test files next to the code they cover.
- Run the full suite with `npm test`.
- Aim for high coverage on new modules; existing coverage is not enforced.

## Commit & Pull Request Guidelines

- Commit messages follow **Conventional Commits** (e.g., `feat: add X`, `fix: resolve Y`).
- Use the provided `commitlint` rules – body and footer lines may be long, but the header must be concise.
- Pull requests must:
  - Include a clear description of changes.
  - Reference any related issues.
  - Pass CI (`npm test`).
  - Optionally add screenshots for UI changes.

## Security & Configuration Tips

- Secrets never belong in the repo; use environment variables or the `secrets` directory (ignored by Git).
- Run `npm audit` periodically and address reported vulnerabilities.

---

## Copilot Instructions

When undertaking a task, pause and ask the user for intent before writing code.

### Code Formatting

- **JavaScript** – double quotes, two‑space indentation.
- **Go** – follow the standard library style; prefer table‑driven tests.
- Run `npm run format` to apply Prettier formatting.

### Commit Message Format

Use Conventional Commits:

```text
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`.
Add `!` after type/scope for breaking changes or include `BREAKING CHANGE:` in the footer.

When committing, wrap the message in double quotes, sign‑off as:

```text
Signed-off-by: Mimi Yasomi <mimi@xeserv.us>
```

Write the commit to a temporary file first, using `printf` (fish shell).

### Additional Guidelines

- Keep descriptions concise, imperative, lowercase, and without a trailing period.
- Reference issues/PRs in the footer when applicable.

---

_This file is consulted by the repository’s tooling. Keep it up‑to‑date as the project evolves._
