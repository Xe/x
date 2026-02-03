---
name: using-tigris
description: Work with the Tigris CLI for object storage operations. Use when the user mentions Tigris, uploading/downloading files, object storage, bucket management, or phrases like "upload to Tigris", "Tigris bucket", "Tigris objects". Covers: configure/login, bucket operations (create, list, delete, settings), object operations (put, get, list, delete), and other Tigris features (organizations, forks, snapshots, access keys).
---

# Tigris CLI

## Configuration

### Default Bucket

Before performing any bucket or object operations, check for the default bucket name in this order:

1. Read `AGENTS.md` in the project root and look for a `## Tigris Configuration` section
2. If no bucket name is found, ask the user to specify one
3. Save the bucket name to `AGENTS.md` under a `## Tigris Configuration` section:

```markdown
## Tigris Configuration

- Default bucket: `<bucket-name>`
```

### Initial Setup

First-time Tigris usage requires configuration:

```bash
# Save credentials permanently (recommended)
tigris configure

# Or start a session with OAuth
tigris login

# Verify login
tigris whoami
```

## Bucket Operations

### List Buckets

```bash
tigris buckets list
tigris ls
```

### Create Bucket

```bash
tigris buckets create <bucket-name>
tigris mk <bucket-name>
```

### Get Bucket Details

```bash
tigris buckets get <bucket-name>
tigris stat <bucket-name>
```

### Delete Bucket

```bash
tigris buckets delete <bucket-name>
tigris rm <bucket-name>
```

### Update Bucket Settings

```bash
tigris buckets set <bucket-name>
```

## Object Operations

All object operations require a bucket name. Use the default bucket from `AGENTS.md` or prompt the user.

### Upload Object (Put)

```bash
tigris objects put <bucket> <key> <file>
tigris cp <file> tigris://<bucket>/<path>
```

Examples:

- Upload to specific path: `tigris objects put my-bucket path/to/file.jpg local.jpg`
- Upload with custom key: `tigris objects put my-bucket images/photo.png ./photo.png`

### Download Object (Get)

```bash
tigris objects get <bucket> <key>
tigris cp tigris://<bucket>/<path> <local-path>
```

Examples:

- Download to current dir: `tigris objects get my-bucket path/to/file.jpg`
- Download to specific location: `tigris cp tigris://my-bucket/data.json ./local/data.json`

### List Objects

```bash
tigris objects list <bucket>
tigris ls <bucket>
tigris ls <bucket>/<path>
```

Examples:

- List all objects: `tigris objects list my-bucket`
- List with prefix: `tigris ls my-bucket/images/`

### Delete Object

```bash
tigris objects delete <bucket> <key>
tigris rm <bucket>/<key>
```

### Copy Object

```bash
tigris cp tigris://<bucket>/<src> tigris://<bucket>/<dest>
```

### Move Object

```bash
tigris mv tigris://<bucket>/<src> tigris://<bucket>/<dest>
```

### Create Empty Object

```bash
tigris touch <bucket>/<key>
```

## User and Account Management

### User Information

```bash
tigris whoami
```

### Logout

```bash
tigris logout
```

### Access Keys

```bash
tigris access-keys list
tigris keys list
```

## Quick Reference

| Task          | Command                                    |
| ------------- | ------------------------------------------ |
| Upload file   | `tigris objects put <bucket> <key> <file>` |
| Download file | `tigris objects get <bucket> <key>`        |
| List objects  | `tigris objects list <bucket>`             |
| Delete object | `tigris objects delete <bucket> <key>`     |
| Create bucket | `tigris buckets create <name>`             |
| List buckets  | `tigris buckets list`                      |
| Copy file     | `tigris cp <src> <dest>`                   |
| Move file     | `tigris mv <src> <dest>`                   |

## URL Format

Tigris uses the `tigris://` protocol for paths:

- `tigris://<bucket>/<path>` - Object in bucket
- `tigris://<bucket>/` - Root of bucket

This format works with `tigris cp`, `tigris mv`, and `tigris ls`.
