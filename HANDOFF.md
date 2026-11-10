# Handoff: IAM policy enforcement

Branch: `feat/iam-policies`.

## Failure and repair

The previous commit failed because the evaluator source directory, `iam/`,
was deleted as a presumed build artifact. That directory contains the real
`within.website/x/iam/authz` package. The reconstructed evaluator also lost
parts of the original implementation and its documentation no longer
matched the public API.

The failed reconstruction left these defects:

- Policy validation declared `principal` as dynamic. Unknown principal
  fields therefore passed the write-time check.
- Request environments did not reliably declare the concrete protobuf
  request type. Valid field filters failed during authorization.
- Boolean filter results did not receive a consistent write-time check.
- DENY filters were ignored, and an early ALLOW skipped later DENY rules.
- Trailing wildcard matches failed when `*` matched an empty suffix.
- The evaluator lost its documented metrics.
- Corrupt policy documents blocked admin repairs. Missing attached documents
  could silently remove a DENY rule. Attach and delete checks were not atomic.
- The service guide used the old variadic constructor and absent adapters.
- The handoff contained incorrect CEL API claims. Dynamic types permit field
  access, but cannot check fields against a concrete protobuf schema.

The repair uses `cel.dev/cel-go`. `TryCompile` registers the concrete
`TokenIdentity` type and requires a boolean result. Request fields remain
dynamic during policy creation because policies can target external services.
Each method's enforcement environment registers its concrete request type.

`authz.New` accepts `map[string]interface{}`. Each entry maps a full method
name to a sample protobuf request. The service guide uses this signature
and supplies its own context helper and Twirp interceptor. It calls the
existing `Evaluator.Check` API, preserves admin bypass, and rejects missing
verified callers.

Filters compile and evaluate per call. There is no compiled-program cache.
The metrics cover authorization decisions, filter duration, and filter errors.

## Feature scope

- Policy protobuf API, generated bindings, SQLite models, and admin-only
  policy management handlers.
- Offline policy validation, bundled CLI templates, and signed management
  commands.
- CEL authorization for iamd's mounted RPC services.
- Policy sets and admin identity in STS responses and verifier context.
- Documentation for downstream services that install their own enforcement.

Only iamd installs policy enforcement in this repository. Policy attachments
alone do not protect another application's routes.

## Required semantics

- Policies match full names: `<package>.<Service>/<Method>`.
- Glob patterns match the entire name. `*` spans `/`, and `?` matches one character.
- Admin callers bypass policy evaluation.
- Any applicable DENY overrides every ALLOW. No applicable ALLOW means denial.
- A filter error makes an ALLOW inapplicable and a DENY applicable.
- iamd reads attachments per call. Downstream verifiers retain policy snapshots
  until the server-advised key-cache TTL expires.
- Policy management handlers require an admin. Policies cannot delegate administration.
- Handlers retain responsibility for request validation and resource ownership.

## Validation and delivery

`npm test` passed after the repair. This command regenerates protobuf and Go
outputs, formats the repository, and runs `go test ./...`. Focused IAM, STS,
and SigV4/SigV4A tests also passed. Race checks passed for `iam/authz`,
`cmd/iamd/...`, and both iamsts verifier packages. The service guide examples compile with
existing IAM bindings substituted for the illustrative widget service.

Regression tests cover DENY precedence and conditions, typed fields, boolean
filters, wildcard boundaries, unavailable documents, admin recovery, and
policy/admin metadata on both fresh and cached verification requests.

Application imports use `cel.dev/cel-go v0.32.0`. The old CEL module remains
an indirect dependency of Protovalidate and the OpenAPI generation tool.

The user authorized a signed Conventional Commit, branch push, and pull request.
