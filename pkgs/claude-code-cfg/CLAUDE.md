# Contribution Guidelines

## Code Quality & Security

### Commit Guidelines

Commit messages follow **Conventional Commits** format:

```text
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`

- Add `!` after type/scope for breaking changes or include `BREAKING CHANGE:` in the footer
- Keep descriptions concise, imperative, lowercase, and without a trailing period
- Reference issues/PRs in the footer when applicable

### Attribution Requirements

AI agents must disclose what tool and model they are using in the "Assisted-by" commit footer:

```text
Assisted-by: [Model Name] via [Tool Name]
```

Example:

```text
Assisted-by: GLM 4.6 via Claude Code
```

### Additional Guidelines

## Pull Request Requirements

- Include a clear description of changes
- Reference any related issues
- Pass CI (`npm test`)
- Optionally add screenshots for UI changes

### Security Best Practices

- Secrets never belong in the repo; use environment variables or the `secrets` directory (ignored by Git)
- Run `npm audit` periodically and address reported vulnerabilities
