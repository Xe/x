# Git Workflow

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

## AI Attribution

AI-generated commits must include these footers:

```text
Assisted-by: [Model Name] via [Tool Name]
Reviewbot-request: yes
```

## Pull Requests

- Include a clear description of changes
- Reference related issues
- Pass CI (`npm test`)
- Add screenshots for UI changes when helpful
- Comment `/reviewbot` to trigger automated code review
