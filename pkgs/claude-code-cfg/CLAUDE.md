# Contribution Guidelines

## Code Quality & Security

### Attribution Requirements

AI agents must disclose what tool and model they are using in the "Assisted-by" commit footer:

```text
Assisted-by: [Model Name] via [Tool Name]
```

Example:

```text
Assisted-by: GLM 4.6 via Claude Code
```

### Security Best Practices

- Secrets never belong in the repo; use environment variables or the `secrets` directory (ignored by Git)
