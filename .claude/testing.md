# Testing Guidelines

## Framework

- Tests are written in Go using the standard `testing` package
- Test files are named `*_test.go`
- Place test files next to the code they cover

## Running Tests

- Run the full suite with `npm test` (this runs `generate` first, then `go test ./...`)
- `npm run generate` regenerates protobuf and Go code before tests

## Coverage

- Aim for high coverage on new modules
- Existing coverage is not enforced

## Patterns

- Prefer table-driven tests for multiple test cases
