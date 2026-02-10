# markdownlang-telemetryd Kubernetes Deployment

This directory contains the Kubernetes manifests for deploying the markdownlang telemetry server.

## Resources

- **Namespace**: `markdownlang` with restricted Pod Security Standards
- **Deployment**: Single replica with resource limits and security hardening
- **Service**: ClusterIP service exposing port 80
- **Ingress**: Exposed at `https://telemetry.markdownlang.lol`
- **OnePassword**: Secrets for Discord webhooks

## Secrets Required

Create a 1Password item named "MarkdownLang Telemetry" with the following fields:

| Field                   | Description                                          |
| ----------------------- | ---------------------------------------------------- |
| `discord-webhook`       | Discord webhook URL for new user notifications       |
| `xe-webhook`            | Discord webhook URL for quota exceeded notifications |
| `aws-access-key-id`     | Tigris AWS access key ID                             |
| `aws-secret-access-key` | Tigris AWS secret access key                         |

## Deployment

Apply the manifests:

```bash
kubectl apply -k kube/alrest/markdownlang-telemetryd
```

## Environment Variables

| Variable                                 | Default                  | Description                                      |
| ---------------------------------------- | ------------------------ | ------------------------------------------------ |
| `MARKDOWNLANG_TELEMETRY_BUCKET`          | `markdownlang-telemetry` | S3 bucket name for storing telemetry reports     |
| `MARKDOWNLANG_TELEMETRY_DISCORD_WEBHOOK` | _(secret)_               | Discord webhook for new user notifications       |
| `MARKDOWNLANG_TELEMETRY_XE_WEBHOOK`      | _(secret)_               | Discord webhook for quota exceeded notifications |
| `AWS_ACCESS_KEY_ID`                      | _(secret)_               | Tigris AWS access key ID                         |
| `AWS_SECRET_ACCESS_KEY`                  | _(secret)_               | Tigris AWS secret access key                     |
| `AWS_REGION`                             | `auto`                   | AWS region (must be "auto" for Tigris)           |
| `AWS_ENDPOINT_URL`                       | `https://t3.storage.dev` | Tigris S3 endpoint URL                           |
| `HTTP_ADDR`                              | `:9100`                  | HTTP listen address                              |

## Health Checks

The service exposes `/healthz` for liveness and readiness probes.

## Storage

All data is stored in S3/Tigris:

- **Telemetry reports**: `{email}/{timestamp}.json`
- **User tracking data**: `_meta/users.json`
- **Execution counts**: `_meta/counts.json`

The container runs with a read-only root filesystem and requires no persistent volumes.
