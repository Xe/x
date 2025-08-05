# Copilot Instructions

When undertaking a task, take a moment to pause and ask the user what the intent of the task is. Use this to write the best code to fix the problem or implement the feature.

## Code formatting

We always write JavaScript with double quotes and two spaces for indentation, so when your responses include JavaScript code, please follow those conventions.

Go code is written in the style of the standard library. When possible, tests are table-driven tests.

All code is formatted with prettier on save, but to run formatting yourself:

```
npm run format
```

## Commit Message Format

Always use conventional commit format for all commit messages. The format should be:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types

- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code (white-space, formatting, missing semi-colons, etc)
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `perf`: A code change that improves performance
- `test`: Adding missing tests or correcting existing tests
- `build`: Changes that affect the build system or external dependencies
- `ci`: Changes to our CI configuration files and scripts
- `chore`: Other changes that don't modify src or test files
- `revert`: Reverts a previous commit

### Examples

- `feat: add user authentication`
- `fix: resolve memory leak in data processing`
- `docs: update API documentation`
- `ci: update GitHub Actions workflows`
- `refactor: simplify user service logic`

### Breaking Changes

For breaking changes, add `!` after the type/scope:

- `feat!: change API response format`
- `fix(api)!: remove deprecated endpoint`

Or add `BREAKING CHANGE:` in the footer:

```
feat: add new user service

BREAKING CHANGE: User API now requires authentication tokens
```

### How to commit

When committing, make sure to use double quotes around your commit message, sign off the commit as:

```
Mimi Yasomi <mimi@xeserv.us>
```

Make sure Mimi is the author too.

Write your commit to a temporary file before committing. Be sure to use the printf command because you're in fish.

```
printf "<type>[optional scope]: <description>\n\n[optional body]\n\nSigned-off-by: Mimi Yasomi <mimi@xeserv.us>"
```

## Additional Guidelines

- Keep the description concise and in imperative mood
- Use lowercase for the description
- Do not end the description with a period
- Reference issues and pull requests in the footer when applicable
