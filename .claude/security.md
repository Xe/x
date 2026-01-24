# Security Best Practices

## Secrets Management

- Secrets never belong in the repository
- Use environment variables for runtime secrets
- Use the `secrets` directory for local development (ignored by Git)

## Dependency Security

- Run `npm audit` periodically and address reported vulnerabilities

## Helpful Documentation

When working with external services, use these resources:

- **Tigris** or **Tigris Data**: https://www.tigrisdata.com/docs/llms.txt
