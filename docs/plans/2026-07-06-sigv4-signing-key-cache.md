# SigV4 Derived Signing Key Caching Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the per-request `STSService/GetCallerIdentity` RPC with local SigV4 verification against a cached derived signing key fetched once per credential scope from a new `SigningKeyService/GetSigningKey` RPC.

**Architecture:** The existing local verifier (`web/middleware/sigv4`) already does full SigV4 verification; it gains a second key source — a `SigningKeyLookuper` that returns the _derived_ key for `(access_key_id, date, region, service)` instead of a raw secret. The `iamsts` package becomes a caching implementation of that interface: cache keyed on the exact scope tuple, TTL from the server's `cache_until` (clamped to `not_valid_after`), singleflight on misses, 30s negative caching. iamd serves `GetSigningKey` by deriving `HMAC(HMAC(HMAC(HMAC("AWS4"+secret, date), region), service), "aws4_request")` from the stored secret. The legacy `STSService`, its server handler, and `sigv4.VerifySignature` are deleted.

**Tech Stack:** Go, Twirp (buf-generated), GORM/SQLite (iamd), `golang.org/x/sync/singleflight` (already a dependency), AWS SDK v2 signer (tests only, already a dependency).

## Global Constraints

- Never log signing key bytes or secret access keys, at any log level. Audit every log line touched or added.
- No new module dependencies. singleflight comes from `golang.org/x/sync` v0.20.0, already in `go.mod`.
- Signature comparison stays `crypto/subtle.ConstantTimeCompare` (already the case in `sigv4.verify`; do not bypass it).
- Zero IAM RPCs on the hot path when the cache is warm. Verification is a pure function of (request bytes, cached key).
- Cache TTL comes from the server's `cache_until`, never past `not_valid_after`. Do not invent a client-side TTL. A response with no `cache_until` is used but not cached.
- Legacy deletion scope: everything in the **verify flow** — `STSService`, `GetCallerIdentityReq/Resp`, `Header`, the `cmd/iamd/services/iam/sts` `GetCallerIdentity` handler, `sigv4.VerifySignature`. `iam.proto`'s `UserService`/`KeyService` are the provisioning plane and stay.
- Client-facing errors: this repo emits Twirp errors (not S3 XML). Preserve the existing `sigv4.Middleware` mapping: missing auth → `unauthenticated`; clock skew / scope mismatch / body too large / streaming / missing signed host → `invalid_argument` with the sentinel message; unknown key, disabled key, and signature mismatch → `permission_denied` "invalid authentication header" (disabled-vs-unknown distinction logged internally only); IAM outage / unexpected → `internal` "internal error".
- `internal.HandleStartup()` calls `flag.Parse()` — binaries must not call `flag.Parse()` themselves.
- All commits: Conventional Commits, `--signoff`.
- Proto regeneration: `npm run generate` (buf generate + go generate + format). Full test run: `npm test`.

---

### Task 1: sigv4 core — derived-key verification support

**Files:**

- Modify: `web/middleware/sigv4/sigv4.go`
- Modify: `web/middleware/sigv4/lookuper.go`
- Test: `web/middleware/sigv4/sigv4_test.go` (append), `web/middleware/sigv4/keylookup_e2e_test.go` (create)

**Interfaces:**

- Consumes: nothing new.
- Produces (used by Tasks 4, 5, 6):
  - `func DeriveSigningKey(secret, date, region, service string) []byte` — exported rename of the private `signingKey`.
  - `type SigningKeyLookuper interface { LookupSigningKey(ctx context.Context, accessKeyID, date, region, service string) ([]byte, error) }` plus `SigningKeyLookuperFunc` adapter.
  - `Verifier.KeyLookup SigningKeyLookuper` field — alternative to `Lookup`; `KeyLookup` wins when both are set.
  - `type Credential struct { AccessKeyID, Date, Region, Service string }` and `func ParseCredential(authorization string) (*Credential, error)`.
  - `func TwirpError(ctx context.Context, err error) error` — the error-mapping switch extracted from `Middleware`.

- [ ] **Step 1: Write the failing tests**

Append to `web/middleware/sigv4/sigv4_test.go` (package `sigv4`, internal — check the existing package clause and match it; if the file is `package sigv4` these compile as-is):

```go
// TestDeriveSigningKey pins the derivation against the worked example in the
// AWS documentation ("Examples of how to derive a signing key for Signature
// Version 4"), so it is a known-answer test against AWS, not against ourselves.
func TestDeriveSigningKey(t *testing.T) {
	got := hex.EncodeToString(DeriveSigningKey(
		"wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY", "20120215", "us-east-1", "iam"))
	const want = "f4780e2d9f65fa895f9c67b32ce1baf0b0d8a43505a000a1a9e090d414db404d"
	if got != want {
		t.Errorf("DeriveSigningKey = %s, want %s", got, want)
	}
}

// TestParseCredential extracts the scope tuple from an Authorization header.
func TestParseCredential(t *testing.T) {
	const h = "AWS4-HMAC-SHA256 Credential=AKIDEXAMPLE/20150830/us-east-1/iam/aws4_request, SignedHeaders=content-type;host;x-amz-date, Signature=5d672d79c15b13162d9279b0855cfba6789a8edb4c82c400e06b5924a6f2b5d7"
	c, err := ParseCredential(h)
	if err != nil {
		t.Fatalf("ParseCredential: %v", err)
	}
	want := Credential{AccessKeyID: "AKIDEXAMPLE", Date: "20150830", Region: "us-east-1", Service: "iam"}
	if *c != want {
		t.Errorf("credential = %+v, want %+v", *c, want)
	}
	if _, err := ParseCredential("Bearer nope"); !errors.Is(err, ErrMissingAuth) {
		t.Errorf("bad header err = %v, want ErrMissingAuth", err)
	}
}
```

Create `web/middleware/sigv4/keylookup_e2e_test.go`. It proves a request signed by the AWS SDK signer verifies identically through the `Lookup` (raw secret) path and the `KeyLookup` (derived key) path, and that `KeyLookup` sees the literal scope strings:

```go
package sigv4_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	signer "github.com/aws/aws-sdk-go-v2/aws/signer/v4"

	"within.website/x/web/middleware/sigv4"
)

const (
	klTestKey    = "AKIDEXAMPLE"
	klTestSecret = "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	klRegion     = "us-east-1"
	klService    = "execute-api"
	// SHA-256 of the empty string: the payload hash of a bodyless GET.
	klEmptySHA = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// signGET signs a bodyless GET for target at ts with the AWS SDK v4 signer,
// the reference implementation.
func signGET(t *testing.T, target string, ts time.Time) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req.Header.Set("X-Amz-Content-Sha256", klEmptySHA)
	if err := signer.NewSigner().SignHTTP(context.Background(),
		aws.Credentials{AccessKeyID: klTestKey, SecretAccessKey: klTestSecret},
		req, klEmptySHA, klService, klRegion, ts); err != nil {
		t.Fatalf("SignHTTP: %v", err)
	}
	return req
}

// TestVerify_KeyLookup verifies an SDK-signed request using only a derived
// signing key, and confirms the lookuper receives the literal scope strings
// from the Credential= component.
func TestVerify_KeyLookup(t *testing.T) {
	var gotScope []string
	v := &sigv4.Verifier{
		Region:  klRegion,
		Service: klService,
		KeyLookup: sigv4.SigningKeyLookuperFunc(func(_ context.Context, akid, date, region, service string) ([]byte, error) {
			gotScope = []string{akid, date, region, service}
			if akid != klTestKey {
				return nil, sigv4.ErrUnknownKey
			}
			return sigv4.DeriveSigningKey(klTestSecret, date, region, service), nil
		}),
	}

	now := time.Now().UTC()
	req := signGET(t, "https://svc.example.com/things?a=1", now)

	keyID, err := v.Verify(req)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if keyID != klTestKey {
		t.Errorf("keyID = %q, want %q", keyID, klTestKey)
	}
	wantScope := []string{klTestKey, now.Format("20060102"), klRegion, klService}
	if strings.Join(gotScope, "/") != strings.Join(wantScope, "/") {
		t.Errorf("scope = %v, want %v", gotScope, wantScope)
	}

	// A tampered signature must fail with ErrUnauthorized.
	bad := signGET(t, "https://svc.example.com/things?a=1", now)
	auth := bad.Header.Get("Authorization")
	bad.Header.Set("Authorization", auth[:len(auth)-1]+flipHex(auth[len(auth)-1]))
	if _, err := v.Verify(bad); err == nil {
		t.Error("tampered signature verified")
	}
}

// flipHex returns a different valid hex digit so the signature stays
// well-formed but wrong.
func flipHex(b byte) string {
	if b == '0' {
		return "1"
	}
	return "0"
}

// TestVerify_NeitherLookupConfigured pins the misconfiguration guard.
func TestVerify_NeitherLookupConfigured(t *testing.T) {
	v := &sigv4.Verifier{Region: klRegion, Service: klService}
	req := signGET(t, "https://svc.example.com/", time.Now().UTC())
	if _, err := v.Verify(req); err == nil {
		t.Error("Verify with no lookuper succeeded")
	}
}
```

Note: if `web/middleware/sigv4/e2e_test.go` already declares helpers with these names in `package sigv4_test`, rename the local ones (`klSignGET`, etc.) to avoid collisions — read that file first.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./web/middleware/sigv4/ -run 'TestDeriveSigningKey|TestParseCredential|TestVerify_KeyLookup|TestVerify_Neither' -v`
Expected: compile errors — `DeriveSigningKey`, `ParseCredential`, `SigningKeyLookuperFunc`, `KeyLookup` undefined.

- [ ] **Step 3: Implement**

In `web/middleware/sigv4/lookuper.go`, append:

```go
// SigningKeyLookuper resolves a credential scope to the SigV4 derived signing
// key HMAC(HMAC(HMAC(HMAC("AWS4"+secret, date), region), service),
// "aws4_request"). The arguments are the literal strings from the request's
// Credential= component, unnormalized. Return ErrUnknownKey when the key does
// not exist or may not sign (disabled key or user); any other error is treated
// as a server fault.
type SigningKeyLookuper interface {
	LookupSigningKey(ctx context.Context, accessKeyID, date, region, service string) ([]byte, error)
}

// SigningKeyLookuperFunc adapts an ordinary function to SigningKeyLookuper.
type SigningKeyLookuperFunc func(ctx context.Context, accessKeyID, date, region, service string) ([]byte, error)

// LookupSigningKey calls f.
func (f SigningKeyLookuperFunc) LookupSigningKey(ctx context.Context, accessKeyID, date, region, service string) ([]byte, error) {
	return f(ctx, accessKeyID, date, region, service)
}
```

(Add `"context"` to that file's imports.)

In `web/middleware/sigv4/sigv4.go`:

1. Rename `signingKey` → `DeriveSigningKey` with a doc comment; update the one call site in `verify`:

```go
// DeriveSigningKey computes the SigV4 derived signing key for a credential
// scope: HMAC(HMAC(HMAC(HMAC("AWS4"+secret, date), region), service),
// "aws4_request"). The derived key can only validate requests whose scope
// matches (date, region, service) exactly, so exposure is bounded to one UTC
// day and one service; it never reveals the secret.
func DeriveSigningKey(secret, date, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte(service))
	return hmacSHA256(kService, []byte(terminator))
}
```

2. Add the `KeyLookup` field to `Verifier` after `Lookup`:

```go
	// KeyLookup resolves a credential scope to its derived signing key. When
	// set it takes precedence over Lookup, and the verifier never sees the
	// raw secret — this is how services that must not hold secrets verify
	// locally (see web/middleware/sigv4/iamsts). Exactly one of Lookup or
	// KeyLookup must be set.
	KeyLookup SigningKeyLookuper
```

3. In `Verify`, replace the guard `if v.Lookup == nil { return "", ErrNotConfigured }` with:

```go
	if v.Lookup == nil && v.KeyLookup == nil {
		return "", ErrNotConfigured
	}
```

Update the `ErrNotConfigured` message in the `var` block to `"sigv4: neither Verifier.Lookup nor Verifier.KeyLookup is set"`.

4. In `verify` (the shared core), replace the secret-lookup-and-derive block:

```go
	secret, err := v.Lookup.Lookup(sr.accessKeyID)
	if err != nil {
		return "", err
	}
```

and, below, `key := signingKey(secret, sr.scope.date, sr.scope.region, sr.scope.service)` with:

```go
	var key []byte
	if v.KeyLookup != nil {
		key, err = v.KeyLookup.LookupSigningKey(r.Context(), sr.accessKeyID, sr.scope.date, sr.scope.region, sr.scope.service)
	} else {
		var secret string
		secret, err = v.Lookup.Lookup(sr.accessKeyID)
		if err == nil {
			key = DeriveSigningKey(secret, sr.scope.date, sr.scope.region, sr.scope.service)
		}
	}
	if err != nil {
		return "", err
	}
```

(`verify` already receives `*http.Request`; `r.Context()` on the synthetic request built by `VerifySignature` returns `context.Background()`, so this compiles while `VerifySignature` still exists. `VerifySignature` is deleted in Task 7.)

Note the ordering property this preserves: scope pinning, signed-host, and clock-skew checks all run **before** the key lookup, so skewed or mis-scoped requests never reach the lookuper (and therefore never trigger an RPC in Task 5).

5. Add `Credential` / `ParseCredential` near `parseAuthHeader`:

```go
// Credential is the parsed Credential= component of a SigV4 Authorization
// header: the access key id plus the literal, unnormalized scope strings.
type Credential struct {
	AccessKeyID string
	Date        string
	Region      string
	Service     string
}

// ParseCredential extracts the credential scope from an AWS4-HMAC-SHA256
// Authorization header value. It returns ErrMissingAuth for anything
// malformed, matching Verify.
func ParseCredential(authorization string) (*Credential, error) {
	sr, err := parseAuthHeader(authorization)
	if err != nil {
		return nil, err
	}
	return &Credential{
		AccessKeyID: sr.accessKeyID,
		Date:        sr.scope.date,
		Region:      sr.scope.region,
		Service:     sr.scope.service,
	}, nil
}
```

6. Extract the error mapping from `Middleware` into an exported function and call it from `Middleware`:

```go
// TwirpError maps a verification error to the twirp error middlewares write
// to clients. Sentinels that describe the caller's own request keep their
// message; key and signature failures collapse to one opaque message so a
// probe cannot distinguish unknown, disabled, and mis-signed credentials.
// Unexpected errors are logged with their cause and surfaced as an opaque
// internal error.
func TwirpError(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, ErrMissingAuth):
		return twirp.WrapError(twirp.Unauthenticated.Error("no authentication header present"), err)
	case errors.Is(err, ErrScopeMismatch),
		errors.Is(err, ErrClockSkew), errors.Is(err, ErrBodyTooLarge),
		errors.Is(err, ErrStreamingUnsupported), errors.Is(err, ErrMissingSignedHost):
		// These sentinels describe the caller's own request and carry no
		// internal detail, so surface them: a client cannot correct clock
		// skew it can't distinguish from a scope mismatch.
		return twirp.WrapError(twirp.InvalidArgument.Error(err.Error()), err)
	case errors.Is(err, ErrUnknownKey), errors.Is(err, ErrUnauthorized),
		errors.Is(err, ErrBodyHash):
		return twirp.WrapError(twirp.PermissionDenied.Error("invalid authentication header"), err)
	default:
		// Unexpected errors (e.g. the key store being down) are server
		// faults; log the cause but never echo it to the caller.
		slog.ErrorContext(ctx, "sigv4 verification failed unexpectedly", "err", err)
		return twirp.WrapError(twirp.Internal.Error("internal error"), err)
	}
}
```

`Middleware` body becomes:

```go
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keyID, err := v.Verify(r)
		if err != nil {
			slog.DebugContext(r.Context(), "cannot serve request", "err", err, "method", r.Method, "path", r.URL.Path)
			twirp.WriteError(w, TwirpError(r.Context(), err))
			return
		}
		next.ServeHTTP(w, r.WithContext(withKeyID(r.Context(), keyID)))
	})
```

Add `"context"` to sigv4.go's imports.

- [ ] **Step 4: Run the package tests**

Run: `go test ./web/middleware/sigv4/... -v`
Expected: PASS, including all pre-existing tests (the `Lookup` path is behavior-identical).

- [ ] **Step 5: Commit**

```bash
git add web/middleware/sigv4/sigv4.go web/middleware/sigv4/lookuper.go web/middleware/sigv4/sigv4_test.go web/middleware/sigv4/keylookup_e2e_test.go
git commit --signoff -m "feat(sigv4): support verification from a derived signing key"
```

---

### Task 2: DAO — key lookup that distinguishes disabled from unknown

**Files:**

- Modify: `cmd/iamd/models/keys.go`
- Test: `cmd/iamd/models/dao_test.go` (append)

**Interfaces:**

- Consumes: existing `models.DAO`, `models.Key` (`gorm.Model`, `User *User`), `PropagateUnscoped: true` set in `Open`.
- Produces (used by Task 4): `func (d *DAO) GetKeyWithUser(ctx context.Context, accessKeyID string) (*Key, error)` — returns the key with `User` preloaded, **including** soft-deleted key/user rows; `gorm.ErrRecordNotFound` only when the access key id has never existed. Disabled state is visible via `k.DeletedAt.Valid` / `k.User.DeletedAt.Valid`.

- [ ] **Step 1: Write the failing test**

Append to `cmd/iamd/models/dao_test.go` (match the existing test-helper style in that file — it already has a way to open a throwaway DAO; reuse it, shown here as `testDAO(t)`, and adapt the name to whatever exists):

```go
func TestGetKeyWithUser(t *testing.T) {
	ctx := context.Background()
	dao := testDAO(t)

	u, err := dao.CreateUser(ctx, "kayla")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	k, err := dao.CreateKey(ctx, u, "test key")
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}

	t.Run("active key and user", func(t *testing.T) {
		got, err := dao.GetKeyWithUser(ctx, k.AccessKeyID)
		if err != nil {
			t.Fatalf("GetKeyWithUser: %v", err)
		}
		if got.DeletedAt.Valid {
			t.Error("active key reported as disabled")
		}
		if got.User == nil || got.User.UUID != u.UUID {
			t.Fatalf("user = %+v, want UUID %s", got.User, u.UUID)
		}
		if got.SecretAccessKey != k.SecretAccessKey {
			t.Error("secret not loaded")
		}
	})

	t.Run("unknown key is not found", func(t *testing.T) {
		if _, err := dao.GetKeyWithUser(ctx, "AKIDNOPE"); !errors.Is(err, gorm.ErrRecordNotFound) {
			t.Fatalf("err = %v, want ErrRecordNotFound", err)
		}
	})

	t.Run("disabled key still loads, marked disabled", func(t *testing.T) {
		if err := dao.DisableKey(ctx, k.AccessKeyID, "test", ""); err != nil {
			t.Fatalf("DisableKey: %v", err)
		}
		got, err := dao.GetKeyWithUser(ctx, k.AccessKeyID)
		if err != nil {
			t.Fatalf("GetKeyWithUser after disable: %v", err)
		}
		if !got.DeletedAt.Valid {
			t.Error("disabled key not marked disabled")
		}
	})

	t.Run("disabled user still loads, marked disabled", func(t *testing.T) {
		u2, err := dao.CreateUser(ctx, "mara")
		if err != nil {
			t.Fatalf("CreateUser: %v", err)
		}
		k2, err := dao.CreateKey(ctx, u2, "second")
		if err != nil {
			t.Fatalf("CreateKey: %v", err)
		}
		if err := dao.DisableUser(ctx, u2.UUID, "test"); err != nil {
			t.Fatalf("DisableUser: %v", err)
		}
		got, err := dao.GetKeyWithUser(ctx, k2.AccessKeyID)
		if err != nil {
			t.Fatalf("GetKeyWithUser: %v", err)
		}
		if got.User == nil || !got.User.DeletedAt.Valid {
			t.Errorf("disabled user = %+v, want loaded with DeletedAt set", got.User)
		}
	})
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./cmd/iamd/models/ -run TestGetKeyWithUser -v`
Expected: compile error — `GetKeyWithUser` undefined.

- [ ] **Step 3: Implement**

Append to `cmd/iamd/models/keys.go`:

```go
// GetKeyWithUser returns the key with the given access key id and its owning
// user, including soft-deleted (disabled) rows for both — disabled state is
// visible on DeletedAt rather than hidden by GORM's default scope. It exists
// for the signing-key issuer, which must distinguish "no such key" from "key
// or user disabled". Use SecretFor when disabled keys must not load at all.
func (d *DAO) GetKeyWithUser(ctx context.Context, accessKeyID string) (*Key, error) {
	var k Key
	// PropagateUnscoped is set on the session, so Unscoped applies to the
	// User preload as well.
	if err := d.db.WithContext(ctx).Unscoped().
		Preload("User").
		Where("access_key_id = ?", accessKeyID).
		First(&k).Error; err != nil {
		return nil, err
	}
	return &k, nil
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./cmd/iamd/... -v`
Expected: PASS. If the disabled-user preload comes back without `DeletedAt` set, `PropagateUnscoped` did not reach the preload — fall back to an explicit unscoped preload: `Preload("User", func(db *gorm.DB) *gorm.DB { return db.Unscoped() })`.

- [ ] **Step 5: Commit**

```bash
git add cmd/iamd/models/keys.go cmd/iamd/models/dao_test.go
git commit --signoff -m "feat(iamd): add DAO lookup that distinguishes disabled keys from unknown"
```

---

### Task 3: proto — add SigningKeyService (additive; legacy removed in Task 7)

**Files:**

- Modify: `pb/within/website/x/iam/sts/v1/sts.proto`
- Generated: `gen/within/website/x/iam/sts/v1/*` (via `npm run generate`)

**Interfaces:**

- Produces (used by Tasks 4, 5): `stsv1.SigningKeyService` (Twirp interface + `NewSigningKeyServiceProtobufClient` + `NewSigningKeyServiceServer` + `SigningKeyServicePathPrefix`), `GetSigningKeyRequest{AccessKeyId, Date, Region, Service}`, `GetSigningKeyResponse{SigningKey []byte, Identity *TokenIdentity, NotValidAfter, CacheUntil *timestamppb.Timestamp}`, `TokenIdentity{AccessKeyId, OrganizationId, PrincipalId, DisplayName}`.

- [ ] **Step 1: Add the service to the proto**

Append to `pb/within/website/x/iam/sts/v1/sts.proto` (leave `STSService` in place for now so the repo keeps compiling; it is deleted in Task 7):

```proto
// SigningKeyService hands out SigV4 derived signing keys so that downstream
// services can verify request signatures locally without ever holding raw
// secret access keys.
//
// A derived key is HMAC(HMAC(HMAC(HMAC("AWS4"+secret, date), region),
// service), "aws4_request"). It can only validate requests whose credential
// scope matches (date, region, service) exactly, so a leaked response is
// bounded to one service, one region, one UTC day.
//
// This API is internal-only and its responses contain key material. Callers
// authenticate to it the same way as every other iamd route: by signing the
// RPC with their own IAM credential (see cmd/iamd's route middleware).
service SigningKeyService {
  // GetSigningKey returns the derived signing key for the given access key
  // id and credential scope, plus the identity the key belongs to.
  //
  // Errors:
  //   NOT_FOUND         - no such access key id
  //   PERMISSION_DENIED - key or owning user is disabled, or the requested
  //                       (region, service, date) scope is outside what this
  //                       deployment issues keys for
  //   INVALID_ARGUMENT  - malformed access_key_id/date/region/service
  rpc GetSigningKey(GetSigningKeyRequest) returns (GetSigningKeyResponse);
}

message GetSigningKeyRequest {
  // The access key id parsed from the request's Credential= component.
  string access_key_id = 1 [(buf.validate.field).required = true];

  // Remaining components of the credential scope, exactly as they will be
  // fed into the HMAC ladder: the literal strings from
  // "date/region/service/aws4_request", not normalized forms.

  // UTC date in YYYYMMDD form, e.g. "20260706".
  string date = 2 [
    (buf.validate.field).required = true,
    (buf.validate.field).string.pattern = "^[0-9]{8}$"
  ];

  // Region code, e.g. "us-east-1".
  string region = 3 [(buf.validate.field).required = true];

  // Service code, e.g. "iam".
  string service = 4 [(buf.validate.field).required = true];
}

message GetSigningKeyResponse {
  // The 32-byte derived signing key. Raw bytes, not hex.
  bytes signing_key = 1;

  // Who this key authenticates as. Downstream services use this for
  // authorization and audit attribution after the signature checks out.
  TokenIdentity identity = 2;

  // Hard upper bound on validity: the scope date's UTC day plus the maximum
  // clock skew, after which no legitimate request can carry this scope.
  // Never cache the key past this instant.
  google.protobuf.Timestamp not_valid_after = 3;

  // How long the caller may cache this response before asking again. Bounds
  // revocation latency: a disabled key stops verifying within one cache TTL
  // instead of at end of day. Always <= not_valid_after.
  google.protobuf.Timestamp cache_until = 4;
}

// TokenIdentity identifies the principal a signing key belongs to.
message TokenIdentity {
  // The access key id the identity was resolved from.
  string access_key_id = 1;

  // The organization the credential belongs to. Reserved for multi-tenant
  // deployments; iamd currently has no organization concept and leaves it
  // empty.
  string organization_id = 2;

  // The principal within the organization: iamd's user UUID.
  string principal_id = 3;

  // Human-readable name for logs and error messages: iamd's user name.
  string display_name = 4;
}
```

- [ ] **Step 2: Regenerate and build**

Run: `npm run generate && go build ./...`
Expected: new symbols appear in `gen/within/website/x/iam/sts/v1/` (`sts.pb.go`, `sts.twirp.go`, `sts_grpc.pb.go`, `stsv1connect/`); everything compiles. Spot-check: `grep -n "SigningKeyServicePathPrefix" gen/within/website/x/iam/sts/v1/sts.twirp.go` has a hit.

- [ ] **Step 3: Commit**

```bash
git add pb/within/website/x/iam/sts/v1/sts.proto gen/within/website/x/iam/sts/v1/
git commit --signoff -m "feat(pb/iam): add SigningKeyService for derived signing key distribution"
```

---

### Task 4: iamd — GetSigningKey server + wiring

**Files:**

- Create: `cmd/iamd/services/iam/sts/signingkey.go`
- Modify: `cmd/iamd/main.go`
- Test: `cmd/iamd/services/iam/sts/signingkey_test.go`

**Interfaces:**

- Consumes: `dao.GetKeyWithUser` (Task 2), `sigv4.DeriveSigningKey` (Task 1), `stsv1.SigningKeyService` (Task 3).
- Produces: `func NewSigningKeys(dao *models.DAO, region, service string, cacheTTL time.Duration) *SigningKeys` implementing `stsv1.SigningKeyService`; `SigningKeys.Now func() time.Time` overridable for tests. New iamd flag `-signing-key-cache-ttl` (default `5m`). Route `stsv1.SigningKeyServicePathPrefix` wrapped in the same `stack` as every other route.

- [ ] **Step 1: Write the failing tests**

Create `cmd/iamd/services/iam/sts/signingkey_test.go`:

```go
package sts

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/twitchtv/twirp"

	"within.website/x/cmd/iamd/models"
	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	"within.website/x/web/middleware/sigv4"
)

const (
	skRegion  = "us-east-1"
	skService = "iam"
)

// fixedNow is an arbitrary instant mid-day UTC so the ±1 day issuance window
// is unambiguous.
var fixedNow = time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)

func newSigningKeysTest(t *testing.T) (*SigningKeys, *models.DAO, string, string) {
	t.Helper()
	dao := newDAO(t) // reuse/mirror the helper pattern already in sts_test.go
	u, err := dao.CreateUser(context.Background(), "tester")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	k, err := dao.CreateKey(context.Background(), u, "test key")
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}
	s := NewSigningKeys(dao, skRegion, skService, 5*time.Minute)
	s.Now = func() time.Time { return fixedNow }
	return s, dao, k.AccessKeyID, k.SecretAccessKey
}

func validReq(akid string) *stsv1.GetSigningKeyRequest {
	return &stsv1.GetSigningKeyRequest{
		AccessKeyId: akid,
		Date:        fixedNow.Format("20060102"),
		Region:      skRegion,
		Service:     skService,
	}
}

func wantTwirpCode(t *testing.T, err error, want twirp.ErrorCode) {
	t.Helper()
	var te twirp.Error
	if !errors.As(err, &te) {
		t.Fatalf("err = %v, want twirp %s", err, want)
	}
	if te.Code() != want {
		t.Fatalf("code = %s, want %s", te.Code(), want)
	}
}

func TestGetSigningKey(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		s, _, akid, secret := newSigningKeysTest(t)
		resp, err := s.GetSigningKey(ctx, validReq(akid))
		if err != nil {
			t.Fatalf("GetSigningKey: %v", err)
		}
		want := sigv4.DeriveSigningKey(secret, fixedNow.Format("20060102"), skRegion, skService)
		if !bytes.Equal(resp.GetSigningKey(), want) {
			t.Error("derived key mismatch")
		}
		if got := resp.GetIdentity().GetDisplayName(); got != "tester" {
			t.Errorf("display_name = %q, want tester", got)
		}
		if resp.GetIdentity().GetPrincipalId() == "" {
			t.Error("principal_id empty")
		}
		if resp.GetIdentity().GetAccessKeyId() != akid {
			t.Errorf("identity access_key_id = %q, want %q", resp.GetIdentity().GetAccessKeyId(), akid)
		}

		nva := resp.GetNotValidAfter().AsTime()
		wantNVA := time.Date(2026, 7, 7, 0, 15, 0, 0, time.UTC) // end of UTC day + 15m skew
		if !nva.Equal(wantNVA) {
			t.Errorf("not_valid_after = %v, want %v", nva, wantNVA)
		}
		cu := resp.GetCacheUntil().AsTime()
		if !cu.Equal(fixedNow.Add(5 * time.Minute)) {
			t.Errorf("cache_until = %v, want now+5m", cu)
		}
		if cu.After(nva) {
			t.Error("cache_until exceeds not_valid_after")
		}
	})

	t.Run("cache_until clamped to not_valid_after", func(t *testing.T) {
		s, _, akid, _ := newSigningKeysTest(t)
		// 5 minutes before the validity bound, cache_until must clamp.
		s.Now = func() time.Time { return time.Date(2026, 7, 7, 0, 12, 0, 0, time.UTC) }
		req := validReq(akid)
		req.Date = "20260706" // still yesterday's scope, inside the ±1 day window
		resp, err := s.GetSigningKey(ctx, req)
		if err != nil {
			t.Fatalf("GetSigningKey: %v", err)
		}
		wantNVA := time.Date(2026, 7, 7, 0, 15, 0, 0, time.UTC)
		if !resp.GetCacheUntil().AsTime().Equal(wantNVA) {
			t.Errorf("cache_until = %v, want clamped to %v", resp.GetCacheUntil().AsTime(), wantNVA)
		}
	})

	t.Run("unknown key is NOT_FOUND", func(t *testing.T) {
		s, _, _, _ := newSigningKeysTest(t)
		_, err := s.GetSigningKey(ctx, validReq("AKIDNOPE"))
		wantTwirpCode(t, err, twirp.NotFound)
	})

	t.Run("disabled key is PERMISSION_DENIED", func(t *testing.T) {
		s, dao, akid, _ := newSigningKeysTest(t)
		if err := dao.DisableKey(ctx, akid, "test", ""); err != nil {
			t.Fatalf("DisableKey: %v", err)
		}
		_, err := s.GetSigningKey(ctx, validReq(akid))
		wantTwirpCode(t, err, twirp.PermissionDenied)
	})

	t.Run("disabled user is PERMISSION_DENIED", func(t *testing.T) {
		s, dao, akid, _ := newSigningKeysTest(t)
		us, err := dao.ListUsers(ctx, 10, 0)
		if err != nil {
			t.Fatalf("ListUsers: %v", err)
		}
		if err := dao.DisableUser(ctx, us[0].UUID, "test"); err != nil {
			t.Fatalf("DisableUser: %v", err)
		}
		_, err = s.GetSigningKey(ctx, validReq(akid))
		wantTwirpCode(t, err, twirp.PermissionDenied)
	})

	t.Run("wrong scope is PERMISSION_DENIED", func(t *testing.T) {
		s, _, akid, _ := newSigningKeysTest(t)
		req := validReq(akid)
		req.Region = "eu-west-1"
		_, err := s.GetSigningKey(ctx, req)
		wantTwirpCode(t, err, twirp.PermissionDenied)
	})

	t.Run("date outside issuance window is PERMISSION_DENIED", func(t *testing.T) {
		s, _, akid, _ := newSigningKeysTest(t)
		req := validReq(akid)
		req.Date = "20260101"
		_, err := s.GetSigningKey(ctx, req)
		wantTwirpCode(t, err, twirp.PermissionDenied)
	})

	t.Run("malformed date is INVALID_ARGUMENT", func(t *testing.T) {
		s, _, akid, _ := newSigningKeysTest(t)
		for _, bad := range []string{"", "2026-07-06", "20261340", "garbage!"} {
			req := validReq(akid)
			req.Date = bad
			_, err := s.GetSigningKey(ctx, req)
			wantTwirpCode(t, err, twirp.InvalidArgument)
		}
	})

	t.Run("missing fields are INVALID_ARGUMENT", func(t *testing.T) {
		s, _, akid, _ := newSigningKeysTest(t)
		for name, mut := range map[string]func(*stsv1.GetSigningKeyRequest){
			"access_key_id": func(r *stsv1.GetSigningKeyRequest) { r.AccessKeyId = "" },
			"region":        func(r *stsv1.GetSigningKeyRequest) { r.Region = "" },
			"service":       func(r *stsv1.GetSigningKeyRequest) { r.Service = "" },
		} {
			req := validReq(akid)
			mut(req)
			_, err := s.GetSigningKey(ctx, req)
			if err == nil {
				t.Fatalf("%s: missing field accepted", name)
			}
			wantTwirpCode(t, err, twirp.InvalidArgument)
		}
	})
}
```

Note: `newDAO` — the existing `sts_test.go` in this package almost certainly has a DAO helper; reuse it. If its name differs, adapt. If this package's old test file blocks compilation because of Task-7-pending deletions, don't delete anything yet — the old tests still pass at this point.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./cmd/iamd/services/iam/sts/ -run TestGetSigningKey -v`
Expected: compile error — `NewSigningKeys` undefined.

- [ ] **Step 3: Implement the server**

Create `cmd/iamd/services/iam/sts/signingkey.go`:

```go
package sts

import (
	"context"
	"errors"
	"regexp"
	"time"

	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	"within.website/x/cmd/iamd/models"
	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	"within.website/x/web/middleware/sigv4"
)

// maxClockSkew mirrors the verifier's clock-skew window: a request signed at
// 23:59:59 with scope date D is legitimately verifiable until 00:15 on D+1,
// so a key for D must outlive its UTC day by exactly this much.
const maxClockSkew = 15 * time.Minute

var dateRe = regexp.MustCompile(`^[0-9]{8}$`)

// SigningKeys implements stsv1.SigningKeyService: it derives per-scope SigV4
// signing keys from stored secrets so downstream services can verify request
// signatures locally. The raw secret never leaves this process; the derived
// key is bounded to one (access key, UTC day, region, service) scope.
type SigningKeys struct {
	dao      *models.DAO
	region   string
	service  string
	cacheTTL time.Duration

	// Now is overridable for tests. Defaults to time.Now.
	Now func() time.Time

	stsv1.UnimplementedSigningKeyServiceServer
}

// NewSigningKeys returns a SigningKeys server that issues keys only for the
// fleet-wide (region, service) scope and advises callers to re-fetch every
// cacheTTL, which bounds how long a disabled key keeps verifying downstream.
func NewSigningKeys(dao *models.DAO, region, service string, cacheTTL time.Duration) *SigningKeys {
	return &SigningKeys{dao: dao, region: region, service: service, cacheTTL: cacheTTL}
}

// GetSigningKey validates the requested scope, resolves the key and its
// owning user, and returns the derived signing key with caching bounds.
// Neither the secret nor the derived key is ever logged.
func (s *SigningKeys) GetSigningKey(ctx context.Context, req *stsv1.GetSigningKeyRequest) (*stsv1.GetSigningKeyResponse, error) {
	if req.GetAccessKeyId() == "" {
		return nil, twirp.RequiredArgumentError("access_key_id")
	}
	if req.GetRegion() == "" {
		return nil, twirp.RequiredArgumentError("region")
	}
	if req.GetService() == "" {
		return nil, twirp.RequiredArgumentError("service")
	}
	if !dateRe.MatchString(req.GetDate()) {
		return nil, twirp.InvalidArgumentError("date", "must be YYYYMMDD")
	}
	day, err := time.Parse("20060102", req.GetDate())
	if err != nil {
		return nil, twirp.InvalidArgumentError("date", "must be a real UTC date in YYYYMMDD form")
	}

	// Scope pinning: this deployment signs for exactly one (region, service)
	// pair, and a key is only useful for dates the verifier's clock-skew
	// window can actually accept — refuse to mint keys for anything else so a
	// compromised verifier credential cannot stockpile future-dated keys.
	if req.GetRegion() != s.region || req.GetService() != s.service {
		return nil, twirp.NewError(twirp.PermissionDenied, "signing keys are not issued for this region/service scope")
	}
	now := s.now()
	today := now.UTC().Truncate(24 * time.Hour)
	if d := day.Sub(today); d > 24*time.Hour || d < -24*time.Hour {
		return nil, twirp.NewError(twirp.PermissionDenied, "signing keys are not issued for this date")
	}

	k, err := s.dao.GetKeyWithUser(ctx, req.GetAccessKeyId())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, twirp.NotFoundError("unknown access key id")
		}
		return nil, twirp.InternalErrorWith(err)
	}
	// Disabled credentials are PERMISSION_DENIED, distinct from NOT_FOUND, so
	// the verifier can log the difference; its client-facing error is the
	// same for both.
	if k.DeletedAt.Valid {
		return nil, twirp.NewError(twirp.PermissionDenied, "access key is disabled")
	}
	if k.User == nil || k.User.DeletedAt.Valid {
		return nil, twirp.NewError(twirp.PermissionDenied, "owning user is disabled")
	}

	notValidAfter := day.AddDate(0, 0, 1).Add(maxClockSkew)
	cacheUntil := now.Add(s.cacheTTL)
	if cacheUntil.After(notValidAfter) {
		cacheUntil = notValidAfter
	}

	return &stsv1.GetSigningKeyResponse{
		SigningKey: sigv4.DeriveSigningKey(k.SecretAccessKey, req.GetDate(), req.GetRegion(), req.GetService()),
		Identity: &stsv1.TokenIdentity{
			AccessKeyId: k.AccessKeyID,
			PrincipalId: k.User.UUID,
			DisplayName: k.User.Name,
		},
		NotValidAfter: timestamppb.New(notValidAfter),
		CacheUntil:    timestamppb.New(cacheUntil),
	}, nil
}

func (s *SigningKeys) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}
```

Adjustment note: the test overrides `s.Now`; the handler must call `s.now()` exactly once per request (as above) so the window check and `cache_until` agree.

- [ ] **Step 4: Run the tests**

Run: `go test ./cmd/iamd/services/iam/sts/ -v`
Expected: PASS (new and old tests).

- [ ] **Step 5: Wire into main.go**

In `cmd/iamd/main.go`:

1. Add to the flag block:

```go
	signingKeyCacheTTL = flag.Duration("signing-key-cache-ttl", 5*time.Minute, "how long downstream verifiers may cache a derived signing key before re-fetching; bounds revocation latency")
```

(add `"time"` to imports.)

2. Extend `newMux` to take the TTL and serve the new route (keep the STSService route for now; Task 7 removes it):

```go
func newMux(lg *slog.Logger, dao *models.DAO, verifier *sigv4.Verifier, signingKeyCacheTTL time.Duration) *http.ServeMux {
```

and inside, after the existing STS route:

```go
	sk := sts.NewSigningKeys(dao, *region, *service, signingKeyCacheTTL)
	mux.Handle(stsv1.SigningKeyServicePathPrefix, stack(stsv1.NewSigningKeyServiceServer(sk, twirp.WithServerInterceptors(twirpslog.Interceptor(lg)))))
```

3. Update the call in `run`: `mux := newMux(lg, dao, verifier, *signingKeyCacheTTL)` and the call in `cmd/iamd/integration_test.go`: `newMux(quietLogger(), dao, verifier, 5*time.Minute)`.

Note: `newMux` reads the package-level `*region`/`*service` flags already (existing pattern); the TTL is passed as a parameter because tests call `newMux` directly.

- [ ] **Step 6: Build and run the iamd tests**

Run: `go build ./... && go test ./cmd/iamd/...`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add cmd/iamd/services/iam/sts/signingkey.go cmd/iamd/services/iam/sts/signingkey_test.go cmd/iamd/main.go cmd/iamd/integration_test.go
git commit --signoff -m "feat(iamd): serve SigV4 derived signing keys via SigningKeyService"
```

---

### Task 5: iamsts — rewrite as caching local verifier

**Files:**

- Rewrite: `web/middleware/sigv4/iamsts/iamsts.go`
- Rewrite: `web/middleware/sigv4/iamsts/iamsts_test.go`
- Modify: `twirp/twirpslog/twirpslog.go`

**Interfaces:**

- Consumes: `sigv4.Verifier{KeyLookup}`, `sigv4.SigningKeyLookuper`, `sigv4.ParseCredential`, `sigv4.TwirpError`, `sigv4.DeriveSigningKey` (tests), `stsv1.SigningKeyService` client (Task 3), `golang.org/x/sync/singleflight`.
- Produces:
  - `type Config struct { BaseURL string; HTTPClient *http.Client; Region, Service string; MaxBodySize int64; NegativeTTL time.Duration; Now func() time.Time }`
  - `func New(cfg Config) *Verifier`
  - `func (v *Verifier) Middleware(next http.Handler) http.Handler`, `func (v *Verifier) Verify(r *http.Request) (*Identity, error)`
  - `type Identity struct { AccessKeyID, OrganizationID, PrincipalID, DisplayName string; SignedAt time.Time }`
  - `func Caller(ctx context.Context) (*Identity, bool)` (name unchanged — twirpslog keeps compiling with a one-line field change).
- Deleted from the old file: `NewVerifier`, `Verifier.Client stsv1.STSService`, `toSTHeaders`, `verifyBody`, `toTwirpErr`, `ErrBodyHash`, `ErrBodyTooLarge` (the `sigv4` package's body checks and sentinels now cover the body; there is no separate body path).

- [ ] **Step 1: Replace iamsts.go**

Full new content of `web/middleware/sigv4/iamsts/iamsts.go`:

```go
// Package iamsts authenticates HTTP requests signed with AWS Signature
// Version 4 by verifying them locally against a cached derived signing key
// fetched from IAM's SigningKeyService.
//
// The verifying service never holds raw secret access keys. It fetches the
// SigV4 derived key HMAC(HMAC(HMAC(HMAC("AWS4"+secret, date), region),
// service), "aws4_request") for a credential scope once, caches it for the
// server-advised TTL, and recomputes/compares signatures itself. When the
// cache is warm, verification is a pure function of the request bytes and
// the cached key — no IAM RPC on the hot path.
//
// Caching rules follow the SigV4 spec: the key is uniquely scoped to the
// exact (access_key_id, YYYYMMDD, region, service) tuple from the request's
// Credential= component, entries honor the response's cache_until and are
// never kept past not_valid_after, concurrent misses for one scope collapse
// into a single RPC, and refusals (unknown or disabled keys) are cached
// briefly so a flood of bad credentials cannot hammer IAM.
//
// Authenticate the SigningKeyService client the same way as any other iamd
// caller: give Config.HTTPClient a sigv4client transport signing with the
// verifier's own IAM credential.
package iamsts

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/twitchtv/twirp"
	"golang.org/x/sync/singleflight"

	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	"within.website/x/web/middleware/sigv4"
)

// amzTimeFormat is the AWS X-Amz-Date timestamp format.
const amzTimeFormat = "20060102T150405Z"

// defaultNegativeTTL bounds how long a refusal (unknown or disabled key) is
// remembered before IAM is asked again.
const defaultNegativeTTL = 30 * time.Second

// Config configures a Verifier. BaseURL, HTTPClient, Region, and Service are
// required.
type Config struct {
	// BaseURL is the iamd endpoint serving SigningKeyService.
	BaseURL string

	// HTTPClient carries the GetSigningKey RPCs. It must authenticate to
	// iamd — typically a sigv4client transport signing with the verifier's
	// own IAM credential.
	HTTPClient *http.Client

	// Region and Service pin the credential scope incoming requests must be
	// signed for, exactly as on sigv4.Verifier.
	Region  string
	Service string

	// MaxBodySize caps the bytes buffered to verify the payload hash. Zero
	// means unlimited.
	MaxBodySize int64

	// NegativeTTL is how long a refusal is cached. Defaults to 30s.
	NegativeTTL time.Duration

	// Now is overridable for tests. Defaults to time.Now.
	Now func() time.Time
}

// Verifier authenticates requests locally using cached derived signing keys.
// It implements sigv4.SigningKeyLookuper against a SigningKeyService client.
type Verifier struct {
	client stsv1.SigningKeyService
	inner  *sigv4.Verifier
	negTTL time.Duration
	now    func() time.Time

	sf    singleflight.Group
	mu    sync.Mutex
	cache map[scopeKey]*entry
}

// scopeKey is the exact tuple a derived key is scoped to: the literal strings
// from the request's Credential= component, unnormalized.
type scopeKey struct {
	accessKeyID string
	date        string
	region      string
	service     string
}

func (k scopeKey) String() string {
	return strings.Join([]string{k.accessKeyID, k.date, k.region, k.service}, "\x00")
}

// entry is one cache slot: either a derived key with its identity, or a
// remembered refusal (err set). expiresAt of zero means "use once, do not
// serve from cache".
type entry struct {
	signingKey []byte
	identity   *stsv1.TokenIdentity
	err        error
	expiresAt  time.Time
}

// New returns a Verifier fetching derived keys from cfg.BaseURL.
func New(cfg Config) *Verifier {
	v := &Verifier{
		client: stsv1.NewSigningKeyServiceProtobufClient(cfg.BaseURL, cfg.HTTPClient),
		negTTL: cfg.NegativeTTL,
		now:    cfg.Now,
		cache:  make(map[scopeKey]*entry),
	}
	if v.negTTL == 0 {
		v.negTTL = defaultNegativeTTL
	}
	if v.now == nil {
		v.now = time.Now
	}
	v.inner = &sigv4.Verifier{
		Region:      cfg.Region,
		Service:     cfg.Service,
		MaxBodySize: cfg.MaxBodySize,
		KeyLookup:   v,
		Now:         cfg.Now,
	}
	return v
}

// Middleware wraps next so every request is verified locally against a cached
// signing key. On success the caller identity is stored in the request
// context (see Caller). Error mapping matches the local sigv4 middleware.
func (v *Verifier) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := v.Verify(r)
		if err != nil {
			slog.DebugContext(r.Context(), "iamsts: cannot verify request", "err", err, "method", r.Method, "path", r.URL.Path)
			twirp.WriteError(w, sigv4.TwirpError(r.Context(), err))
			return
		}
		next.ServeHTTP(w, r.WithContext(withCaller(r.Context(), id)))
	})
}

// Verify authenticates r locally: the inner sigv4 verifier performs every
// pre-check (clock skew, scope pinning, signed host, payload hash) and the
// constant-time signature comparison, pulling the derived key through this
// Verifier's cache. The request body is buffered and reset so downstream
// handlers can read it. On success it returns the caller identity.
func (v *Verifier) Verify(r *http.Request) (*Identity, error) {
	keyID, err := v.inner.Verify(r)
	if err != nil {
		return nil, err
	}

	// The signature checked out, so the credential parses and its scope
	// entry was just used; re-read it for the identity. In the unlikely case
	// it expired in between, this refetches — still off the hot path.
	cred, err := sigv4.ParseCredential(r.Header.Get("Authorization"))
	if err != nil {
		return nil, err
	}
	e, err := v.entry(r.Context(), scopeKey{cred.AccessKeyID, cred.Date, cred.Region, cred.Service})
	if err != nil {
		return nil, err
	}

	id := &Identity{
		AccessKeyID:    keyID,
		OrganizationID: e.identity.GetOrganizationId(),
		PrincipalID:    e.identity.GetPrincipalId(),
		DisplayName:    e.identity.GetDisplayName(),
	}
	if t, terr := time.Parse(amzTimeFormat, r.Header.Get("X-Amz-Date")); terr == nil {
		id.SignedAt = t
	}
	return id, nil
}

// LookupSigningKey implements sigv4.SigningKeyLookuper through the cache. The
// inner verifier only calls it after the clock-skew and scope checks pass, so
// unverifiable garbage never triggers an RPC.
func (v *Verifier) LookupSigningKey(ctx context.Context, accessKeyID, date, region, service string) ([]byte, error) {
	e, err := v.entry(ctx, scopeKey{accessKeyID, date, region, service})
	if err != nil {
		return nil, err
	}
	return e.signingKey, nil
}

// entry returns the cache slot for k, fetching it once (singleflight) on a
// miss. Remembered refusals return their error.
func (v *Verifier) entry(ctx context.Context, k scopeKey) (*entry, error) {
	now := v.now()
	v.mu.Lock()
	if e, ok := v.cache[k]; ok && now.Before(e.expiresAt) {
		v.mu.Unlock()
		if e.err != nil {
			return nil, e.err
		}
		return e, nil
	}
	v.mu.Unlock()

	res, err, _ := v.sf.Do(k.String(), func() (any, error) {
		return v.fetch(ctx, k)
	})
	if err != nil {
		return nil, err
	}
	e := res.(*entry)
	if e.err != nil {
		return nil, e.err
	}
	return e, nil
}

// fetch performs the GetSigningKey RPC and stores the result. NOT_FOUND and
// PERMISSION_DENIED become cached refusals surfacing as ErrUnknownKey, so a
// probe cannot distinguish unknown from disabled credentials; the distinction
// is logged here. Any other failure (iamd down, transport fault) is returned
// uncached and surfaces as an internal error — an IAM outage must read as an
// outage, not as a denial. Signing key bytes are never logged.
func (v *Verifier) fetch(ctx context.Context, k scopeKey) (*entry, error) {
	resp, err := v.client.GetSigningKey(ctx, &stsv1.GetSigningKeyRequest{
		AccessKeyId: k.accessKeyID,
		Date:        k.date,
		Region:      k.region,
		Service:     k.service,
	})
	if err != nil {
		var te twirp.Error
		if errors.As(err, &te) {
			switch te.Code() {
			case twirp.NotFound, twirp.PermissionDenied:
				slog.InfoContext(ctx, "iamsts: signing key refused",
					"code", string(te.Code()),
					"access_key_id", k.accessKeyID,
					"date", k.date,
				)
				e := &entry{err: sigv4.ErrUnknownKey, expiresAt: v.now().Add(v.negTTL)}
				v.store(k, e)
				return e, nil
			}
		}
		return nil, err
	}

	e := &entry{
		signingKey: resp.GetSigningKey(),
		identity:   resp.GetIdentity(),
	}
	// Honor the server's caching bounds exactly: cache_until is the TTL,
	// clamped by not_valid_after. A response without cache_until is used for
	// this request but never cached — the server declined to grant a TTL and
	// we do not invent one.
	if cu := resp.GetCacheUntil(); cu != nil {
		e.expiresAt = cu.AsTime()
		if nva := resp.GetNotValidAfter(); nva != nil && nva.AsTime().Before(e.expiresAt) {
			e.expiresAt = nva.AsTime()
		}
	}
	if e.expiresAt.After(v.now()) {
		v.store(k, e)
	}
	return e, nil
}

// store inserts e and sweeps expired slots, so scopes for rolled-over dates
// do not accumulate: at UTC midnight every cached key expires and the next
// insert evicts the stale day.
func (v *Verifier) store(k scopeKey, e *entry) {
	now := v.now()
	v.mu.Lock()
	defer v.mu.Unlock()
	for old, oe := range v.cache {
		if !now.Before(oe.expiresAt) {
			delete(v.cache, old)
		}
	}
	v.cache[k] = e
}

// cacheLen reports the live slot count, for tests.
func (v *Verifier) cacheLen() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return len(v.cache)
}

// Identity is the verified caller stored in the request context on success.
// The fields come from the SigningKeyService response identity; SignedAt is
// parsed from the request's X-Amz-Date.
type Identity struct {
	AccessKeyID    string
	OrganizationID string
	PrincipalID    string
	DisplayName    string
	SignedAt       time.Time
}

type ctxKey struct{}

func withCaller(ctx context.Context, c *Identity) context.Context {
	return context.WithValue(ctx, ctxKey{}, c)
}

// Caller returns the verified identity stored by Middleware, if any.
func Caller(ctx context.Context) (*Identity, bool) {
	c, ok := ctx.Value(ctxKey{}).(*Identity)
	return c, ok
}
```

- [ ] **Step 2: Adapt twirpslog**

In `twirp/twirpslog/twirpslog.go`, change the caller-attribution branch:

```go
			} else if caller, ok := iamsts.Caller(ctx); ok {
				userID = caller.PrincipalID
			}
```

(the comment above it stays accurate; `PrincipalID` carries the same user UUID `caller.User.GetId()` did.)

- [ ] **Step 3: Rewrite the unit tests**

Replace `web/middleware/sigv4/iamsts/iamsts_test.go` entirely. The stub serves real derived keys over a real Twirp round trip and counts RPCs; requests are signed by the AWS SDK v4 signer (the reference implementation) at controllable times:

```go
package iamsts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	signer "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/timestamppb"

	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	"within.website/x/web/middleware/sigv4"
)

const (
	testKey    = "AKIDEXAMPLE"
	testSecret = "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	testRegion = "us-east-1"
	testSvc    = "execute-api"
	emptySHA   = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
)

// fakeKeys is a SigningKeyService stub served over a real Twirp endpoint. It
// derives real keys for testKey/testSecret, counts calls, and can be flipped
// to refuse.
type fakeKeys struct {
	mu         sync.Mutex
	calls      atomic.Int64
	refuseCode twirp.ErrorCode // when non-empty, refuse with this code
	cacheTTL   time.Duration
	now        func() time.Time
}

func (f *fakeKeys) setRefuse(code twirp.ErrorCode) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refuseCode = code
}

func (f *fakeKeys) GetSigningKey(_ context.Context, req *stsv1.GetSigningKeyRequest) (*stsv1.GetSigningKeyResponse, error) {
	f.calls.Add(1)
	f.mu.Lock()
	code := f.refuseCode
	f.mu.Unlock()
	if code != "" {
		return nil, twirp.NewError(code, "refused")
	}
	if req.GetAccessKeyId() != testKey {
		return nil, twirp.NotFoundError("unknown access key id")
	}
	now := f.now()
	day, err := time.Parse("20060102", req.GetDate())
	if err != nil {
		return nil, twirp.InvalidArgumentError("date", "bad date")
	}
	nva := day.AddDate(0, 0, 1).Add(15 * time.Minute)
	cu := now.Add(f.cacheTTL)
	if cu.After(nva) {
		cu = nva
	}
	return &stsv1.GetSigningKeyResponse{
		SigningKey: sigv4.DeriveSigningKey(testSecret, req.GetDate(), req.GetRegion(), req.GetService()),
		Identity: &stsv1.TokenIdentity{
			AccessKeyId: req.GetAccessKeyId(),
			PrincipalId: "u1",
			DisplayName: "tester",
		},
		NotValidAfter: timestamppb.New(nva),
		CacheUntil:    timestamppb.New(cu),
	}, nil
}

// harness wires a fakeKeys stub, a clock, and a Verifier-wrapped test handler.
type harness struct {
	fake     *fakeKeys
	verifier *Verifier
	handler  http.Handler
	now      time.Time
	mu       sync.Mutex
}

func (h *harness) clock() time.Time {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.now
}

func (h *harness) advance(d time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.now = h.now.Add(d)
}

func newHarness(t *testing.T, cacheTTL time.Duration) (*harness, func()) {
	t.Helper()
	h := &harness{now: time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)}
	h.fake = &fakeKeys{cacheTTL: cacheTTL, now: h.clock}
	srv := httptest.NewServer(stsv1.NewSigningKeyServiceServer(h.fake))
	h.verifier = New(Config{
		BaseURL:     srv.URL,
		HTTPClient:  http.DefaultClient,
		Region:      testRegion,
		Service:     testSvc,
		MaxBodySize: 1 << 20,
		Now:         h.clock,
	})
	h.handler = h.verifier.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	return h, srv.Close
}

// signedGET returns a bodyless GET signed at ts by the AWS SDK signer.
func signedGET(t *testing.T, ts time.Time) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "https://svc.example.com/things?a=1", nil)
	req.Header.Set("X-Amz-Content-Sha256", emptySHA)
	if err := signer.NewSigner().SignHTTP(context.Background(),
		aws.Credentials{AccessKeyID: testKey, SecretAccessKey: testSecret},
		req, emptySHA, testSvc, testRegion, ts); err != nil {
		t.Fatalf("SignHTTP: %v", err)
	}
	return req
}

func do(h *harness, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	return rec
}

func TestVerifier_CacheHit(t *testing.T) {
	h, closeSrv := newHarness(t, 5*time.Minute)
	defer closeSrv()

	if rec := do(h, signedGET(t, h.clock())); rec.Code != http.StatusNoContent {
		t.Fatalf("first request: status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if got := h.fake.calls.Load(); got != 1 {
		t.Fatalf("after miss: %d RPCs, want 1", got)
	}
	// Warm path: zero additional RPCs.
	for range 5 {
		if rec := do(h, signedGET(t, h.clock())); rec.Code != http.StatusNoContent {
			t.Fatalf("warm request: status = %d", rec.Code)
		}
	}
	if got := h.fake.calls.Load(); got != 1 {
		t.Errorf("after warm requests: %d RPCs, want 1 (hot path must not call IAM)", got)
	}
}

func TestVerifier_IdentityThreaded(t *testing.T) {
	h, closeSrv := newHarness(t, 5*time.Minute)
	defer closeSrv()

	var got *Identity
	h.handler = h.verifier.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = Caller(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	if rec := do(h, signedGET(t, h.clock())); rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
	if got == nil || got.AccessKeyID != testKey || got.PrincipalID != "u1" || got.DisplayName != "tester" {
		t.Fatalf("caller = %+v, want key %s principal u1", got, testKey)
	}
	if got.SignedAt.IsZero() {
		t.Error("SignedAt not populated")
	}
}

func TestVerifier_MidnightRollover(t *testing.T) {
	h, closeSrv := newHarness(t, time.Hour)
	defer closeSrv()

	if rec := do(h, signedGET(t, h.clock())); rec.Code != http.StatusNoContent {
		t.Fatalf("day-1 request: status = %d", rec.Code)
	}

	// Cross UTC midnight: the scope date changes, so the old key cannot
	// serve the new day — a fresh fetch must happen and the old slot must be
	// swept.
	h.advance(13 * time.Hour) // 12:00 -> 01:00 next day
	if rec := do(h, signedGET(t, h.clock())); rec.Code != http.StatusNoContent {
		t.Fatalf("day-2 request: status = %d, body=%s", rec.Code, rec.Body.String())
	}
	if got := h.fake.calls.Load(); got != 2 {
		t.Errorf("RPCs = %d, want 2 (one per scope date)", got)
	}
	if got := h.verifier.cacheLen(); got != 1 {
		t.Errorf("cache slots = %d, want 1 (day-1 key evicted)", got)
	}
}

func TestVerifier_RevocationAfterCacheUntil(t *testing.T) {
	h, closeSrv := newHarness(t, time.Minute)
	defer closeSrv()

	if rec := do(h, signedGET(t, h.clock())); rec.Code != http.StatusNoContent {
		t.Fatalf("initial request: status = %d", rec.Code)
	}

	// Key gets disabled server-side. Within cache_until the old key still
	// verifies (bounded staleness)...
	h.fake.setRefuse(twirp.PermissionDenied)
	if rec := do(h, signedGET(t, h.clock())); rec.Code != http.StatusNoContent {
		t.Fatalf("within TTL: status = %d, want 204 (cached)", rec.Code)
	}

	// ...and after cache_until expires, the refetch sees the refusal and the
	// request fails like an unknown key.
	h.advance(2 * time.Minute)
	if rec := do(h, signedGET(t, h.clock())); rec.Code != http.StatusForbidden {
		t.Fatalf("after TTL: status = %d, want 403", rec.Code)
	}
	if got := h.fake.calls.Load(); got != 2 {
		t.Errorf("RPCs = %d, want 2 (refetch after cache_until)", got)
	}
}

func TestVerifier_NegativeCache(t *testing.T) {
	h, closeSrv := newHarness(t, 5*time.Minute)
	defer closeSrv()
	h.fake.setRefuse(twirp.NotFound)

	for range 5 {
		if rec := do(h, signedGET(t, h.clock())); rec.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", rec.Code)
		}
	}
	if got := h.fake.calls.Load(); got != 1 {
		t.Errorf("RPCs = %d, want 1 (refusal cached ~30s)", got)
	}

	// The negative entry expires and IAM is asked again.
	h.advance(time.Minute)
	do(h, signedGET(t, h.clock()))
	if got := h.fake.calls.Load(); got != 2 {
		t.Errorf("RPCs = %d, want 2 after negative TTL", got)
	}
}

func TestVerifier_Singleflight(t *testing.T) {
	h, closeSrv := newHarness(t, 5*time.Minute)
	defer closeSrv()

	const n = 8
	var wg sync.WaitGroup
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			do(h, signedGET(t, h.clock()))
		}()
	}
	wg.Wait()
	// All n concurrent misses for one scope must collapse into one RPC.
	if got := h.fake.calls.Load(); got != 1 {
		t.Errorf("RPCs = %d, want 1 (singleflight)", got)
	}
}

func TestVerifier_SignatureMismatch(t *testing.T) {
	h, closeSrv := newHarness(t, 5*time.Minute)
	defer closeSrv()

	req := signedGET(t, h.clock())
	req.URL.RawQuery = "a=2" // change what was signed
	if rec := do(h, req); rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (SignatureDoesNotMatch-equivalent)", rec.Code)
	}
}

func TestVerifier_SkewRejectedWithoutRPC(t *testing.T) {
	h, closeSrv := newHarness(t, 5*time.Minute)
	defer closeSrv()

	req := signedGET(t, h.clock().Add(-time.Hour)) // outside ±15m
	rec := do(h, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (RequestTimeTooSkewed-equivalent)", rec.Code)
	}
	if got := h.fake.calls.Load(); got != 0 {
		t.Errorf("RPCs = %d, want 0 (skew check precedes key lookup)", got)
	}
}

func TestVerifier_IAMOutageIs500(t *testing.T) {
	h, closeSrv := newHarness(t, 5*time.Minute)
	// Kill the backend before the first fetch: the failure must read as an
	// outage, not a denial, and must not be cached.
	closeSrv()

	rec := do(h, signedGET(t, h.clock()))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

// TestVerifier_KnownAnswerFixture replays the worked example from the AWS
// SigV4 documentation (GET iam.amazonaws.com ListUsers, AKIDEXAMPLE,
// 20150830T123600Z) whose final signature AWS publishes, so the full
// canonicalization + derivation + comparison pipeline is checked against
// AWS's own reference vector rather than our own signer.
func TestVerifier_KnownAnswerFixture(t *testing.T) {
	h := &harness{now: time.Date(2015, 8, 30, 12, 36, 0, 0, time.UTC)}
	h.fake = &fakeKeys{cacheTTL: 5 * time.Minute, now: h.clock}
	srv := httptest.NewServer(stsv1.NewSigningKeyServiceServer(h.fake))
	defer srv.Close()
	v := New(Config{
		BaseURL:    srv.URL,
		HTTPClient: http.DefaultClient,
		Region:     "us-east-1",
		Service:    "iam",
		Now:        h.clock,
	})

	req := httptest.NewRequest(http.MethodGet, "https://iam.amazonaws.com/?Action=ListUsers&Version=2010-05-08", nil)
	req.Host = "iam.amazonaws.com"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	req.Header.Set("X-Amz-Date", "20150830T123600Z")
	req.Header.Set("Authorization", strings.Join([]string{
		"AWS4-HMAC-SHA256 Credential=AKIDEXAMPLE/20150830/us-east-1/iam/aws4_request",
		" SignedHeaders=content-type;host;x-amz-date",
		" Signature=5d672d79c15b13162d9279b0855cfba6789a8edb4c82c400e06b5924a6f2b5d7",
	}, ","))

	id, err := v.Verify(req)
	if err != nil {
		t.Fatalf("Verify(AWS doc fixture): %v", err)
	}
	if id.AccessKeyID != "AKIDEXAMPLE" {
		t.Errorf("AccessKeyID = %q", id.AccessKeyID)
	}
}
```

- [ ] **Step 4: Run the tests**

Run: `go test ./web/middleware/sigv4/iamsts/ ./twirp/twirpslog/ -v`
Expected: unit tests PASS. `e2e_test.go` in iamsts will now fail to compile (it references the deleted `NewVerifier`/old flow) — that file is rewritten in the next step of this task, not deferred: rewrite it now so the package is green before committing.

- [ ] **Step 5: Rewrite the e2e test**

Replace `web/middleware/sigv4/iamsts/e2e_test.go`:

```go
package iamsts

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	"within.website/x/web/middleware/sigv4"
	"within.website/x/web/middleware/sigv4/sigv4client"
)

// TestEndToEnd_SigV4ClientToLocalVerify drives the full chain in its
// deployment shape: sigv4client (AWS SDK signer) signs an outgoing request,
// the iamsts middleware verifies it locally with a derived key fetched over a
// real Twirp round trip from a SigningKeyService stub that derives real keys.
// Any canonicalization disagreement between signer and verifier fails here.
func TestEndToEnd_SigV4ClientToLocalVerify(t *testing.T) {
	fake := &fakeKeys{cacheTTL: 5 * time.Minute, now: time.Now}
	keySrv := httptest.NewServer(stsv1.NewSigningKeyServiceServer(fake))
	defer keySrv.Close()

	v := New(Config{
		BaseURL:     keySrv.URL,
		HTTPClient:  http.DefaultClient,
		Region:      testRegion,
		Service:     testSvc,
		MaxBodySize: 1 << 20,
	})

	var gotBody string
	var gotCaller *Identity
	app := httptest.NewServer(v.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotCaller, _ = Caller(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})))
	defer app.Close()

	rt, err := sigv4client.NewSigV4RoundTripper(&sigv4client.Config{
		Region:      testRegion,
		AccessKey:   testKey,
		SecretKey:   testSecret,
		ServiceName: testSvc,
	}, nil)
	if err != nil {
		t.Fatalf("round tripper: %v", err)
	}
	client := &http.Client{Transport: rt}

	t.Run("signed POST with body verifies and body survives", func(t *testing.T) {
		resp, err := client.Post(app.URL+"/things?a=1&b=2", "application/json", strings.NewReader(`{"x":1}`))
		if err != nil {
			t.Fatalf("POST: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			body, _ := io.ReadAll(resp.Body)
			t.Fatalf("status = %d, body=%s", resp.StatusCode, body)
		}
		if gotBody != `{"x":1}` {
			t.Errorf("downstream body = %q", gotBody)
		}
		if gotCaller == nil || gotCaller.PrincipalID != "u1" {
			t.Errorf("caller = %+v, want principal u1", gotCaller)
		}
	})

	t.Run("unsigned request rejected", func(t *testing.T) {
		resp, err := http.Get(app.URL + "/things")
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})

	t.Run("tampered body rejected", func(t *testing.T) {
		// Sign one request, then replay its headers over a different body.
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, app.URL+"/things", strings.NewReader(`{"x":1}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("signed POST: %v", err)
		}
		resp.Body.Close()

		forged, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, app.URL+"/things", strings.NewReader(`{"x":2}`))
		// sigv4client clones and signs per attempt, so forge manually: reuse
		// a stale signature against new bytes via a plain client.
		forged.Header = resp.Request.Header.Clone()
		plainResp, err := http.DefaultClient.Do(forged)
		if err != nil {
			t.Fatalf("forged POST: %v", err)
		}
		defer plainResp.Body.Close()
		if plainResp.StatusCode != http.StatusForbidden {
			t.Fatalf("status = %d, want 403 (body swap must not verify)", plainResp.StatusCode)
		}
	})
}
```

(`fakeKeys` needs `now` to work when the harness clock isn't used — the struct literal above passes `time.Now` directly, which matches the field type `func() time.Time`.)

- [ ] **Step 6: Run the full package**

Run: `go test ./web/middleware/sigv4/... ./twirp/... -v`
Expected: PASS.

- [ ] **Step 7: Audit log lines**

Run: `grep -n "slog\." web/middleware/sigv4/iamsts/iamsts.go cmd/iamd/services/iam/sts/signingkey.go`
Expected: no line logs `signingKey`, `SigningKey`, `secret`, or entry contents — only error values, access key ids, dates, and codes. Fix anything that does before committing.

- [ ] **Step 8: Commit**

```bash
git add web/middleware/sigv4/iamsts/ twirp/twirpslog/twirpslog.go
git commit --signoff -m "feat(iamsts)!: verify locally with cached derived signing keys

Replaces per-request STS GetCallerIdentity RPCs with local SigV4
verification against derived signing keys fetched once per credential
scope from SigningKeyService and cached per the server-advised TTL.

BREAKING CHANGE: iamsts.NewVerifier is replaced by iamsts.New(Config);
iamsts.Identity now carries TokenIdentity fields (PrincipalID) instead
of an iamv1.User."
```

---

### Task 6: iamd integration test for the full chain

**Files:**

- Modify: `cmd/iamd/integration_test.go` (append)

**Interfaces:**

- Consumes: `newMux` (Task 4 signature), `iamsts.New`/`iamsts.Caller` (Task 5), existing helpers `newDAO`, `bootstrapCreds`, `signedTransport`.

- [ ] **Step 1: Write the test**

Append to `cmd/iamd/integration_test.go` (add `"time"`, `stsv1` is not needed; import `"within.website/x/web/middleware/sigv4/iamsts"`):

```go
// TestIntegration_SigningKeyVerifierChain drives the new deployment shape end
// to end against a real iamd: a downstream service verifies incoming SigV4
// requests locally with derived keys fetched from iamd's SigningKeyService,
// authenticating that fetch with its own IAM credential.
func TestIntegration_SigningKeyVerifierChain(t *testing.T) {
	dao := newDAO(t)
	verifierAKID, verifierSecret := bootstrapCreds(t, dao)

	// A second identity acts as the end user calling the downstream service.
	ctx := context.Background()
	endUser, err := dao.CreateUser(ctx, "end-user")
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	endKey, err := dao.CreateKey(ctx, endUser, "end user key")
	if err != nil {
		t.Fatalf("CreateKey: %v", err)
	}

	verifier := newVerifier(dao, intRegion, intService, 1<<20)
	iamd := httptest.NewServer(newMux(quietLogger(), dao, verifier, 5*time.Minute))
	defer iamd.Close()

	// The downstream service: fetches signing keys from iamd (signing those
	// fetches with its own credential) and verifies end-user requests locally.
	downstreamVerifier := iamsts.New(iamsts.Config{
		BaseURL:     iamd.URL,
		HTTPClient:  &http.Client{Transport: signedTransport(t, verifierAKID, verifierSecret)},
		Region:      intRegion,
		Service:     intService,
		MaxBodySize: 1 << 20,
	})
	var gotPrincipal string
	app := httptest.NewServer(downstreamVerifier.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, ok := iamsts.Caller(r.Context()); ok {
			gotPrincipal = c.PrincipalID
		}
		w.WriteHeader(http.StatusNoContent)
	})))
	defer app.Close()

	endUserClient := &http.Client{Transport: signedTransport(t, endKey.AccessKeyID, endKey.SecretAccessKey)}

	t.Run("end user verified locally", func(t *testing.T) {
		resp, err := endUserClient.Get(app.URL + "/resource")
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", resp.StatusCode)
		}
		if gotPrincipal != endUser.UUID {
			t.Errorf("principal = %q, want %q", gotPrincipal, endUser.UUID)
		}
	})

	t.Run("disabled key stops verifying after cache expiry", func(t *testing.T) {
		if err := dao.DisableKey(ctx, endKey.AccessKeyID, "compromised", ""); err != nil {
			t.Fatalf("DisableKey: %v", err)
		}
		// The cached key may still verify (bounded staleness by design); a
		// fresh verifier simulates post-TTL behavior deterministically.
		fresh := iamsts.New(iamsts.Config{
			BaseURL:    iamd.URL,
			HTTPClient: &http.Client{Transport: signedTransport(t, verifierAKID, verifierSecret)},
			Region:     intRegion,
			Service:    intService,
		})
		freshApp := httptest.NewServer(fresh.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})))
		defer freshApp.Close()
		resp, err := endUserClient.Get(freshApp.URL + "/resource")
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("status = %d, want 403 (disabled key)", resp.StatusCode)
		}
	})
}
```

- [ ] **Step 2: Run it**

Run: `go test ./cmd/iamd/ -run TestIntegration -v`
Expected: PASS (both the new chain test and the pre-existing round-trip test).

- [ ] **Step 3: Commit**

```bash
git add cmd/iamd/integration_test.go
git commit --signoff -m "test(iamd): cover the signing-key local-verification chain end to end"
```

---

### Task 7: delete the legacy verify flow

**Files:**

- Modify: `pb/within/website/x/iam/sts/v1/sts.proto` (remove `STSService`, `GetCallerIdentityReq`, `GetCallerIdentityResp`, `Header`; rewrite the file-top doc comment for SigningKeyService)
- Delete content: `cmd/iamd/services/iam/sts/sts.go` (delete file; `signingkey.go` keeps the package), `cmd/iamd/services/iam/sts/sts_test.go` (delete file — its coverage of skew/scope/tamper/disabled behavior now lives in the sigv4 package tests, iamsts unit tests, and `signingkey_test.go`)
- Modify: `cmd/iamd/main.go` (remove the `STSService` route and the stale "One verifier backs …" comment half about STS handler checks)
- Modify: `web/middleware/sigv4/sigv4.go` (delete `VerifySignature`)
- Modify: `web/middleware/sigv4/sigv4_test.go` (delete `TestVerifySignature`, `TestVerifySignature_UnsignedPayload`, `TestVerifySignature_DoubleSlashPath`)
- Generated: `gen/within/website/x/iam/sts/v1/*` (regenerate)

Coverage note before deleting the `VerifySignature` tests: they pinned bodyless verification, the unsigned-payload sentinel, and double-slash path handling. The first two are covered by `Verify`-path tests that already exist (`resolvePayloadHash` tests) plus the new KeyLookup e2e; **double-slash path handling is only covered there** — `VerifySignature` built a synthetic request from a forwarded path string, which is exactly the code being deleted, so the hazard it guarded (url.Parse mis-parsing `//foo`) leaves with it. `Verify` reads `r.URL` from the server's own parser. No port needed; note it in the commit message.

- [ ] **Step 1: Remove the proto service**

In `pb/within/website/x/iam/sts/v1/sts.proto`, delete the `STSService` service block, `GetCallerIdentityReq`, `Header`, and `GetCallerIdentityResp` messages, and the file-top comment describing central validation (SigningKeyService's own comment already describes the new model). The remaining file: syntax/package/imports + the SigningKeyService content from Task 3. Drop the `within/website/x/iam/v1/iam.proto` import if nothing in the file still references `within.website.x.iam.v1` types (after this deletion, nothing does).

- [ ] **Step 2: Delete the Go code**

```bash
git rm cmd/iamd/services/iam/sts/sts.go cmd/iamd/services/iam/sts/sts_test.go
```

In `cmd/iamd/main.go`: remove the two lines wiring `stsSvc` (`stsSvc := sts.New(dao, verifier)` and its `mux.Handle`), and fix the comment above `newVerifier` usage in `run` ("One verifier backs the route middleware … and the STS handler's bodyless end-user checks" → the middleware is now its only consumer). `signingkey_test.go` referenced a `newDAO` helper that lived in `sts_test.go` — move that helper into `signingkey_test.go` when deleting.

In `web/middleware/sigv4/sigv4.go`: delete the `VerifySignature` method (lines starting at the `// VerifySignature verifies a SigV4 signature over request material` comment through the end of the method). Remove now-unused imports (`net/url`, `strconv` — verify with the compiler).

In `web/middleware/sigv4/sigv4_test.go`: delete the three `TestVerifySignature*` functions and any helpers used only by them.

`cmd/iamd/services/iam/sts/signingkey.go`'s package comment is now the package's only doc — give the package clause a proper comment:

```go
// Package sts hosts iamd's security-token-service surface: the
// SigningKeyService that distributes SigV4 derived signing keys to
// downstream verifiers.
package sts
```

(and remove the old package comment from wherever it lived.)

- [ ] **Step 3: Regenerate and verify nothing survives**

```bash
npm run generate
go build ./...
grep -rn "GetCallerIdentity\|STSService" --include='*.go' --include='*.proto' . | grep -v node_modules
```

Expected: build passes; the grep prints **zero** lines — including under `gen/` (buf regenerates `sts.pb.go`/`sts.twirp.go`/`sts_grpc.pb.go`/`stsv1connect` in place from the trimmed proto; if any stale generated file still mentions `STSService`, delete it and rerun `npm run generate`).

- [ ] **Step 4: Full test suite**

Run: `npm test`
Expected: PASS across the repo.

- [ ] **Step 5: Commit**

```bash
git add -A pb/ gen/ cmd/iamd/ web/middleware/sigv4/
git commit --signoff -m "refactor(iam)!: delete the per-request STS verification flow

Removes STSService/GetCallerIdentity (proto, generated stubs, iamd
handler, and sigv4.VerifySignature) now that downstream services verify
locally with cached derived signing keys.

The deleted double-slash-path test guarded VerifySignature's synthetic
request construction, which is removed with it; the local path reads
r.URL from net/http's own parser.

BREAKING CHANGE: the STSService Twirp/gRPC/Connect APIs no longer
exist; downstream verifiers must upgrade to iamsts.New + SigningKeyService."
```

---

### Task 8: docs + final verification

**Files:**

- Modify: `docs/plans/2026-06-29-sigv4-auth.md`

- [ ] **Step 1: Update the design doc**

`docs/plans/2026-06-29-sigv4-auth.md` documents central validation as the chosen architecture. Update it: in "The architecture decision" section, add a dated addendum noting the decision was revised on 2026-07-06 — per-request central validation put an RPC on the hot path and made iamd a synchronous auth dependency under load; the replacement is derived-signing-key caching (bounded secret exposure: one scope, one UTC day), with revocation latency bounded by `-signing-key-cache-ttl`. Update the Components table row for `iamsts` ("Fetches derived signing keys from SigningKeyService, caches per scope, verifies locally") and the STS contract row (`GetSigningKey` RPC). Update Wiring & usage to show `iamsts.New(iamsts.Config{...})`. Do not rewrite history elsewhere in the doc; append status.

- [ ] **Step 2: Final verification sweep**

```bash
go build ./...
npm test
grep -rn "GetCallerIdentity\|STSService" --include='*.go' --include='*.proto' . | grep -v node_modules
grep -rn "slog\." web/middleware/sigv4/iamsts/ cmd/iamd/services/iam/sts/ | grep -iv test
```

Expected: build + tests green; first grep zero hits; second grep shows no log line carrying key material.

- [ ] **Step 3: Write the latency summary (deliverable to the user, not a file)**

Report: RPCs on the request path — **zero when warm**. Cold path: exactly one `GetSigningKey` RPC per `(access_key_id, date, region, service)` tuple per `cache_until` window, singleflighted across concurrent requests; skewed/mis-scoped/malformed requests never trigger an RPC at all (pre-checks run before key lookup); refused credentials cost one RPC per 30s per tuple.

- [ ] **Step 4: Commit**

```bash
git add docs/plans/2026-06-29-sigv4-auth.md
git commit --signoff -m "docs(sigv4): record the move to derived signing key caching"
```
