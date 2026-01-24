# Go Style Guidelines

## Formatting

- Use `go fmt` or `goimports` for formatting
- Tabs for indentation
- `camelCase` for variables and private identifiers
- `PascalCase` for exported identifiers
- File names are `snake_case`
- Package names use lower-case module names

## Code Organization

- Source code primarily lives in `cmd/` (binaries) and `internal/` (private packages)
- Test files live alongside source code as `*_test.go`
- Follow the standard library's code style patterns

## Running Formatters

Use `npm run format` to apply formatting to all files (Go with `goimports`, JS/HTML/CSS with Prettier).
