# SigV4A Middleware Implementation Review

## Overall Verdict

The AWS Signature Version 4A (SigV4A) middleware implementation in `web/middleware/sigv4a/` is **correct, secure, and well-tested**. No bugs, security gaps, or spec deviations were found. The signer and verifier agree by construction through the shared `web/middleware/internal/awssig` canonicalization package. All tests pass, including with `-race` and `go vet` clean.

## Review Scope

The review covered all files under `web/middleware/sigv4a/`:

| File                                                                                                                                                   | Purpose                                                                                                 |
| ------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------- |
| `sigv4a.go`                                                                                                                                            | Verifier, Middleware, Verify, verify, canonicalRequest, parseAuthHeader, ParseCredential, error mapping |
| `sign.go`                                                                                                                                              | Signer, Sign, sign (client-side signing)                                                                |
| `keyderivation.go`                                                                                                                                     | DeriveKeyPair (ECDSA P-256 keypair derivation)                                                          |
| `lookuper.go`                                                                                                                                          | Lookuper, PublicKeyLookuper interfaces                                                                  |
| `regionset.go`                                                                                                                                         | regionSetMatches (X-Amz-Region-Set wildcard matching)                                                   |
| `context.go`                                                                                                                                           | authctx helpers (KeyID/User/WithUser/WithCaller)                                                        |
| `iamsts/iamsts.go`                                                                                                                                     | Cached public-key verifier with singleflight                                                            |
| `sigv4aclient/sigv4aclient.go`                                                                                                                         | Signing RoundTripper                                                                                    |
| `web/middleware/internal/awssig/`                                                                                                                      | Shared canonicalization + payload hash                                                                  |
| Tests: `sigv4a_test.go`, `keyderivation_test.go`, `iamsts/iamsts_test.go`, `iamsts/e2e_test.go`, `sigv4aclient/sigv4aclient_test.go`, `testdata/v4a/*` | Test coverage                                                                                           |

## Findings by Area

### 1. Key derivation — CORRECT

**`web/middleware/sigv4a/keyderivation.go`** faithfully implements the aws-c-auth SigV4A KDF (verified line-for-line against `source/key_derivation.c`):

- Input layout: `BE32(1) || "AWS4-ECDSA-P256-SHA256" || 0x00 || accessKeyID || counterByte || BE32(256)` (lines 48-53) — exact match.
- HMAC key `"AWS4A" + secret`, PRF = HMAC-SHA256 via `awssig.HMACSHA256` (line 43, `awssig.go:34-38`) — exact.
- Counter range 1..254 with rejection sampling against `N-2`; accepted candidate `+1` landing in `[1, N-1]` (lines 44, 56-59) — exact.
- Scalar → P-256 keypair via `ecdh.P256().NewPrivateKey` (validates scalar ∈ [1, N−1], derives point) (line 64) — exact.
- Uncompressed SEC1 public bytes split `X = pub[1:33]`, `Y = pub[33:65]` (lines 68-74) — exact.

**Tests:** `TestDeriveKeyPair_FixedVector` (fixed KDF vector from aws-c-auth) and `TestDeriveKeyPair_Vectors` (all copied aws-c-auth vector public keys) pass.

**Nits (not bugs):**

- Comment line 30 states per-attempt rejection probability is ~2⁻³²; for P-256 it is actually ~2⁻¹⁰¹. Harmless documentation error.
- Non-constant-time `big.Int` comparison (line 56) is explicitly justified (lines 39-41): derivation runs only over stored secrets, never attacker-controlled input.

### 2. Signing — CORRECT

**`web/middleware/sigv4a/sign.go`:**

- Signed headers: `host`, `x-amz-content-sha256`, `x-amz-date`, `x-amz-region-set` (line 53).
- Canonical request over sorted signed headers via `awssig.BuildCanonicalRequest` (lines 72-75).
- Scope = `date/service/aws4_request` (no region) (line 78).
- `stringToSign = algorithm\namzDate\nscope\nhashed(canonicalRequest)` (lines 80-85).
- `ECDSA-SignASN1` over `sha256(stringToSign)` (line 88).
- `Sign` does not touch `r.Body`; `sign` is split from `Sign` so tests can reproduce AWS vectors that omit `x-amz-content-sha256`.

**Interop:** The verifier uses the same `awssig.BuildCanonicalRequest` with `disablePathEscaping=false` (non-S3 within.website services), so signer↔verifier agree by construction. End-to-end interop validated by `TestSigner_Vectors`, `TestRoundTrip_SignVerify`, and `iamsts/e2e_test.go`.

### 3. Verification — CORRECT

**`web/middleware/sigv4a/sigv4a.go` (`verify`, lines 180-280):**

- Scope pinning: service match (line 193), date vs X-Amz-Date (line 229).
- Signed `host` required (line 200).
- Signed `x-amz-region-set` required + `regionSetMatches` (lines 206-211).
- Clock skew vs default 15 min (lines 232-234).
- `X-Amz-Date` with `Date` header fallback (lines 213-224).
- Payload hash via `awssig.ResolvePayloadHash` in `Verify`; `verify` never touches `r.Body` (doc at line 179, confirmed by code).
- Public key via `KeyLookup` (precedence) or `Lookup`+`DeriveKeyPair` (lines 236-249).
- **Nil-key panic guard** (lines 253-259): a lookuper returning `(nil, nil)` → `ErrUnknownKey`, never reaches `ecdsa.VerifyASN1` which would panic on a nil key.
- `ECDSA-VerifyASN1` (line 276).
- Body buffered and reset in `Verify` (lines 163-171).

**Observation (not a security issue):** The verifier requires `host` and `x-amz-region-set` signed but does not require `x-amz-date` or `x-amz-content-sha256` to be signed. This is safe because: (a) the signer always signs all four, and (b) `x-amz-date` is checked via clock-skew/scope independently of the signature, and `x-amz-content-sha256` is independently verified by `ResolvePayloadHash` (body re-hash + constant-time compare). An attacker stripping a signed header after signing cannot forge a valid signature because the signature covers the canonical request.

### 4. parseAuthHeader / ParseCredential — CORRECT

**`sigv4a.go:300-369`:** Scope must be exactly 4 parts ending in `aws4_request`; 5-part SigV4-style scopes rejected with `ErrMalformedAuth` (lines 336-339). Malformed → `ErrMalformedAuth`; missing/foreign prefix → `ErrMissingAuth` (line 301). Duplicate fields rejected (lines 314-326). `ParseCredential` wraps `parseAuthHeader`, propagating the same errors (lines 359-369).

### 5. regionSetMatches — CORRECT

**`regionset.go:10-30`:** `"*"` global wildcard, `"prefix*"` prefix wildcards, case-sensitive, empty region → `false` (line 11-13), empty entries skipped (lines 16-18), `*` anywhere but the end is not a wildcard (handled by `CutSuffix` only matching trailing `*`). Comprehensive test coverage in `regionset_test.go:5-31`.

### 6. iamsts caching — CORRECT

**`web/middleware/sigv4a/iamsts/iamsts.go`:**

- **singleflight + context.WithoutCancel** (lines 195-209): `entry()` collapses concurrent misses via `v.sf.Do`. The fetch runs on `context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)`. A disconnected caller cannot fail collapsed waiters. Tested by `TestVerifier_FetchSurvivesCallerCancel`.
- **Refusal caching** (lines 232-239): `NOT_FOUND`/`PERMISSION_DENIED` → cached `ErrUnknownKey` refusal with `negTTL` (default 30s). Unknown-vs-disabled distinction logged, not exposed. Tested by `TestVerifier_NegativeCache`.
- **Other errors uncached → 500** (lines 242-243): iamd down, transport fault, malformed key propagate as `Internal` (500), not denial. Tested by `TestVerifier_IAMOutageIs500`.
- **cache_until honored** (lines 258-263): absent `cache_until` → entry used once, never cached. Tested by `TestVerifier_TTLSweep`.
- **Store sweep** (lines 269-279): `store()` sweeps TTL-expired entries under `mu`. O(n) sweep is fine for small key cardinality. Tested by `TestVerifier_TTLSweep`.
- **Identity re-read** (Verify lines 153-156): re-calls `v.entry` after `inner.Verify` succeeds; refetches if expired between. Tested by `TestVerifier_IdentityThreaded` and `TestVerifier_RevocationAfterCacheUntil`.

### 7. Error mapping (TwirpError) — CORRECT

**`sigv4a.go:126-149`:**

| Error                                                                                                                           | Twirp code          | HTTP |
| ------------------------------------------------------------------------------------------------------------------------------- | ------------------- | ---- |
| `ErrMissingAuth`                                                                                                                | `Unauthenticated`   | 401  |
| `ErrMalformedAuth`                                                                                                              | `InvalidArgument`   | 400  |
| `ErrScopeMismatch`, `ErrClockSkew`, `ErrBodyTooLarge`, `ErrStreamingUnsupported`, `ErrMissingSignedHost`, `ErrMissingRegionSet` | `InvalidArgument`   | 400  |
| `ErrUnknownKey`, `ErrUnauthorized`, `ErrBodyHash`                                                                               | `PermissionDenied`  | 403  |
| Unexpected (key store down)                                                                                                     | `Internal` (logged) | 500  |

Key/signature failures collapse to one opaque message ("invalid authentication header") so a probe cannot distinguish unknown, disabled, and mis-signed credentials. Tested by `TestTwirpErrorMapping`.

### 8. Shared canonicalization (awssig) — CORRECT

**`web/middleware/internal/awssig/`** (`canonicalization.go`, `payload.go`, `awssig.go`):

- `BuildCanonicalRequest`, `CanonicalHeaderValue`, `collapseSpaces`, `CanonicalQuery`, `AWSURIEncode`, `CanonicalURI`, `wirePath`, `ResolvePayloadHash`.
- `collapseSpaces` folds whitespace runs unconditionally (matches AWS SDKs and aws-c-auth v4a vectors).
- `CanonicalQuery` sorts by name then value (not by joined string), avoiding prefix-name bugs.
- `ResolvePayloadHash`: STREAMING- rejected, body buffered+reset (capped at MaxBodySize), declared hash confirmed via `ConstantTimeCompare`, lowercase hex enforced, falls back to body hash when undeclared.

### 9. Replay doc — PRESENT AND ACCURATE

**`sigv4a.go:58-69`:** The `Verifier` doc comment clearly states SigV4A does NOT prevent replay, that a captured request verifies within `MaxClockSkew` (15 min default), and that callers requiring freshness must layer their own nonce/sequence/challenge. Accurate per the AWS SigV4A spec.

### 10. Tests — mostly complete, minor gaps

All test suites pass (`sigv4a`, `keyderivation`, `iamsts`, `sigv4aclient`, `awssig`) with `-race` and `go vet` clean.

**Minor test-coverage gaps (not defects):**

- `TestTwirpErrorMapping` does not exercise `ErrClockSkew`, `ErrBodyTooLarge`, `ErrStreamingUnsupported`, `ErrMissingSignedHost`, `ErrMissingRegionSet`, `ErrBodyHash` (all → InvalidArgument), or `ErrNotConfigured` (→ Internal). Straightforward switch branches; not a real gap.
- No test asserts replay protection (it is documentation, appropriately not tested).

## Security Summary

- **Replay:** Documented as not prevented; callers advised to layer nonce/sequence/challenge.
- **Probe indistinguishability:** Unknown key, disabled key, and signature mismatch all collapse to `PermissionDenied` (403) with opaque message.
- **Verification-only material:** `KeyLookup` path never sees the raw secret; public keys are verification-only.
- **Body size cap:** `MaxBodySize` (default 10 MiB) bounds memory an attacker can force the verifier to allocate.
- **Nil-key panic guard:** Prevents a lookuper bug from becoming a DoS via `ecdsa.VerifyASN1` nil dereference.
- **Singleflight resilience:** `context.WithoutCancel` ensures a disconnected caller cannot fail collapsed fetches.
- **Outage fail-closed:** IAM outages return 500, never 403.

## Recommendations

No code changes are required. The implementation is correct and spec-compliant. Optionally, expand `TestTwirpErrorMapping` to cover all InvalidArgument and Internal branches for completeness.
