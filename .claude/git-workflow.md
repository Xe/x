# Git Workflow

## Commit Messages

Follow **Conventional Commits** format:

```text
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

**Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `build`, `ci`, `chore`, `revert`

### Rules

- Add `!` after type/scope for breaking changes, or include `BREAKING CHANGE:` in the footer
- Keep descriptions concise, imperative, lowercase, and without a trailing period
- Reference issues/PRs in the footer when applicable
- **ALL git commits MUST be made with `--signoff`. This is mandatory.**

## AI Attribution

AI agents must disclose their tool and model in the commit footer:

```text
Assisted-by: [Model Name] via [Tool Name]
```

Example: `Assisted-by: GLM 4.6 via Claude Code`

Additionally, AI-created commits should include `Reviewbot-request: yes` to trigger automated code review.

## Pull Requests

- Include a clear description of changes
- Reference any related issues
- Pass CI (`npm test`)
- Optionally add screenshots for UI changes
- Add a comment `/reviewbot` to trigger automated code review
