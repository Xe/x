# Sakurajima - Project Memory

Conventions and gotchas specific to the Sakurajima reverse proxy.

## Gotchas

### H2C with http2.Transport

Configure `DialTLS` on `http2.Transport` **even for cleartext** H2C connections.
Leaving it unset does not fall back to a plain dialer.

### SSRF Protection

Target URLs are validated against private IP ranges. Reaching a private target
requires an explicit `allow_private_target` opt-in in the domain configuration.

### Validation Timing

Validate configurations at parse/load time, not when the configuration is
applied. Applying first lets an invalid config be partially applied before it
fails.

## Conventions

### Human-Readable Durations

Timeout values in HCL are human-readable strings (`"5s"`, `"100ms"`, `"1h30m"`),
parsed with `time.ParseDuration`. When a timeout is not configured, fall back to
a sensible default (e.g. 30s reads, 10s dial) rather than zero.

### atomic.Value for Hot-Reloading

Logger instances are stored in an `atomic.Value` so `*lumberjack.Logger` and
`*slog.Logger` can be swapped during configuration reload without restarting the
service. Load and type-assert on each use; do not cache the pointer.

### Configuration Errors Are Fatal

Do not swallow errors from configuration parsing. An invalid timeout must fail
the load explicitly, wrapped with context.
