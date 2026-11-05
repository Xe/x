# IAM policies (iamd)

iamd authorizes RPC calls with AWS-IAM-style policy documents whose
conditions are CEL filters evaluated over the method name and the decoded
request message. This page documents the model, the evaluation semantics,
and how downstream services enforce the same policies locally.

- Proto: `pb/within/website/x/iam/v1/policy.proto`
- Evaluator: `within.website/x/iam/authz`
- Management API: `iamd` `PolicyService` (admin-only)
- Storage: `cmd/iamd/models/policies.go` (SQLite via gorm)

## Model

A **policy** is a named document (UUIDv7 id, unique name) holding a list of
**statements**:

| Field     | Meaning                                                               |
| --------- | --------------------------------------------------------------------- |
| `effect`  | `EFFECT_ALLOW` or `EFFECT_DENY`                                       |
| `methods` | anchored glob patterns over full method names (below)                 |
| `filter`  | optional CEL expression over `(method, req, principal)`; empty = true |

Method names are `<proto package>.<Service>/<Method>`, e.g.
`within.website.x.iam.v1.KeyService/ListKeys`. Patterns match the whole
string; `*` spans any characters including `/`, `?` matches one character:

```
*                                       # every method
within.website.x.iam.v1.KeyService/*    # one whole service
within.website.x.iam.v1.*/List*         # listers across services
```

Filters bind three names:

- `method` — string, the full method name
- `req` — the decoded request proto for that method (typed at the enforcing service)
- `principal` — `within.website.x.iam.sts.v1.TokenIdentity`; `principal.principal_id` is the user UUID

Examples:

```cel
""                                      // no condition
req.user_id == principal.principal_id   // own resources only
req.user_id == "" || req.user_id == principal.principal_id
req.comment.startsWith("ci-")
```

## Evaluation

Per call, over every policy attached to the caller, in order:

1. An admin caller (`is_admin`) bypasses everything: allow.
2. A statement **applies** when a `methods` pattern matches and its filter
   is true.
3. Any applying `DENY` denies the call — explicit deny wins.
4. Otherwise any applying `ALLOW` allows the call.
5. Otherwise the call is implicitly denied.

Filter errors fail closed. These errors include compilation failures against
the method's request type and evaluation failures. A failed `ALLOW` filter
grants nothing. A failed `DENY` filter denies the matching method.

Failures increment `iam_authz_filter_errors_total`. If filter errors cause
a denial, the denial reason includes the cause. Another valid `ALLOW` can
still grant access after a failed `ALLOW` filter.

Response scoping (e.g. non-admin `ListKeys` being pinned to the caller's own
user) is deliberately **not** policy machinery; it stays in the handlers,
which already clamp requests before the DAO.

## Management

`PolicyService` (CreatePolicy, GetPolicy, ListPolicies, DeletePolicy,
AttachPolicy, DetachPolicy, ListPoliciesForUser) is gated on `is_admin`
inside its handlers — never on policies — so no document can lock an admin
out of the management surface. `CreatePolicy` checks every filter with
`authz.TryCompile`. This check requires a boolean result and checks the
concrete principal type. Request fields remain dynamic until enforcement,
where the evaluator uses the owning service's request types.

Policies are immutable after creation:
edit = create new + reattach + delete old, so an enforcing service's fetched
snapshot always compiles the same. `DeletePolicy` refuses while attachments
exist (`FAILED_PRECONDITION`).

`iamd`'s bootstrap user is an admin and bypasses all checks.

### CLI and policy files

Use `iam policies` to manage documents through the signed API. Remote
commands use the existing `iam --config <path>` configuration and require
admin credentials. `validate` runs offline and does not load credentials or
contact iamd.

Files contain a bare `Policy` document in protobuf JSON, not a
`{"policy": ...}` RPC request wrapper. For example, `own-keys.json`:

```json
{
  "name": "own-keys",
  "statements": [
    {
      "effect": "EFFECT_ALLOW",
      "methods": ["within.website.x.iam.v1.KeyService/ListKeys"],
      "filter": "req.user_id == '' || req.user_id == principal.principal_id"
    }
  ]
}
```

Validate and upload the document, then attach it to a user:

```sh
iam policies validate --file own-keys.json
POLICY_ID=$(iam policies create --file own-keys.json)
iam policies attach "$POLICY_ID" --user-id <user-id>
```

`create` prints only the assigned policy ID on stdout. Both `validate` and
`create` accept `--file -` to read stdin; both reject unknown/duplicate JSON
fields, invalid policy field values, and CEL compilation errors before any
upload. Request-specific field/type checks still happen at enforcement:
offline validation cannot prove a filter is valid for every matching RPC.

```sh
iam policies validate --file - < own-keys.json
iam policies create --file - < another-policy.json

iam policies get "$POLICY_ID"
iam policies get "$POLICY_ID" --json > exported-policy.json
iam policies validate --file exported-policy.json

iam policies list
iam policies list --count 20 --page 0 --json
iam policies list --user-id <user-id> --json

iam policies detach "$POLICY_ID" --user-id <user-id>
iam policies delete "$POLICY_ID"
```

`get --json` exports a bare document accepted by the file loader, including
server-assigned ID and timestamps. `create` always assigns fresh metadata;
to create a copy of an existing document, give it a different name first.
Names are unique, so importing the same document does not update it.
`list --json` returns a `{"policies": [...]}` list, not a single importable
document. Global lists use zero-based pages (default count 40, page 0).
`list --user-id` returns all attachments for that user and cannot be combined
with pagination flags.

Documents remain immutable. To replace one, create a differently named
policy, explicitly change attachments, and delete the old document once it
is unattached. Attaching the new document before detaching the old avoids
a gap in grants, but both policies apply during that overlap and any deny
still wins. Choose that order deliberately. Local file edits have no effect
until uploaded; there is no `apply`, file watcher, or automatic replacement.

### Checked-in examples

| File                                                                       | Grant                                                                                                    |
| -------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| [`own-keys.json`](../cmd/iam/examples/own-keys.json)                       | List and create keys for the caller's own user.                                                          |
| [`users-read-only.json`](../cmd/iam/examples/users-read-only.json)         | List users with page size 1–50 and a non-negative page; no user creation or disabling.                   |
| [`sts-sigv4a-verifier.json`](../cmd/iam/examples/sts-sigv4a-verifier.json) | Fetch public verification keys and their associated identity/policy metadata; no symmetric signing keys. |
| [`sts-sigv4-verifier.json`](../cmd/iam/examples/sts-sigv4-verifier.json)   | Fetch symmetric derived signing keys only for region `yow`, service `iam`.                               |

The CLI bundles these files as offline templates, so an installed binary
does not need a source checkout or credentials to render them:

```sh
iam policies templates list
iam policies templates render own-keys --name alice-keys > alice-keys.json
iam policies validate --file alice-keys.json

POLICY_ID=$(iam policies templates render sts-sigv4a-verifier --name backend-verifier |
  iam policies create --file -)
iam policies attach "$POLICY_ID" --user-id <backend-user-id>
```

`render` writes only a bare protobuf-JSON policy to stdout; it does not
create, attach, or replace anything on the server. `--name` overrides the
document name on any template. Otherwise, the example's name is retained.
Use a unique name when creating another document.

For the classic SigV4 template, override the lookup scope with both
`--region` and `--service`:

```sh
iam policies templates render sts-sigv4-verifier \
  --region us-east-1 --service mi --name mi-verifier > mi-verifier.json
```

Both scope flags must be supplied together and apply only to
`sts-sigv4-verifier`. Without them its example scope remains `yow`/`iam`.
With them the default document name becomes
`sts-sigv4-verifier-<region>-<service>`, unless `--name` overrides it.
Scope values are quoted as CEL string literals; they cannot inject new
conditions into the filter. Changing scope does not wire the named
application to IAM enforcement. Classic signing-key access still carries
the forgery capability described below.

The JSON examples remain the source of the bundled templates. Rebuild the
CLI after changing them; templates never read an editable runtime directory.

For example, to give a non-admin backend credential permission to fetch
SigV4A verification keys:

```sh
iam policies validate --file cmd/iam/examples/sts-sigv4a-verifier.json
POLICY_ID=$(iam policies create --file cmd/iam/examples/sts-sigv4a-verifier.json)
iam policies attach "$POLICY_ID" --user-id <backend-user-id>
```

Attach verifier policies to the backend's own IAM user, not to every end
user it authenticates. Lookup is intentionally not restricted to the
backend's own access key: verifiers need to look up incoming callers' keys.
Both lookup APIs also expose those callers' identity and attached policies,
so only grant lookup access to trusted backends.

Prefer the SigV4A example: public keys permit signature verification, not
forgery. The classic SigV4 example grants **forgery-capable symmetric key
material** for any enabled credential within the allowed scope; use it only
for a backend you trust with that capability. Its `yow` region matches
`kube/alrest/x/iamd.yaml`, and `iam` matches iamd's default service name.
Before upload, set the condition to the region and service that the backend
verifies. These fields describe the requested key scope. The backend uses
its own credentials to authenticate the lookup RPC to iamd.

These files grant only the listed operations; they do not revoke permissions
from other attached policies, and admin users bypass them entirely.
`PolicyService` requires `is_admin` in its handlers, so a policy document
cannot delegate policy administration to a non-admin.

Currently only iamd's RPC routes are wired to IAM policy enforcement in
this repository. Other applications such as `mi` and `xedn` do not yet
use that enforcement; the mi IAM migration document is a plan, not live
wiring. Attaching a document does not protect an application's routes until
that application installs the verifier and policy evaluator.

## Enforcement

### In iamd

A Twirp interceptor (`cmd/iamd/authz.go`) runs inside every mounted service
after the dual SigV4/SigV4A middleware authenticates the caller:

```
signature verify → resolve user → policy interceptor → handler
```

The interceptor loads the caller's attached policies per call, so attach and
detach take effect on the next request. Denials are `permission_denied`
with the deciding rule in the message.

### Downstream (distributed)

`SigningKeyService.GetSigningKey`/`GetPublicKey` responses carry the
caller's policy set and the identity's `is_admin` flag. The iamsts verifiers
(`web/middleware/sigv4/iamsts`, `web/middleware/sigv4a/iamsts`) cache both
alongside the key material for the server-advised TTL and store them in the
request context (`authctx.PrincipalPolicies`, `authctx.Caller().IsAdmin`).
Downstream services then enforce locally with the same semantics — no IAM
RPC on the hot path.

For complete server/client wiring, backend lookup credentials, caller
policies, and handler responsibilities, see
[Integrating a service with iamd policies](iam-service-integration.md).

Revocation/attachment latency is bounded by the signing-key cache TTL
(`-signing-key-cache-ttl`, default 5m).

## Metrics

- `iam_authz_decisions_total{decision}` — allow_admin / allow_policy /
  deny_explicit / deny_implicit / deny_filter_error
- `iam_authz_filter_seconds` — per-filter evaluation time
- `iam_authz_filter_errors_total` — compile/eval failures (fail-closed)

## Known limitations

- Filters require boolean results and valid principal fields at write time.
  Errors in request fields and types cause fail-closed decisions at enforcement.
- Policy documents are immutable; there is no in-place edit RPC.
- No groups or policy versioning yet; policies attach to users directly.
