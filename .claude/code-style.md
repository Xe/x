# Code Style

## Go

- Format with `go fmt` / `goimports`
- Tabs for indentation
- `camelCase` for unexported, `PascalCase` for exported identifiers
- Follow standard library style
- Prefer table-driven tests using the `testing` package
- Test files (`*_test.go`) live alongside source

## JavaScript / HTML / CSS

- Format with Prettier (`npm run format`)
- Double quotes, two-space indentation, trailing commas

## File Naming

- Files: `snake_case`
- Go packages: lowercase

## Testing

- Run full suite: `npm test`
- Aim for high coverage on new modules; existing coverage is not enforced
