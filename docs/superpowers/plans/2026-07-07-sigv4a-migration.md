# SigV4A Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add AWS Signature Version 4A (`AWS4-ECDSA-P256-SHA256`) authentication as a new `web/middleware/sigv4a` package tree and move `cmd/iamd` onto it, so downstream verifiers hold only public keys and can never forge signatures — while the existing `web/middleware/sigv4` package keeps working for stock-AWS-SDK interop.

**Architecture:** SigV4A derives a deterministic ECDSA P-256 keypair from the existing `(access key ID, secret access key)` pair — existing credentials keep working, nothing is reissued. The canonical-request machinery is identical to SigV4, so it is extracted into a shared `web/middleware/internal/awssig` package that both `sigv4` (non-breaking refactor) and the new `sigv4a` import; agreement between the two algorithms' signers and verifiers is then by construction. What differs in SigV4A: the credential scope drops region (`<date>/<service>/aws4_request`), region moves into a mandatory signed `X-Amz-Region-Set` header, and the signature is ECDSA over SHA-256 (hex-encoded ASN.1 DER) instead of HMAC. iamd's STS surface stops distributing forgery-capable derived signing keys and starts distributing PKIX-encoded public keys. Because ECDSA signatures are randomized, verification uses `ecdsa.VerifyASN1` (never recompute-and-compare) and tests anchor against AWS's official test-vector suite from `awslabs/aws-c-auth` instead of round-tripping through the AWS SDK signer (whose SigV4A implementation is `internal/` and unimportable).

**Tech Stack:** Go 1.25.7 stdlib only for the new crypto: `crypto/hmac`, `crypto/sha256`, `crypto/ecdsa`, `crypto/ecdh`, `crypto/elliptic`, `crypto/x509`, `math/big`. Protobuf via `buf` (`npm run generate`). `aws-sdk-go-v2` remains a dependency of `sigv4client` and of `sigv4`'s round-trip tests, both untouched.

## Global Constraints

- Go version: `go 1.25.7` (from `go.mod`) — `ecdsa.SignASN1/VerifyASN1`, `crypto/ecdh`, `strings.SplitSeq` are all available.
- Full test command: `npm test` (runs codegen then `go test ./...`). Every commit must leave the whole repo compiling and green.
- All commits: Conventional Commits, imperative lowercase description, and `--signoff`.
- Binaries must NOT call `flag.Parse()` (`internal.HandleStartup()` does it). This plan never touches flag parsing.
- Vendored/copied third-party material keeps its license attribution (aws-c-auth is Apache-2.0).
- `web/middleware/sigv4` keeps its full public API and behavior: it still verifies classic SigV4 from stock AWS SDKs. Only its internals move to `awssig`. `sigv4client` (signs SigV4 for real AWS services) is untouched.
- Algorithm constants (copy exactly): algorithm string `AWS4-ECDSA-P256-SHA256`; KDF secret prefix `AWS4A`; credential scope `<YYYYMMDD>/<service>/aws4_request` (NO region); signed-header requirement adds `x-amz-region-set`; signatures are hex-encoded ASN.1 DER.

## Key Design Decisions (locked in)

1. **Parallel package, shared internals.** `web/middleware/sigv4a` is a sibling of `sigv4`, not a rewrite of it. Shared spec-frozen code (canonicalization, payload-hash/body plumbing, timestamp/terminator constants) is extracted to `web/middleware/internal/awssig` — placed under `web/middleware/internal/` because Go's internal-visibility rule means a package under `web/middleware/sigv4/internal/` would be invisible to the sibling `sigv4a`.
2. **`sigv4` survives untouched in API and semantics.** Its refactor onto `awssig` is proven non-breaking by its existing AWS-SDK round-trip tests passing unchanged. `DeriveSigningKey`, `SigningKeyLookuper`, `sigv4keygen`, and the `sigv4/iamsts` subpackage all stay. The `GetSigningKey` RPC and its derived-key distribution chain are kept deliberately (user decision, 2026-07-07) as a working illustration of the classic SigV4 flow alongside the SigV4A one — nothing is deleted.
3. **`Lookuper` (access key ID → secret) is redefined identically in `sigv4a`** — iamd derives the keypair from the stored secret. The remote-verifier interface is `PublicKeyLookuper` (access key ID → `*ecdsa.PublicKey`): SigV4A keys are not scoped to date/region/service, so `sigv4`'s 4-tuple `SigningKeyLookuper` has no analogue.
4. **Public keys travel as PKIX/DER** (`x509.MarshalPKIXPublicKey` bytes) in the proto — self-describing, stdlib both ways.
5. **Trust model win (the point of all this):** `GetSigningKey` today hands verifiers symmetric keys they could forge with (recorded in `docs/plans/2026-06-29-sigv4-auth.md`). `GetPublicKey` hands out verification-only material. Revocation latency is still bounded by `cache_until`; the UTC-midnight `not_valid_after` bound disappears because SigV4A keys are not date-scoped. `GetSigningKey` remains served for illustration, so its recorded trust trade-off continues to apply to any deployment that still uses the classic chain.
6. **Proto change is purely additive:** Task 6 adds `GetPublicKey` beside `GetSigningKey` on `SigningKeyService`. Both RPCs remain served; no proto deletions or renames anywhere in this plan.
7. **`math/big` (non-constant-time) is fine in key derivation.** The comparison runs against our _own stored secret_ during derivation, not attacker-supplied input; there is no cross-trust-boundary timing oracle. Signature comparison itself is `ecdsa.VerifyASN1`, which is the correct primitive (ECDSA signatures are randomized, so constant-time byte comparison of signatures is meaningless).
8. **Client transport is `sigv4a/sigv4aclient`**, whose API (`Config` + `NewSigV4ARoundTripper`) deliberately mirrors `sigv4client.NewSigV4RoundTripper` so call-site migration in `iamclient` and tests is mechanical. The low-level `Signer` lives in the `sigv4a` package itself, next to the verifier and the unexported test hooks its vector tests need.

## File Structure

```
web/middleware/internal/awssig/       NEW  shared SigV4/SigV4A internals
  awssig.go                                consts, shared error sentinels, HMACSHA256
  canonicalization.go                      moved from sigv4 (identifiers exported)
  payload.go                               ResolvePayloadHash + body buffering
web/middleware/sigv4/                 MOD  non-breaking refactor onto awssig
  sigv4.go                                 delegate to awssig; API unchanged
  canonicalization.go                      DELETED (moved to awssig)
  iamsts/                                  kept (classic chain, illustration); test fake gains a stub in T6
  sigv4client/, sigv4keygen/, tests        untouched
web/middleware/sigv4a/                NEW  SigV4A implementation
  sigv4a.go                                Verifier, Middleware, TwirpError, parse
  keyderivation.go                         DeriveKeyPair
  regionset.go                             X-Amz-Region-Set matching
  sign.go                                  Signer (low-level)
  lookuper.go                              Lookuper, PublicKeyLookuper
  context.go                               KeyID / WithUser / User
  *_test.go                                vector-anchored tests
  testdata/v4a/                            curated aws-c-auth vectors + LICENSE
  sigv4aclient/sigv4aclient.go             Config + NewSigV4ARoundTripper
  iamsts/iamsts.go                         public-key caching verifier (+tests, integration.md)
pb/within/website/x/iam/sts/v1/sts.proto  MOD  +GetPublicKey (T6); GetSigningKey kept
cmd/iamd/
  services/iam/sts/publickey.go       NEW  GetPublicKey implementation (+test)
  services/iam/sts/signingkey.go      ---  untouched (GetSigningKey kept for illustration)
  auth.go, main.go                    MOD  verifier → sigv4a (T8)
  integration_test.go                 MOD  sign with sigv4aclient (T8)
  pub/iam/iamclient.go                MOD  sigv4client → sigv4aclient (T8)
docs/plans/2026-06-29-sigv4-auth.md   MOD  trust-model update (T9)
```

---

### Task 1: Extract shared internals into `web/middleware/internal/awssig`

Pure refactor. `sigv4`'s exported API, error values (via aliasing), and behavior are unchanged; its existing AWS-SDK round-trip tests are the proof.

**Files:**

- Create: `web/middleware/internal/awssig/awssig.go`
- Create: `web/middleware/internal/awssig/canonicalization.go`
- Create: `web/middleware/internal/awssig/payload.go`
- Modify: `web/middleware/sigv4/sigv4.go`
- Delete: `web/middleware/sigv4/canonicalization.go`

**Interfaces:**

- Consumes: current `sigv4` internals (`sigv4.go`, `canonicalization.go`).
- Produces (package `awssig`, used by `sigv4` now and `sigv4a` from Task 2 on): consts `AmzTimeFormat`, `ShortDateFormat`, `Terminator`, `UnsignedPayload`; error sentinels `ErrStreamingUnsupported`, `ErrBodyHash`, `ErrBodyTooLarge`; funcs `HMACSHA256(key, data []byte) []byte`, `CanonicalHeaderValue(r *http.Request, name string) string`, `CanonicalQuery(values url.Values, exclude string) string`, `AWSURIEncode(s string, encodeSlash bool) string`, `BuildCanonicalRequest(r *http.Request, sortedSignedHeaders []string, payloadHash string, disablePathEscaping bool) string`, `CanonicalURI(r *http.Request, disablePathEscaping bool) string`, `ResolvePayloadHash(r *http.Request, maxBodySize int64) (string, error)`.

- [ ] **Step 1: Create the shared package**

Create `web/middleware/internal/awssig/awssig.go`:

```go
// Package awssig holds the algorithm-independent internals shared by the
// SigV4 (web/middleware/sigv4) and SigV4A (web/middleware/sigv4a)
// middlewares: canonical-request construction, payload-hash resolution, and
// the constants both schemes define identically. Keeping one copy means the
// two packages' signers and verifiers agree by construction.
package awssig

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
)

const (
	// AmzTimeFormat is the X-Amz-Date timestamp layout.
	AmzTimeFormat = "20060102T150405Z"
	// ShortDateFormat is the credential-scope date layout.
	ShortDateFormat = "20060102"
	// Terminator ends every credential scope.
	Terminator = "aws4_request"
	// UnsignedPayload is the payload-hash sentinel for unsigned bodies. AWS
	// defines it case-sensitively.
	UnsignedPayload = "UNSIGNED-PAYLOAD"
)

// Error sentinels shared by both middlewares (each package re-exports them
// under its own name, so errors.Is works with either export).
var (
	ErrStreamingUnsupported = errors.New("sigv4: streaming payloads are not supported")
	ErrBodyHash             = errors.New("sigv4: body does not match x-amz-content-sha256")
	ErrBodyTooLarge         = errors.New("sigv4: request body exceeds limit")
)

// HMACSHA256 computes HMAC-SHA256(key, data).
func HMACSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}
```

Create `web/middleware/internal/awssig/canonicalization.go` by **moving** `web/middleware/sigv4/canonicalization.go` verbatim — change the package clause to `awssig` and export the four identifiers (`canonicalHeaderValue` → `CanonicalHeaderValue`, `collapseSpaces` → `collapseSpaces` (stays unexported, only used here), `canonicalQuery` → `CanonicalQuery`, `awsURIEncode` → `AWSURIEncode`). Function bodies and comments are unchanged. Then append the canonical-request builder (moved out of `sigv4.go:306-339`, generalized to free functions):

```go
// BuildCanonicalRequest renders the canonical request over exactly the given
// (already sorted, lowercase) signed-header names. Identical for SigV4 and
// SigV4A.
func BuildCanonicalRequest(r *http.Request, sortedSignedHeaders []string, payloadHash string, disablePathEscaping bool) string {
	var ch strings.Builder
	for _, h := range sortedSignedHeaders {
		ch.WriteString(h)
		ch.WriteByte(':')
		ch.WriteString(CanonicalHeaderValue(r, h))
		ch.WriteByte('\n')
	}

	return strings.Join([]string{
		r.Method,
		CanonicalURI(r, disablePathEscaping),
		CanonicalQuery(r.URL.Query(), "X-Amz-Signature"),
		ch.String(),
		strings.Join(sortedSignedHeaders, ";"),
		payloadHash,
	}, "\n")
}

// CanonicalURI renders the canonical URI. When disablePathEscaping is true
// (S3 style) the on-the-wire encoded path is used directly; otherwise the
// already-encoded path is encoded a second time, as AWS mandates for every
// non-S3 service.
func CanonicalURI(r *http.Request, disablePathEscaping bool) string {
	path := r.URL.EscapedPath()
	if path == "" {
		return "/"
	}
	if disablePathEscaping {
		return path
	}
	return AWSURIEncode(path, false)
}
```

Create `web/middleware/internal/awssig/payload.go` by moving `resolvePayloadHash` and `readAndLimitBody` from `sigv4.go:249-304`:

```go
package awssig

import (
	"bytes"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
)

// ResolvePayloadHash returns the hash string to place in the canonical
// request, buffering and resetting r.Body (capped at maxBodySize; zero means
// unlimited) so downstream handlers see a re-readable stream. When the client
// sent a concrete hash in x-amz-content-sha256 the buffered body is re-hashed
// and confirmed to match, otherwise the signed hash proves nothing about the
// bytes actually received. Any declared hash with a "STREAMING-" prefix is
// rejected — chunked-signing integrity lives in per-chunk signatures in the
// body framing, which neither middleware verifies, and rejecting the whole
// reserved sentinel family is the safe direction for both algorithms.
func ResolvePayloadHash(r *http.Request, maxBodySize int64) (string, error) {
	declared := r.Header.Get("X-Amz-Content-Sha256")

	if strings.HasPrefix(strings.ToUpper(declared), "STREAMING-") {
		return "", ErrStreamingUnsupported
	}

	body, err := readAndLimitBody(r, maxBodySize)
	if err != nil {
		return "", err
	}

	if declared == UnsignedPayload {
		return UnsignedPayload, nil
	}

	sum := sha256.Sum256(body)
	computed := hex.EncodeToString(sum[:])

	if declared == "" {
		// No content hash was signed; fall back to the body hash.
		return computed, nil
	}
	if subtle.ConstantTimeCompare([]byte(strings.ToLower(declared)), []byte(computed)) != 1 {
		return "", ErrBodyHash
	}
	// Use the verified lowercase hash in the canonical request; AWS requires
	// lowercase hex and some clients emit uppercase.
	return computed, nil
}

// readAndLimitBody reads the request body, enforcing maxBodySize when set,
// and resets r.Body to a reader over the bytes that were read on every return
// path so the body is always re-readable afterwards.
func readAndLimitBody(r *http.Request, maxBodySize int64) ([]byte, error) {
	var reader io.Reader = r.Body
	if maxBodySize > 0 {
		reader = io.LimitReader(r.Body, maxBodySize+1)
	}
	body, err := io.ReadAll(reader)
	// The original stream is consumed regardless; surface whatever was read.
	r.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if maxBodySize > 0 && int64(len(body)) > maxBodySize {
		return nil, ErrBodyTooLarge
	}
	return body, nil
}
```

(The one intentional behavior delta from `sigv4.go:255`: the streaming rejection broadens from case-insensitive-equality with one sentinel to any `STREAMING-` prefix. It only rejects more, matching the existing comment's rationale.)

- [ ] **Step 2: Refactor `sigv4` onto `awssig`**

In `web/middleware/sigv4/sigv4.go`:

1. Import `"within.website/x/web/middleware/internal/awssig"`; drop now-unused imports (`crypto/hmac`, `bytes`, `io` if nothing else uses them — the compiler will say).
2. Replace the const block (lines 31–36) and payload sentinels (43–46) with aliases so the exported API is untouched:

```go
const (
	algorithm       = "AWS4-HMAC-SHA256"
	terminator      = awssig.Terminator
	amzTimeFormat   = awssig.AmzTimeFormat
	shortDateFormat = awssig.ShortDateFormat
)

// Payload-hash sentinels defined by AWS SigV4, re-exported from the shared
// internals so both middlewares agree on one definition.
const (
	UnsignedPayload  = awssig.UnsignedPayload
	StreamingPayload = "STREAMING-AWS4-HMAC-SHA256-PAYLOAD"
)
```

3. In the errors block (50–61), alias the three shared sentinels so `errors.Is` keeps matching across packages:

```go
	ErrStreamingUnsupported = awssig.ErrStreamingUnsupported
	ErrBodyHash             = awssig.ErrBodyHash
	ErrBodyTooLarge         = awssig.ErrBodyTooLarge
```

4. Delete `resolvePayloadHash` and `readAndLimitBody` (lines 249–304); in `Verify` (line 154) call `awssig.ResolvePayloadHash(r, v.MaxBodySize)` instead.
5. Replace `canonicalRequest`/`canonicalURI` (lines 306–339) with a delegating method:

```go
func (v *Verifier) canonicalRequest(r *http.Request, sr *signedRequest, payloadHash string) string {
	headers := append([]string(nil), sr.signedHeaders...)
	sort.Strings(headers)
	return awssig.BuildCanonicalRequest(r, headers, payloadHash, v.DisablePathEscaping)
}
```

6. Delete the local `hmacSHA256` (lines 418–422); `verify` (line 237) and `DeriveSigningKey` (lines 429–434) call `awssig.HMACSHA256`.
7. Delete `web/middleware/sigv4/canonicalization.go`.

- [ ] **Step 3: Prove the refactor is invisible**

Run: `go build ./... && go test ./web/middleware/sigv4/... ./cmd/iamd/...`
Expected: PASS with zero test-file changes — the AWS-SDK round-trip tests exercising every moved code path are the non-breakage proof.

- [ ] **Step 4: Commit**

```bash
git add web/middleware/internal/awssig web/middleware/sigv4
git commit --signoff -m "refactor(sigv4): extract shared signing internals into awssig"
```

---

### Task 2: `sigv4a` key derivation + AWS test vectors

**Files:**

- Create: `web/middleware/sigv4a/testdata/v4a/` (vector directories + `LICENSE` + `README.md`)
- Create: `web/middleware/sigv4a/keyderivation.go`
- Create: `web/middleware/sigv4a/keyderivation_test.go`
- Create: `web/middleware/sigv4a/vectors_test.go`

**Interfaces:**

- Consumes: `awssig.HMACSHA256` (Task 1).
- Produces (package `sigv4a`): const `algorithm = "AWS4-ECDSA-P256-SHA256"`; `DeriveKeyPair(accessKeyID, secretAccessKey string) (*ecdsa.PrivateKey, error)`; test helpers `loadVectorContext`, `readVectorFile`, `vectorPublicKey`, `vectorDirs` (Tasks 4–5 reuse these).

- [ ] **Step 1: Import the AWS test vectors**

Clone the reference repo (shallow) and copy the curated vector set. These exercise query sorting, URI encoding, UTF-8 paths, and header canonicalization. Deliberately excluded: vectors signing `content-type` (the production signer signs a fixed header set), path-normalization vectors (`get-relative-*`, `get-slash*`, `get-space-*` — Go's URL parsing normalizes differently and the SigV4 verifier never normalized paths either), `get-header-value-multiline`, and `get-vanilla-with-session-token` (no session-token support).

```bash
cd /home/xe/Code/Xe/x
git clone --depth 1 https://github.com/awslabs/aws-c-auth /tmp/aws-c-auth-vectors
mkdir -p web/middleware/sigv4a/testdata/v4a
for v in get-vanilla get-vanilla-query get-vanilla-query-order-key \
         get-vanilla-query-unreserved get-vanilla-empty-query-key \
         get-unreserved get-utf8 get-header-value-trim \
         post-vanilla post-vanilla-query post-header-key-sort post-header-key-case; do
  cp -r /tmp/aws-c-auth-vectors/tests/aws-signing-test-suite/v4a/$v web/middleware/sigv4a/testdata/v4a/
done
cp /tmp/aws-c-auth-vectors/LICENSE web/middleware/sigv4a/testdata/v4a/LICENSE
```

Then create `web/middleware/sigv4a/testdata/v4a/README.md`:

```markdown
# SigV4A signing test vectors

Copied from https://github.com/awslabs/aws-c-auth
(`tests/aws-signing-test-suite/v4a/`), Apache License 2.0 — see LICENSE in
this directory.

Each directory is one scenario: `context.json` holds the credentials, region,
service, and timestamp; `header-canonical-request.txt`,
`header-string-to-sign.txt`, and `header-signed-request.txt` are the expected
signing artifacts; `public-key.json` is the ECDSA P-256 public key derived
from the credentials.

ECDSA signatures are randomized, so the `Signature=` values in these files
cannot be byte-compared against our output — tests must verify signatures
against `public-key.json` instead.

Vectors that sign `content-type`, use session tokens, or depend on AWS-style
path normalization are intentionally not copied; see the curation note in
docs/superpowers/plans/2026-07-07-sigv4a-migration.md.
```

- [ ] **Step 2: Write the vector-loading test helpers**

Create `web/middleware/sigv4a/vectors_test.go`:

```go
package sigv4a

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// vectorContext mirrors the context.json in each aws-c-auth v4a test vector.
type vectorContext struct {
	Credentials struct {
		AccessKeyID     string `json:"access_key_id"`
		SecretAccessKey string `json:"secret_access_key"`
	} `json:"credentials"`
	Region    string `json:"region"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}

func loadVectorContext(t *testing.T, dir string) vectorContext {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "v4a", dir, "context.json"))
	if err != nil {
		t.Fatalf("read context.json: %v", err)
	}
	var vc vectorContext
	if err := json.Unmarshal(b, &vc); err != nil {
		t.Fatalf("parse context.json: %v", err)
	}
	return vc
}

func (vc vectorContext) signingTime(t *testing.T) time.Time {
	t.Helper()
	ts, err := time.Parse(time.RFC3339, vc.Timestamp)
	if err != nil {
		t.Fatalf("parse timestamp: %v", err)
	}
	return ts
}

// readVectorFile returns the named file from a vector directory verbatim.
func readVectorFile(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "v4a", dir, name))
	if err != nil {
		t.Fatalf("read %s/%s: %v", dir, name, err)
	}
	return string(b)
}

// vectorPublicKey parses the vector's public-key.json (hex X/Y coordinates).
func vectorPublicKey(t *testing.T, dir string) *ecdsa.PublicKey {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "v4a", dir, "public-key.json"))
	if err != nil {
		t.Fatalf("read public-key.json: %v", err)
	}
	var pk struct{ X, Y string }
	if err := json.Unmarshal(b, &pk); err != nil {
		t.Fatalf("parse public-key.json: %v", err)
	}
	x, ok := new(big.Int).SetString(pk.X, 16)
	if !ok {
		t.Fatalf("bad X coordinate %q", pk.X)
	}
	y, ok := new(big.Int).SetString(pk.Y, 16)
	if !ok {
		t.Fatalf("bad Y coordinate %q", pk.Y)
	}
	return &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}
}

// vectorDirs lists every vector directory under testdata/v4a.
func vectorDirs(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join("testdata", "v4a"))
	if err != nil {
		t.Fatalf("read testdata/v4a: %v", err)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) == 0 {
		t.Fatal("no test vectors found")
	}
	return dirs
}
```

- [ ] **Step 3: Write the failing key-derivation tests**

Create `web/middleware/sigv4a/keyderivation_test.go`. Two independent known-answer anchors: aws-c-auth's fixed KDF vector (`tests/key_derivation_tests.c`, values AWS cross-validated against their own Go implementation), and each copied vector's public key (validated through aws-c-auth's full signing pipeline).

```go
package sigv4a

import (
	"encoding/hex"
	"testing"
)

// TestDeriveKeyPair_FixedVector pins the KDF against the known-answer vector
// in aws-c-auth tests/key_derivation_tests.c ("Values derived in
// synchronicity with Golang and IAM implementations").
func TestDeriveKeyPair_FixedVector(t *testing.T) {
	priv, err := DeriveKeyPair("AKISORANDOMAASORANDOM", "q+jcrXGc+0zWN6uzclKVhvMmUsIfRPa4rlRandom")
	if err != nil {
		t.Fatalf("DeriveKeyPair: %v", err)
	}
	if got, want := hex.EncodeToString(priv.D.FillBytes(make([]byte, 32))),
		"7fd3bd010c0d9c292141c2b77bfbde1042c92e6836fff749d1269ec890fca1bd"; got != want {
		t.Errorf("private key = %s, want %s", got, want)
	}
	if got, want := hex.EncodeToString(priv.PublicKey.X.FillBytes(make([]byte, 32))),
		"15d242ceebf8d8169fd6a8b5a746c41140414c3b07579038da06af89190fffcb"; got != want {
		t.Errorf("public X = %s, want %s", got, want)
	}
	if got, want := hex.EncodeToString(priv.PublicKey.Y.FillBytes(make([]byte, 32))),
		"0515242cedd82e94799482e4c0514b505afccf2c0c98d6a553bf539f424c5ec0"; got != want {
		t.Errorf("public Y = %s, want %s", got, want)
	}
}

// TestDeriveKeyPair_Vectors derives a keypair from each test vector's
// credentials and checks it matches the vector's published public key.
func TestDeriveKeyPair_Vectors(t *testing.T) {
	for _, dir := range vectorDirs(t) {
		t.Run(dir, func(t *testing.T) {
			vc := loadVectorContext(t, dir)
			priv, err := DeriveKeyPair(vc.Credentials.AccessKeyID, vc.Credentials.SecretAccessKey)
			if err != nil {
				t.Fatalf("DeriveKeyPair: %v", err)
			}
			want := vectorPublicKey(t, dir)
			if priv.PublicKey.X.Cmp(want.X) != 0 || priv.PublicKey.Y.Cmp(want.Y) != 0 {
				t.Error("derived public key does not match vector public-key.json")
			}
		})
	}
}
```

- [ ] **Step 4: Run tests to verify they fail**

Run: `go test ./web/middleware/sigv4a/ -run TestDeriveKeyPair -v`
Expected: FAIL — `undefined: DeriveKeyPair` (compile error).

- [ ] **Step 5: Implement DeriveKeyPair**

Create `web/middleware/sigv4a/keyderivation.go`:

```go
// Package sigv4a signs and verifies HTTP requests with AWS Signature Version
// 4A (the AWS4-ECDSA-P256-SHA256 scheme): the same canonical-request
// construction as classic SigV4 (shared via web/middleware/internal/awssig),
// but signed with an ECDSA P-256 key derived deterministically from the
// credential instead of an HMAC key. Verifiers can therefore hold only the
// public key — material that verifies signatures but can never mint them.
package sigv4a

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"math/big"

	"within.website/x/web/middleware/internal/awssig"
)

// algorithm is the SigV4A signing algorithm name. It is also the label in
// the key-derivation input.
const algorithm = "AWS4-ECDSA-P256-SHA256"

// nMinusTwo is the P-256 curve order minus two, the rejection-sampling bound
// from the SigV4A key-derivation spec (aws-c-auth key_derivation.c).
var nMinusTwo = new(big.Int).Sub(elliptic.P256().Params().N, big.NewInt(2))

// DeriveKeyPair deterministically derives the SigV4A ECDSA P-256 keypair for
// a credential. It implements AWS's counter-mode KDF (NIST SP 800-108 style,
// PRF = HMAC-SHA256, key = "AWS4A"+secret) with rejection sampling: a
// candidate above N-2 retries with the next counter byte (1..254), an
// accepted candidate is incremented by one to land in [1, N-1]. The
// per-attempt rejection probability is ~2^-32, so the loop virtually always
// succeeds on the first pass.
//
// The keypair is a pure function of (accessKeyID, secretAccessKey): unlike
// the SigV4 HMAC ladder there is no date/region/service scoping, so the same
// key signs and verifies for the credential's whole lifetime.
//
// The big.Int comparison is not constant-time; that is acceptable because
// derivation only ever runs over our own stored secret, never comparing
// against attacker-controlled input.
func DeriveKeyPair(accessKeyID, secretAccessKey string) (*ecdsa.PrivateKey, error) {
	inputKey := []byte("AWS4A" + secretAccessKey)
	for counter := 1; counter <= 254; counter++ {
		// fixedInput layout (aws-c-auth key_derivation.c):
		//   BE32(1) || label || 0x00 || accessKeyID || counterByte || BE32(256)
		fixedInput := make([]byte, 0, 32+len(accessKeyID))
		fixedInput = append(fixedInput, 0x00, 0x00, 0x00, 0x01)
		fixedInput = append(fixedInput, algorithm...)
		fixedInput = append(fixedInput, 0x00)
		fixedInput = append(fixedInput, accessKeyID...)
		fixedInput = append(fixedInput, byte(counter))
		fixedInput = append(fixedInput, 0x00, 0x00, 0x01, 0x00)

		candidate := new(big.Int).SetBytes(awssig.HMACSHA256(inputKey, fixedInput))
		if candidate.Cmp(nMinusTwo) > 0 {
			continue
		}
		candidate.Add(candidate, big.NewInt(1))

		// Round-trip the scalar through crypto/ecdh: NewPrivateKey validates
		// it is in [1, N-1] and computes the public point without deprecated
		// elliptic API calls.
		ek, err := ecdh.P256().NewPrivateKey(candidate.FillBytes(make([]byte, 32)))
		if err != nil {
			return nil, err
		}
		pub := ek.PublicKey().Bytes() // uncompressed SEC1: 0x04 || X || Y
		return &ecdsa.PrivateKey{
			PublicKey: ecdsa.PublicKey{
				Curve: elliptic.P256(),
				X:     new(big.Int).SetBytes(pub[1:33]),
				Y:     new(big.Int).SetBytes(pub[33:65]),
			},
			D: candidate,
		}, nil
	}
	return nil, errors.New("sigv4a: key derivation exhausted its counter space")
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./web/middleware/sigv4a/ -run TestDeriveKeyPair -v`
Expected: PASS (both tests, all vector subtests).

- [ ] **Step 7: Commit**

```bash
git add web/middleware/sigv4a
git commit --signoff -m "feat(sigv4a): add ecdsa p-256 key derivation with aws test vectors"
```

---

### Task 3: X-Amz-Region-Set matching

**Files:**

- Create: `web/middleware/sigv4a/regionset.go`
- Create: `web/middleware/sigv4a/regionset_test.go`

**Interfaces:**

- Produces: `regionSetMatches(regionSet, region string) bool` (unexported; the Task 4 verifier calls it).

- [ ] **Step 1: Write the failing table-driven test**

Create `web/middleware/sigv4a/regionset_test.go` (follow the repo's table-driven test conventions):

```go
package sigv4a

import "testing"

func TestRegionSetMatches(t *testing.T) {
	tests := []struct {
		name      string
		regionSet string
		region    string
		want      bool
	}{
		{name: "exact match", regionSet: "us-east-1", region: "us-east-1", want: true},
		{name: "exact mismatch", regionSet: "us-east-1", region: "eu-west-1", want: false},
		{name: "list match", regionSet: "us-east-1,eu-west-1", region: "eu-west-1", want: true},
		{name: "list with spaces", regionSet: "us-east-1, eu-west-1", region: "eu-west-1", want: true},
		{name: "global wildcard", regionSet: "*", region: "anything", want: true},
		{name: "prefix wildcard match", regionSet: "us-west-*", region: "us-west-2", want: true},
		{name: "prefix wildcard mismatch", regionSet: "us-west-*", region: "us-east-1", want: false},
		{name: "empty set", regionSet: "", region: "us-east-1", want: false},
		{name: "empty region", regionSet: "*", region: "", want: false},
		{name: "empty entries skipped", regionSet: ",,us-east-1", region: "us-east-1", want: true},
		{name: "wildcard is prefix only", regionSet: "us-*-1", region: "us-east-1", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := regionSetMatches(tt.regionSet, tt.region); got != tt.want {
				t.Errorf("regionSetMatches(%q, %q) = %v, want %v", tt.regionSet, tt.region, got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./web/middleware/sigv4a/ -run TestRegionSetMatches -v`
Expected: FAIL — `undefined: regionSetMatches`.

- [ ] **Step 3: Implement regionSetMatches**

Create `web/middleware/sigv4a/regionset.go`:

```go
package sigv4a

import "strings"

// regionSetMatches reports whether region is covered by a SigV4A
// X-Amz-Region-Set value: a comma-separated list of region names where an
// entry may be "*" or carry a trailing "*" for prefix matching ("us-west-*").
// A "*" anywhere but the end of an entry is not a wildcard; entries match
// case-sensitively, matching how the verifier pins Region elsewhere.
func regionSetMatches(regionSet, region string) bool {
	if region == "" {
		return false
	}
	for entry := range strings.SplitSeq(regionSet, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if prefix, ok := strings.CutSuffix(entry, "*"); ok {
			if strings.HasPrefix(region, prefix) {
				return true
			}
			continue
		}
		if entry == region {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./web/middleware/sigv4a/ -run TestRegionSetMatches -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/middleware/sigv4a/regionset.go web/middleware/sigv4a/regionset_test.go
git commit --signoff -m "feat(sigv4a): add x-amz-region-set matching"
```

---

### Task 4: `sigv4a` verifier

The verifier is a fork of `sigv4.Verifier` with the algorithm-specific parts swapped. Tests are fully vector-driven: the AWS suite's pre-signed requests are verified as-is, and perturbed for the rejection paths — no signer needed yet.

**Files:**

- Create: `web/middleware/sigv4a/lookuper.go`
- Create: `web/middleware/sigv4a/context.go`
- Create: `web/middleware/sigv4a/sigv4a.go`
- Create: `web/middleware/sigv4a/sigv4a_test.go`

**Interfaces:**

- Consumes: `DeriveKeyPair`, `algorithm` (Task 2); `regionSetMatches` (Task 3); `awssig.*` (Task 1); vector helpers (Task 2).
- Produces (package `sigv4a`): `Verifier{Region, Service string; Lookup Lookuper; KeyLookup PublicKeyLookuper; MaxClockSkew time.Duration; DisablePathEscaping bool; MaxBodySize int64; Now func() time.Time}` with `Middleware(next http.Handler) http.Handler` and `Verify(r *http.Request) (string, error)`; `TwirpError(ctx, err) error`; error sentinels `ErrMissingAuth`, `ErrMissingSignedHost`, `ErrMissingRegionSet`, `ErrUnknownKey`, `ErrClockSkew`, `ErrScopeMismatch`, `ErrStreamingUnsupported`, `ErrBodyHash`, `ErrUnauthorized`, `ErrBodyTooLarge`, `ErrNotConfigured`; `Lookuper`/`LookuperFunc`; `PublicKeyLookuper`/`PublicKeyLookuperFunc`; `Credential{AccessKeyID, Date, Service string}` + `ParseCredential`; `KeyID(ctx)`, `WithUser(ctx, u)`, `User(ctx)`; test helper `parseVectorRequest`. Tasks 5–8 rely on these names exactly.

- [ ] **Step 1: Create the lookup interfaces**

Create `web/middleware/sigv4a/lookuper.go`:

```go
package sigv4a

import (
	"context"
	"crypto/ecdsa"
)

// Lookuper encapsulates the secret access key lookup so that the underlying
// logic can have arbitrary implementations.
type Lookuper interface {
	// Lookup resolves an access key id to its secret access key. Return
	// ErrUnknownKey for unknown keys. This is the one piece you must supply.
	Lookup(accessKeyID string) (secretAccessKey string, err error)
}

// LookuperFunc adapts an ordinary function to the Lookuper interface, the
// same way http.HandlerFunc adapts a function to http.Handler.
type LookuperFunc func(accessKeyID string) (secretAccessKey string, err error)

// Lookup calls f(accessKeyID).
func (f LookuperFunc) Lookup(accessKeyID string) (secretAccessKey string, err error) {
	return f(accessKeyID)
}

// PublicKeyLookuper resolves an access key id to the credential's SigV4A
// ECDSA P-256 public key. Return ErrUnknownKey when the key does not exist
// or may not sign (disabled key or user); any other error is treated as a
// server fault. Public keys are verification-only material: an
// implementation can hold and cache them without ever being able to mint a
// signature.
type PublicKeyLookuper interface {
	LookupPublicKey(ctx context.Context, accessKeyID string) (*ecdsa.PublicKey, error)
}

// PublicKeyLookuperFunc adapts an ordinary function to PublicKeyLookuper.
type PublicKeyLookuperFunc func(ctx context.Context, accessKeyID string) (*ecdsa.PublicKey, error)

// LookupPublicKey calls f.
func (f PublicKeyLookuperFunc) LookupPublicKey(ctx context.Context, accessKeyID string) (*ecdsa.PublicKey, error) {
	return f(ctx, accessKeyID)
}
```

- [ ] **Step 2: Create the context plumbing**

Copy `web/middleware/sigv4/context.go` to `web/middleware/sigv4a/context.go` changing only the package clause to `sigv4a` (it carries `KeyID`/`withKeyID`, `WithUser`/`User`; same iam proto import).

- [ ] **Step 3: Create the verifier**

Copy `web/middleware/sigv4/sigv4.go` to `web/middleware/sigv4a/sigv4a.go`, then apply these transformations (each shown in full):

1. Package clause: the file starts with only `package sigv4a` — the package doc already lives on `keyderivation.go`. Delete the copied package comment.
2. Imports: add `"crypto/ecdsa"` and `"within.website/x/web/middleware/internal/awssig"`; remove `"crypto/hmac"` if present after the Task 1 refactor and any import the compiler flags as unused.
3. Constants:

```go
const (
	terminator      = awssig.Terminator
	amzTimeFormat   = awssig.AmzTimeFormat
	shortDateFormat = awssig.ShortDateFormat
)

// Payload-hash sentinels. UnsignedPayload matches exactly (AWS defines it
// case-sensitively); any STREAMING-* sentinel is rejected inside
// awssig.ResolvePayloadHash.
const (
	UnsignedPayload  = awssig.UnsignedPayload
	StreamingPayload = "STREAMING-AWS4-ECDSA-P256-SHA256-PAYLOAD"
)
```

(`algorithm` is already declared in `keyderivation.go` — delete the copied declaration.)

4. Errors block — replace wholesale:

```go
// Errors returned by Verify. Callers typically map ErrUnauthorized and
// ErrUnknownKey to 403 and the rest to 400.
var (
	ErrMissingAuth          = errors.New("sigv4a: missing or malformed Authorization")
	ErrMissingSignedHost    = errors.New("sigv4a: host must appear in SignedHeaders")
	ErrMissingRegionSet     = errors.New("sigv4a: x-amz-region-set must appear in SignedHeaders")
	ErrUnknownKey           = errors.New("sigv4a: unknown access key id")
	ErrClockSkew            = errors.New("sigv4a: request time outside allowed skew")
	ErrScopeMismatch        = errors.New("sigv4a: credential scope does not match")
	ErrStreamingUnsupported = awssig.ErrStreamingUnsupported
	ErrBodyHash             = awssig.ErrBodyHash
	ErrUnauthorized         = errors.New("sigv4a: signature mismatch")
	ErrBodyTooLarge         = awssig.ErrBodyTooLarge
	ErrNotConfigured        = errors.New("sigv4a: neither Verifier.Lookup nor Verifier.KeyLookup is set")
)
```

5. `Verifier` struct: the `Region` doc becomes `// Region must be covered by the request's signed X-Amz-Region-Set; Service must match the credential scope.`; the key-lookup field becomes:

```go
	// KeyLookup resolves an access key id to its ECDSA public key. When set
	// it takes precedence over Lookup, and the verifier never sees the raw
	// secret — this is how services that must not hold secrets verify
	// locally (see web/middleware/sigv4a/iamsts). Exactly one of Lookup or
	// KeyLookup must be set.
	KeyLookup PublicKeyLookuper
```

6. In `TwirpError`, add `errors.Is(err, ErrMissingRegionSet)` to the `InvalidArgument` case group beside `ErrMissingSignedHost`.
7. In `Verify`, the payload-hash call is `awssig.ResolvePayloadHash(r, v.MaxBodySize)` (the copied `resolvePayloadHash`/`readAndLimitBody` methods were already removed from sigv4.go in Task 1, so they are not in the copy).
8. In `verify`: replace the scope pin with

```go
	// Pin the scope. Without this, a signature valid for some other service
	// would be accepted here. Region is not part of the SigV4A credential
	// scope; it is checked below through the signed X-Amz-Region-Set.
	if sr.scope.service != v.Service {
		return "", ErrScopeMismatch
	}
```

after the signed-host check add

```go
	// The region set feeds the scope decision, so it must be signed —
	// otherwise a relay could rewrite the audience of a captured signature.
	if !slices.Contains(sr.signedHeaders, "x-amz-region-set") {
		return "", ErrMissingRegionSet
	}
	if !regionSetMatches(r.Header.Get("X-Amz-Region-Set"), v.Region) {
		return "", ErrScopeMismatch
	}
```

replace the key-resolution block with

```go
	var pub *ecdsa.PublicKey
	if v.KeyLookup != nil {
		pub, err = v.KeyLookup.LookupPublicKey(r.Context(), sr.accessKeyID)
	} else {
		var secret string
		secret, err = v.Lookup.Lookup(sr.accessKeyID)
		if err == nil {
			var priv *ecdsa.PrivateKey
			priv, err = DeriveKeyPair(sr.accessKeyID, secret)
			if err == nil {
				pub = &priv.PublicKey
			}
		}
	}
	if err != nil {
		return "", err
	}
```

replace the scope string with

```go
	scopeStr := strings.Join([]string{sr.scope.date, sr.scope.service, terminator}, "/")
```

and replace the HMAC recompute-and-compare with

```go
	sig, err := hex.DecodeString(sr.signature)
	if err != nil {
		return "", ErrUnauthorized
	}
	digest := sha256.Sum256([]byte(stringToSign))
	if !ecdsa.VerifyASN1(pub, digest[:], sig) {
		return "", ErrUnauthorized
	}
	return sr.accessKeyID, nil
```

(`crypto/subtle` drops out of the imports.) 9. `canonicalRequest` delegates to `awssig.BuildCanonicalRequest` exactly as sigv4's does after Task 1. 10. `credentialScope` loses `region`; `parseAuthHeader` requires 4 credential parts:

```go
	cp := strings.Split(cred, "/")
	if len(cp) != 4 || cp[3] != terminator {
		return nil, fmt.Errorf("%w: bad credential scope", ErrMissingAuth)
	}
	return &signedRequest{
		accessKeyID:   cp[0],
		scope:         credentialScope{date: cp[1], service: cp[2]},
		signedHeaders: strings.Split(signed, ";"),
		signature:     sig,
	}, nil
```

11. `Credential` loses its `Region` field; `ParseCredential`'s doc says `AWS4-ECDSA-P256-SHA256`.
12. Delete the copied `DeriveSigningKey` and any `hmacSHA256` remnant — SigV4A has no HMAC signing path.

- [ ] **Step 4: Write the vector-driven verifier tests**

Create `web/middleware/sigv4a/sigv4a_test.go`:

```go
package sigv4a

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// parseVectorRequest builds an *http.Request from a vector's request.txt or
// header-signed-request.txt.
func parseVectorRequest(t *testing.T, raw string) (*http.Request, []byte) {
	t.Helper()
	r, err := http.ReadRequest(bufio.NewReader(strings.NewReader(raw)))
	if err != nil {
		t.Fatalf("ReadRequest: %v", err)
	}
	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return r, body
}

// vectorVerifier returns a Verifier configured for a vector's scope and
// clock, resolving only that vector's credential.
func vectorVerifier(t *testing.T, vc vectorContext) *Verifier {
	t.Helper()
	return &Verifier{
		Region:  vc.Region,
		Service: vc.Service,
		Lookup: LookuperFunc(func(id string) (string, error) {
			if id == vc.Credentials.AccessKeyID {
				return vc.Credentials.SecretAccessKey, nil
			}
			return "", ErrUnknownKey
		}),
		Now: func() time.Time { return vc.signingTime(t).Add(5 * time.Second) },
	}
}

// TestVectors_Verify feeds each pre-signed request from the AWS test suite
// through the Verifier: an independent implementation's signature must
// verify against the key derived from the same credentials.
func TestVectors_Verify(t *testing.T) {
	for _, dir := range vectorDirs(t) {
		t.Run(dir, func(t *testing.T) {
			vc := loadVectorContext(t, dir)
			req, _ := parseVectorRequest(t, readVectorFile(t, dir, "header-signed-request.txt"))
			got, err := vectorVerifier(t, vc).Verify(req)
			if err != nil {
				t.Fatalf("verify: %v", err)
			}
			if got != vc.Credentials.AccessKeyID {
				t.Fatalf("key = %q, want %q", got, vc.Credentials.AccessKeyID)
			}
		})
	}
}

// TestVerifyRejections perturbs the get-vanilla vector to exercise each
// rejection path.
func TestVerifyRejections(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(v *Verifier)
		tamper  func(req *http.Request)
		wantErr error
	}{
		{
			name: "tampered signature",
			tamper: func(req *http.Request) {
				req.Header.Set("Authorization", flipLastDigit(req.Header.Get("Authorization")))
			},
			wantErr: ErrUnauthorized,
		},
		{
			name: "clock skew",
			mutate: func(v *Verifier) {
				v.Now = func() time.Time { return time.Date(2015, 8, 30, 14, 0, 0, 0, time.UTC) } // +~1.5h
			},
			wantErr: ErrClockSkew,
		},
		{
			name: "unknown key",
			mutate: func(v *Verifier) {
				v.Lookup = LookuperFunc(func(string) (string, error) { return "", ErrUnknownKey })
			},
			wantErr: ErrUnknownKey,
		},
		{
			// A signature minted for one service must not verify against
			// another, even though the math would otherwise check out.
			name:    "wrong service scope",
			mutate:  func(v *Verifier) { v.Service = "other-svc" },
			wantErr: ErrScopeMismatch,
		},
		{
			// The verifier's region must be covered by the signed
			// X-Amz-Region-Set.
			name:    "region not in region set",
			mutate:  func(v *Verifier) { v.Region = "eu-west-1" },
			wantErr: ErrScopeMismatch,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vc := loadVectorContext(t, "get-vanilla")
			req, _ := parseVectorRequest(t, readVectorFile(t, "get-vanilla", "header-signed-request.txt"))
			if tc.tamper != nil {
				tc.tamper(req)
			}
			v := vectorVerifier(t, vc)
			if tc.mutate != nil {
				tc.mutate(v)
			}
			if _, err := v.Verify(req); !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func flipLastDigit(auth string) string {
	b := []byte(auth)
	last := len(b) - 1
	if b[last] == '0' {
		b[last] = '1'
	} else {
		b[last] = '0'
	}
	return string(b)
}

// host must be a signed header; otherwise a captured request can be replayed
// against any other host sharing the same region/service verifier.
func TestUnsignedHostRejected(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)
	req.Header.Set("Authorization", "AWS4-ECDSA-P256-SHA256 "+
		"Credential=AKIDEXAMPLE/20150830/service/aws4_request, "+
		"SignedHeaders=x-amz-date, Signature=0000")
	vc := loadVectorContext(t, "get-vanilla")
	if _, err := vectorVerifier(t, vc).Verify(req); err != ErrMissingSignedHost {
		t.Fatalf("err = %v, want ErrMissingSignedHost", err)
	}
}

// x-amz-region-set must be a signed header: it feeds the scope decision, so
// an unsigned value would let a relay rewrite the audience of a signature.
func TestUnsignedRegionSetRejected(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)
	req.Header.Set("Authorization", "AWS4-ECDSA-P256-SHA256 "+
		"Credential=AKIDEXAMPLE/20150830/service/aws4_request, "+
		"SignedHeaders=host;x-amz-date, Signature=0000")
	vc := loadVectorContext(t, "get-vanilla")
	if _, err := vectorVerifier(t, vc).Verify(req); err != ErrMissingRegionSet {
		t.Fatalf("err = %v, want ErrMissingRegionSet", err)
	}
}

// A Verifier without a Lookup must return an error, not panic.
func TestNilLookuper(t *testing.T) {
	v := &Verifier{Region: "us-east-1", Service: "service"}
	req := httptest.NewRequest(http.MethodGet, "https://api.example.com/", nil)
	if _, err := v.Verify(req); err != ErrNotConfigured {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}

// TestParseCredential extracts the scope tuple from an Authorization header.
func TestParseCredential(t *testing.T) {
	const h = "AWS4-ECDSA-P256-SHA256 Credential=AKIDEXAMPLE/20150830/iam/aws4_request, SignedHeaders=host;x-amz-date;x-amz-region-set, Signature=3045022100aaaa022100bbbb"
	c, err := ParseCredential(h)
	if err != nil {
		t.Fatalf("ParseCredential: %v", err)
	}
	want := Credential{AccessKeyID: "AKIDEXAMPLE", Date: "20150830", Service: "iam"}
	if *c != want {
		t.Errorf("credential = %+v, want %+v", *c, want)
	}
	if _, err := ParseCredential("Bearer nope"); !errors.Is(err, ErrMissingAuth) {
		t.Errorf("bad header err = %v, want ErrMissingAuth", err)
	}
	if _, err := ParseCredential("AWS4-ECDSA-P256-SHA256 Credential=AKID/20150830/us-east-1/iam/aws4_request, SignedHeaders=host, Signature=00"); !errors.Is(err, ErrMissingAuth) {
		t.Errorf("5-part (SigV4-style) scope err = %v, want ErrMissingAuth", err)
	}
}
```

- [ ] **Step 5: Run, fix, pass**

Run: `go test ./web/middleware/sigv4a/ -v`
Expected: initially FAIL while Step 3's transformations are incomplete, then PASS. If a specific vector fails only inside `http.ReadRequest`/URL parsing (not in canonicalization), delete that vector directory from testdata and record it in `testdata/v4a/README.md` — do not add skip logic.

- [ ] **Step 6: Commit**

```bash
go build ./... && go test ./web/middleware/sigv4a/...
git add web/middleware/sigv4a
git commit --signoff -m "feat(sigv4a): add request verifier middleware"
```

---

### Task 5: `sigv4a` signer + `sigv4aclient` transport

**Files:**

- Create: `web/middleware/sigv4a/sign.go`
- Create: `web/middleware/sigv4a/sign_test.go`
- Create: `web/middleware/sigv4a/sigv4aclient/sigv4aclient.go`
- Create: `web/middleware/sigv4a/sigv4aclient/sigv4aclient_test.go`

**Interfaces:**

- Consumes: `DeriveKeyPair`, `algorithm`, verifier, `awssig.BuildCanonicalRequest`, vector helpers, `parseVectorRequest` (Tasks 1–4).
- Produces: in `sigv4a`: `NewSigner(accessKeyID, secretAccessKey, region, service string) (*Signer, error)`; `(*Signer).Sign(r *http.Request, body []byte) error`; exported field `Signer.Now`; unexported `(*Signer).sign(r, signedHeaders, payloadHash)`. In `sigv4aclient`: `Config{Region, AccessKey, SecretKey, ServiceName string}` with `(*Config).Valid() error`; `NewSigV4ARoundTripper(cfg *Config, next http.RoundTripper) (http.RoundTripper, error)` — mirrors `sigv4client.NewSigV4RoundTripper`. Tasks 7–9 rely on these names exactly.

- [ ] **Step 1: Write the failing signer tests**

Create `web/middleware/sigv4a/sign_test.go`. The vector test reproduces each vector's exact canonical request, then proves our signature verifies against a digest of the _vector's_ string-to-sign file — so any single-byte disagreement in canonicalization or string-to-sign construction fails the ECDSA check. A round-trip test then closes the loop against our own verifier.

```go
package sigv4a

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"within.website/x/web/middleware/internal/awssig"
)

// authField extracts one comma-separated field ("Credential",
// "SignedHeaders", "Signature") from an Authorization header value.
func authField(t *testing.T, auth, field string) string {
	t.Helper()
	for _, part := range strings.Split(auth, " ") {
		part = strings.TrimSuffix(strings.TrimSpace(part), ",")
		if v, ok := strings.CutPrefix(part, field+"="); ok {
			return v
		}
	}
	t.Fatalf("no %s= in %q", field, auth)
	return ""
}

// TestSigner_Vectors signs each AWS test-suite request and checks every
// signing artifact against the vector: the canonical request and the
// Credential/SignedHeaders fields byte-for-byte, and the signature by ECDSA
// verification against the vector's public key over the vector's own
// string-to-sign. Signature bytes themselves are never compared: ECDSA is
// randomized (the suite itself ships two different valid signatures per
// vector).
func TestSigner_Vectors(t *testing.T) {
	for _, dir := range vectorDirs(t) {
		t.Run(dir, func(t *testing.T) {
			vc := loadVectorContext(t, dir)
			r, body := parseVectorRequest(t, readVectorFile(t, dir, "request.txt"))
			sum := sha256.Sum256(body)
			payloadHash := hex.EncodeToString(sum[:])

			s, err := NewSigner(vc.Credentials.AccessKeyID, vc.Credentials.SecretAccessKey, vc.Region, vc.Service)
			if err != nil {
				t.Fatalf("NewSigner: %v", err)
			}
			s.Now = func() time.Time { return vc.signingTime(t) }

			wantSigned := authField(t, readVectorFile(t, dir, "header-signed-request.txt"), "SignedHeaders")
			signedHeaders := strings.Split(wantSigned, ";")
			if err := s.sign(r, signedHeaders, payloadHash); err != nil {
				t.Fatalf("sign: %v", err)
			}

			sorted := append([]string(nil), signedHeaders...)
			sort.Strings(sorted)
			if got, want := awssig.BuildCanonicalRequest(r, sorted, payloadHash, false),
				readVectorFile(t, dir, "header-canonical-request.txt"); got != want {
				t.Errorf("canonical request mismatch:\ngot:\n%s\nwant:\n%s", got, want)
			}

			auth := r.Header.Get("Authorization")
			wantCred := authField(t, readVectorFile(t, dir, "header-signed-request.txt"), "Credential")
			if got := authField(t, auth, "Credential"); got != wantCred {
				t.Errorf("Credential = %q, want %q", got, wantCred)
			}
			if got := authField(t, auth, "SignedHeaders"); got != wantSigned {
				t.Errorf("SignedHeaders = %q, want %q", got, wantSigned)
			}

			sig, err := hex.DecodeString(authField(t, auth, "Signature"))
			if err != nil {
				t.Fatalf("signature is not hex: %v", err)
			}
			digest := sha256.Sum256([]byte(readVectorFile(t, dir, "header-string-to-sign.txt")))
			if !ecdsa.VerifyASN1(vectorPublicKey(t, dir), digest[:], sig) {
				t.Error("signature does not verify against the vector string-to-sign and public key")
			}
		})
	}
}

func testSigner(t *testing.T) *Signer {
	t.Helper()
	s, err := NewSigner("AKIDEXAMPLE", "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY", "us-east-1", "execute-api")
	if err != nil {
		t.Fatalf("NewSigner: %v", err)
	}
	s.Now = func() time.Time { return time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC) }
	return s
}

func testVerifier() *Verifier {
	return &Verifier{
		Region:  "us-east-1",
		Service: "execute-api",
		Lookup: LookuperFunc(func(id string) (string, error) {
			if id == "AKIDEXAMPLE" {
				return "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY", nil
			}
			return "", ErrUnknownKey
		}),
		Now: func() time.Time { return time.Date(2026, 6, 29, 12, 0, 5, 0, time.UTC) },
	}
}

// TestRoundTrip_SignVerify closes the loop: our signer's output verifies
// with our verifier, bodies survive, and tampering is caught.
func TestRoundTrip_SignVerify(t *testing.T) {
	t.Run("GET with query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://api.example.com/v1/things?b=2&a=1&list=1&list-type=2", nil)
		if err := testSigner(t).Sign(req, nil); err != nil {
			t.Fatalf("Sign: %v", err)
		}
		got, err := testVerifier().Verify(req)
		if err != nil {
			t.Fatalf("verify: %v", err)
		}
		if got != "AKIDEXAMPLE" {
			t.Fatalf("key = %q", got)
		}
	})

	t.Run("POST body verifies and survives", func(t *testing.T) {
		body := []byte(`{"hello":"world"}`)
		req := httptest.NewRequest(http.MethodPost, "https://api.example.com/v1/submit", strings.NewReader(string(body)))
		if err := testSigner(t).Sign(req, body); err != nil {
			t.Fatalf("Sign: %v", err)
		}
		if _, err := testVerifier().Verify(req); err != nil {
			t.Fatalf("verify: %v", err)
		}
		rest, _ := io.ReadAll(req.Body)
		if string(rest) != string(body) {
			t.Fatalf("body not reset: got %q", rest)
		}
	})

	t.Run("tampered body rejected", func(t *testing.T) {
		body := []byte(`{"amount":1}`)
		req := httptest.NewRequest(http.MethodPost, "https://api.example.com/pay", strings.NewReader(string(body)))
		if err := testSigner(t).Sign(req, body); err != nil {
			t.Fatalf("Sign: %v", err)
		}
		req.Body = io.NopCloser(strings.NewReader(`{"amount":1000000}`))
		if _, err := testVerifier().Verify(req); err == nil {
			t.Fatal("expected rejection of tampered body")
		}
	})

	t.Run("path with space double-encodes", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://api.example.com/a%20b/c", nil)
		if err := testSigner(t).Sign(req, nil); err != nil {
			t.Fatalf("Sign: %v", err)
		}
		if _, err := testVerifier().Verify(req); err != nil {
			t.Fatalf("verify: %v", err)
		}
	})
}

// TestMiddleware checks the full http.Handler path, including status mapping.
func TestMiddleware(t *testing.T) {
	var gotKey string
	h := testVerifier().Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey, _ = KeyID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))

	t.Run("valid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://api.example.com/ok", nil)
		if err := testSigner(t).Sign(req, nil); err != nil {
			t.Fatalf("Sign: %v", err)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", rec.Code)
		}
		if gotKey != "AKIDEXAMPLE" {
			t.Fatalf("context key = %q", gotKey)
		}
	})

	t.Run("unsigned", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "https://api.example.com/ok", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./web/middleware/sigv4a/ -run 'TestSigner|TestRoundTrip_SignVerify|TestMiddleware' -v`
Expected: FAIL — `undefined: NewSigner`.

- [ ] **Step 3: Implement the signer**

Create `web/middleware/sigv4a/sign.go`:

```go
package sigv4a

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"within.website/x/web/middleware/internal/awssig"
)

// Signer signs outgoing HTTP requests with AWS Signature Version 4A. It is
// the client-side counterpart to Verifier and shares its canonicalization
// through awssig, so the two agree by construction.
//
// The ECDSA keypair is derived once at construction; the secret access key
// is not retained. Application code usually wants the sigv4aclient
// subpackage, which wraps a Signer in an http.RoundTripper.
type Signer struct {
	accessKeyID string
	priv        *ecdsa.PrivateKey
	region      string
	service     string

	// Now is overridable for tests. Defaults to time.Now.
	Now func() time.Time
}

// NewSigner derives the SigV4A keypair for the credential and returns a
// Signer that signs for the given region (the X-Amz-Region-Set value; a
// single region name for within.website services) and service.
func NewSigner(accessKeyID, secretAccessKey, region, service string) (*Signer, error) {
	priv, err := DeriveKeyPair(accessKeyID, secretAccessKey)
	if err != nil {
		return nil, err
	}
	return &Signer{accessKeyID: accessKeyID, priv: priv, region: region, service: service}, nil
}

// Sign signs r in place: it hashes body (which must be the full request
// payload; nil means empty), declares it in X-Amz-Content-Sha256, and signs
// the host, x-amz-content-sha256, x-amz-date, and x-amz-region-set headers.
// It does not touch r.Body.
func (s *Signer) Sign(r *http.Request, body []byte) error {
	sum := sha256.Sum256(body)
	payloadHash := hex.EncodeToString(sum[:])
	r.Header.Set("X-Amz-Content-Sha256", payloadHash)
	return s.sign(r, []string{"host", "x-amz-content-sha256", "x-amz-date", "x-amz-region-set"}, payloadHash)
}

// sign stamps X-Amz-Date and X-Amz-Region-Set, builds the canonical request
// over exactly signedHeaders, and writes the Authorization header. It is
// split from Sign so tests can reproduce the AWS test-suite vectors, which
// sign without x-amz-content-sha256.
func (s *Signer) sign(r *http.Request, signedHeaders []string, payloadHash string) error {
	if r.Host == "" {
		r.Host = r.URL.Host
	}
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	amzDate := now().UTC().Format(amzTimeFormat)
	r.Header.Set("X-Amz-Date", amzDate)
	r.Header.Set("X-Amz-Region-Set", s.region)

	headers := append([]string(nil), signedHeaders...)
	sort.Strings(headers)

	canonReq := awssig.BuildCanonicalRequest(r, headers, payloadHash, false)
	// SigV4A credential scope: date/service/aws4_request — no region. The
	// region set is bound to the signature as a signed header instead.
	scope := strings.Join([]string{amzDate[:len(shortDateFormat)], s.service, terminator}, "/")
	hashed := sha256.Sum256([]byte(canonReq))
	stringToSign := strings.Join([]string{
		algorithm,
		amzDate,
		scope,
		hex.EncodeToString(hashed[:]),
	}, "\n")

	digest := sha256.Sum256([]byte(stringToSign))
	sig, err := ecdsa.SignASN1(rand.Reader, s.priv, digest[:])
	if err != nil {
		return fmt.Errorf("sigv4a: signing: %w", err)
	}

	r.Header.Set("Authorization", fmt.Sprintf("%s Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		algorithm, s.accessKeyID, scope, strings.Join(headers, ";"), hex.EncodeToString(sig)))
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./web/middleware/sigv4a/ -v`
Expected: PASS, every vector subtest included.

- [ ] **Step 5: Write the failing sigv4aclient tests**

Create `web/middleware/sigv4a/sigv4aclient/sigv4aclient_test.go`:

```go
package sigv4aclient

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func validConfig() *Config {
	return &Config{
		Region:      "us-east-1",
		AccessKey:   "AKIDEXAMPLE",
		SecretKey:   "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		ServiceName: "iam",
	}
}

func TestConfigValid(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{name: "complete", mutate: func(*Config) {}, wantErr: false},
		{name: "missing region", mutate: func(c *Config) { c.Region = "" }, wantErr: true},
		{name: "missing access key", mutate: func(c *Config) { c.AccessKey = "" }, wantErr: true},
		{name: "missing secret key", mutate: func(c *Config) { c.SecretKey = "" }, wantErr: true},
		{name: "missing service", mutate: func(c *Config) { c.ServiceName = "" }, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(cfg)
			if err := cfg.Valid(); (err != nil) != tt.wantErr {
				t.Errorf("Valid() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// authField extracts one comma-separated field from an Authorization header.
func authField(t *testing.T, auth, field string) string {
	t.Helper()
	for _, part := range strings.Split(auth, " ") {
		part = strings.TrimSuffix(strings.TrimSpace(part), ",")
		if v, ok := strings.CutPrefix(part, field+"="); ok {
			return v
		}
	}
	t.Fatalf("no %s= in %q", field, auth)
	return ""
}

// TestRoundTrip checks the deployment shape: body buffered and preserved,
// payload hash declared, all four standard headers signed, and the caller's
// request left unmutated.
func TestRoundTrip(t *testing.T) {
	var got *http.Request
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(r.Context())
		gotBody, _ = io.ReadAll(r.Body)
	}))
	defer srv.Close()

	rt, err := NewSigV4ARoundTripper(validConfig(), nil)
	if err != nil {
		t.Fatalf("NewSigV4ARoundTripper: %v", err)
	}
	client := &http.Client{Transport: rt}

	const body = `{"x":1}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/things", strings.NewReader(body))
	if _, err := client.Do(req); err != nil {
		t.Fatalf("Do: %v", err)
	}

	if string(gotBody) != body {
		t.Errorf("body = %q, want %q", gotBody, body)
	}
	auth := got.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "AWS4-ECDSA-P256-SHA256 ") {
		t.Errorf("Authorization = %q, want AWS4-ECDSA-P256-SHA256 prefix", auth)
	}
	if want := "host;x-amz-content-sha256;x-amz-date;x-amz-region-set"; authField(t, auth, "SignedHeaders") != want {
		t.Errorf("SignedHeaders = %q, want %q", authField(t, auth, "SignedHeaders"), want)
	}
	if got.Header.Get("X-Amz-Region-Set") != "us-east-1" {
		t.Error("X-Amz-Region-Set not set")
	}
	sum := sha256.Sum256([]byte(body))
	if got.Header.Get("X-Amz-Content-Sha256") != hex.EncodeToString(sum[:]) {
		t.Error("X-Amz-Content-Sha256 does not match body")
	}
	if req.Header.Get("Authorization") != "" {
		t.Error("RoundTrip mutated the caller's request")
	}
}

func TestInvalidConfigRejected(t *testing.T) {
	cfg := validConfig()
	cfg.SecretKey = ""
	if _, err := NewSigV4ARoundTripper(cfg, nil); err == nil {
		t.Fatal("expected error for invalid config")
	}
}
```

- [ ] **Step 6: Run to verify failure, then implement**

Run: `go test ./web/middleware/sigv4a/sigv4aclient/ -v` — expected FAIL (`undefined: Config`).

Create `web/middleware/sigv4a/sigv4aclient/sigv4aclient.go`:

```go
// Package sigv4aclient provides an http.RoundTripper that signs outgoing
// requests with AWS Signature Version 4A. It is the SigV4A counterpart to
// web/middleware/sigv4/sigv4client (which signs classic SigV4 for real AWS
// services): use this package for within.website services verified by
// web/middleware/sigv4a. The API deliberately mirrors sigv4client so
// migrating a call site is a rename.
package sigv4aclient

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"within.website/x/web/middleware/sigv4a"
)

// Config holds the static credential and scope a round tripper signs with.
// All fields are required: SigV4A for within.website services has no
// AWS-style credential chain or default region to fall back on.
type Config struct {
	// Region becomes the X-Amz-Region-Set header value.
	Region string
	// AccessKey and SecretKey are the IAM credential to sign with.
	AccessKey string
	SecretKey string
	// ServiceName is the credential-scope service, e.g. "iam".
	ServiceName string
}

// Valid reports whether every required field is set.
func (c *Config) Valid() error {
	switch {
	case c == nil:
		return fmt.Errorf("sigv4aclient: nil config")
	case c.Region == "":
		return fmt.Errorf("sigv4aclient: Region is required")
	case c.AccessKey == "":
		return fmt.Errorf("sigv4aclient: AccessKey is required")
	case c.SecretKey == "":
		return fmt.Errorf("sigv4aclient: SecretKey is required")
	case c.ServiceName == "":
		return fmt.Errorf("sigv4aclient: ServiceName is required")
	}
	return nil
}

// NewSigV4ARoundTripper returns an http.RoundTripper that signs every
// request per cfg and forwards it to next (http.DefaultTransport when nil).
// The caller's request is cloned and its body buffered for hashing; the
// original is never mutated.
func NewSigV4ARoundTripper(cfg *Config, next http.RoundTripper) (http.RoundTripper, error) {
	if err := cfg.Valid(); err != nil {
		return nil, err
	}
	signer, err := sigv4a.NewSigner(cfg.AccessKey, cfg.SecretKey, cfg.Region, cfg.ServiceName)
	if err != nil {
		return nil, err
	}
	if next == nil {
		next = http.DefaultTransport
	}
	return &roundTripper{signer: signer, next: next}, nil
}

type roundTripper struct {
	signer *sigv4a.Signer
	next   http.RoundTripper
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	var body []byte
	if req.Body != nil {
		var err error
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		req.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(body))
		r.ContentLength = int64(len(body))
	}
	if err := rt.signer.Sign(r, body); err != nil {
		return nil, err
	}
	return rt.next.RoundTrip(r)
}
```

- [ ] **Step 7: Run tests to verify they pass, commit**

```bash
go test ./web/middleware/sigv4a/... && go build ./...
git add web/middleware/sigv4a
git commit --signoff -m "feat(sigv4a): add request signer and sigv4aclient transport"
```

---

### Task 6: `GetPublicKey` RPC — proto + iamd implementation (additive)

**Files:**

- Modify: `pb/within/website/x/iam/sts/v1/sts.proto`
- Create: `cmd/iamd/services/iam/sts/publickey.go`
- Create: `cmd/iamd/services/iam/sts/publickey_test.go`
- Modify: `web/middleware/sigv4/iamsts/iamsts_test.go` and/or `web/middleware/sigv4/iamsts/e2e_test.go` — one-line stub (see Step 3)

**Interfaces:**

- Consumes: `sigv4a.DeriveKeyPair` (Task 2); existing `models.DAO.GetKeyWithUser` (`cmd/iamd/models/keys.go:128`); existing `SigningKeys` struct fields `dao`, `cacheTTL`, `now()` (`signingkey.go:32-49,120-125`).
- Produces: proto `GetPublicKey(GetPublicKeyRequest) returns (GetPublicKeyResponse)` on `SigningKeyService`; message `GetPublicKeyResponse{public_key bytes, identity TokenIdentity, cache_until Timestamp}`; Go method `(*SigningKeys).GetPublicKey`. Task 7's iamsts fetch path calls this RPC.

- [ ] **Step 1: Extend the proto**

In `pb/within/website/x/iam/sts/v1/sts.proto`, add inside `service SigningKeyService` (after the `GetSigningKey` rpc at line 29):

```proto
  // GetPublicKey returns the SigV4A (ECDSA P-256) public verification key
  // for an access key id, plus the identity it authenticates. Public keys
  // are not secret: holding one lets a service verify signatures but never
  // mint them, unlike the symmetric derived keys from GetSigningKey.
  //
  // Errors:
  //   NOT_FOUND         - no such access key id
  //   PERMISSION_DENIED - key or owning user is disabled
  //   INVALID_ARGUMENT  - missing access_key_id
  rpc GetPublicKey(GetPublicKeyRequest) returns (GetPublicKeyResponse);
```

And add after the `GetSigningKeyResponse` message (line 70):

```proto
message GetPublicKeyRequest {
  // The access key id parsed from the request's Credential= component.
  string access_key_id = 1 [(buf.validate.field).required = true];
}

message GetPublicKeyResponse {
  // PKIX, ASN.1 DER-encoded ECDSA P-256 public key, as produced by Go's
  // x509.MarshalPKIXPublicKey and parsed by x509.ParsePKIXPublicKey.
  bytes public_key = 1;

  // Who this key authenticates as. Downstream services use this for
  // authorization and audit attribution after the signature checks out.
  TokenIdentity identity = 2;

  // How long the caller may cache this response before asking again. Bounds
  // revocation latency: a disabled key stops verifying within one cache TTL.
  // A response without cache_until may be used once but never cached.
  google.protobuf.Timestamp cache_until = 3;
}
```

- [ ] **Step 2: Regenerate**

Run: `npm run generate`
Expected: exits 0; `gen/within/website/x/iam/sts/v1/` regenerates with the new RPC.

- [ ] **Step 3: Fix the now-broken test stub in the old iamsts**

`go build ./... && go vet ./web/middleware/sigv4/iamsts/` will show the `fakeKeys` test double (in `web/middleware/sigv4/iamsts`'s test files) no longer satisfies the widened `stsv1.SigningKeyService` interface. Add a stub method next to its `GetSigningKey`:

```go
func (f *fakeKeys) GetPublicKey(ctx context.Context, req *stsv1.GetPublicKeyRequest) (*stsv1.GetPublicKeyResponse, error) {
	return nil, twirp.NewError(twirp.Unimplemented, "not implemented in this fake")
}
```

(Locate it with `grep -rn "fakeKeys" web/middleware/sigv4/iamsts/`. If more than one fake implements the interface, stub each. The package itself stays — it remains the classic chain's working illustration.)

Run: `go test ./... 2>&1 | tail -5` — everything must compile and pass again before continuing.

- [ ] **Step 4: Write the failing service test**

Create `cmd/iamd/services/iam/sts/publickey_test.go`. It reuses `newSigningKeysTest` and `wantTwirpCode` from `signingkey_test.go:37,62`:

```go
package sts

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"testing"
	"time"

	"github.com/twitchtv/twirp"

	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	"within.website/x/web/middleware/sigv4a"
)

func TestGetPublicKey(t *testing.T) {
	ctx := context.Background()

	t.Run("success returns the derived public key", func(t *testing.T) {
		s, _, akid, secret := newSigningKeysTest(t)
		resp, err := s.GetPublicKey(ctx, &stsv1.GetPublicKeyRequest{AccessKeyId: akid})
		if err != nil {
			t.Fatalf("GetPublicKey: %v", err)
		}

		parsed, err := x509.ParsePKIXPublicKey(resp.GetPublicKey())
		if err != nil {
			t.Fatalf("response public key does not parse: %v", err)
		}
		pub, ok := parsed.(*ecdsa.PublicKey)
		if !ok {
			t.Fatalf("public key is %T, want *ecdsa.PublicKey", parsed)
		}
		priv, err := sigv4a.DeriveKeyPair(akid, secret)
		if err != nil {
			t.Fatalf("DeriveKeyPair: %v", err)
		}
		if !pub.Equal(&priv.PublicKey) {
			t.Error("response public key does not match the key derived from the stored secret")
		}

		if got := resp.GetIdentity().GetDisplayName(); got != "tester" {
			t.Errorf("display_name = %q, want tester", got)
		}
		if resp.GetIdentity().GetAccessKeyId() != akid {
			t.Errorf("identity access_key_id = %q, want %q", resp.GetIdentity().GetAccessKeyId(), akid)
		}
		if got := resp.GetCacheUntil().AsTime(); !got.Equal(fixedNow.Add(5 * time.Minute)) {
			t.Errorf("cache_until = %v, want now+5m", got)
		}
	})

	t.Run("unknown key is NOT_FOUND", func(t *testing.T) {
		s, _, _, _ := newSigningKeysTest(t)
		_, err := s.GetPublicKey(ctx, &stsv1.GetPublicKeyRequest{AccessKeyId: "AKIDNOPE"})
		wantTwirpCode(t, err, twirp.NotFound)
	})

	t.Run("disabled key is PERMISSION_DENIED", func(t *testing.T) {
		s, dao, akid, _ := newSigningKeysTest(t)
		if err := dao.DisableKey(ctx, akid, "test", ""); err != nil {
			t.Fatalf("DisableKey: %v", err)
		}
		_, err := s.GetPublicKey(ctx, &stsv1.GetPublicKeyRequest{AccessKeyId: akid})
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
		_, err = s.GetPublicKey(ctx, &stsv1.GetPublicKeyRequest{AccessKeyId: akid})
		wantTwirpCode(t, err, twirp.PermissionDenied)
	})

	t.Run("missing access_key_id is INVALID_ARGUMENT", func(t *testing.T) {
		s, _, _, _ := newSigningKeysTest(t)
		_, err := s.GetPublicKey(ctx, &stsv1.GetPublicKeyRequest{})
		wantTwirpCode(t, err, twirp.InvalidArgument)
	})
}
```

- [ ] **Step 5: Run test to verify it fails**

Run: `go test ./cmd/iamd/services/iam/sts/ -run TestGetPublicKey -v`
Expected: FAIL — `s.GetPublicKey undefined` (or Unimplemented errors if an embed satisfies the compiler; either way the subtests fail).

- [ ] **Step 6: Implement GetPublicKey**

Create `cmd/iamd/services/iam/sts/publickey.go`:

```go
package sts

import (
	"context"
	"crypto/x509"
	"errors"

	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	stsv1 "within.website/x/gen/within/website/x/iam/sts/v1"
	"within.website/x/web/middleware/sigv4a"
)

// GetPublicKey resolves the access key and its owning user, derives the
// SigV4A keypair from the stored secret, and returns the public half in PKIX
// DER form. Unlike GetSigningKey there is no scope validation: the keypair
// is a pure function of the credential, not of a (date, region, service)
// tuple, and the public key is not sensitive material.
func (s *SigningKeys) GetPublicKey(ctx context.Context, req *stsv1.GetPublicKeyRequest) (*stsv1.GetPublicKeyResponse, error) {
	if req.GetAccessKeyId() == "" {
		return nil, twirp.RequiredArgumentError("access_key_id")
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

	priv, err := sigv4a.DeriveKeyPair(k.AccessKeyID, k.SecretAccessKey)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}

	return &stsv1.GetPublicKeyResponse{
		PublicKey: der,
		Identity: &stsv1.TokenIdentity{
			AccessKeyId: k.AccessKeyID,
			PrincipalId: k.User.UUID,
			DisplayName: k.User.Name,
		},
		CacheUntil: timestamppb.New(s.now().Add(s.cacheTTL)),
	}, nil
}
```

- [ ] **Step 7: Run tests to verify they pass, commit**

```bash
go test ./cmd/iamd/... ./web/middleware/...
git add pb/within/website/x/iam/sts/v1/sts.proto gen cmd/iamd/services/iam/sts web/middleware/sigv4/iamsts
git commit --signoff -m "feat(iamd): serve sigv4a public verification keys via GetPublicKey"
```

---

### Task 7: `sigv4a/iamsts` — public-key caching verifier

A new package modeled on `web/middleware/sigv4/iamsts` (which stays untouched until Task 9). Same operational shape — negative cache, singleflight, `cache_until`, sweep-on-insert — but the cache is keyed by access key id alone and holds public keys.

**Files:**

- Create: `web/middleware/sigv4a/iamsts/iamsts.go`
- Create: `web/middleware/sigv4a/iamsts/iamsts_test.go`
- Create: `web/middleware/sigv4a/iamsts/e2e_test.go`

**Interfaces:**

- Consumes: `sigv4a.Verifier`, `sigv4a.PublicKeyLookuper`, `sigv4a.ErrUnknownKey`, `sigv4a.TwirpError`; `stsv1.SigningKeyService.GetPublicKey` (Task 6); `sigv4a.NewSigner` (tests).
- Produces: `Config{BaseURL string; HTTPClient *http.Client; Region, Service string; MaxBodySize int64; NegativeTTL time.Duration; Now func() time.Time}`; `New(cfg Config) *Verifier`; `(*Verifier).Middleware`, `(*Verifier).Verify(r) (*Identity, error)`, `(*Verifier).LookupPublicKey`; `Identity{AccessKeyID, OrganizationID, PrincipalID, DisplayName string; SignedAt time.Time}`; `Caller(ctx) (*Identity, bool)`. Task 8's downstream story and Task 9's docs rely on these.

- [ ] **Step 1: Write the package**

Create `web/middleware/sigv4a/iamsts/iamsts.go`. This is a port of `web/middleware/sigv4/iamsts/iamsts.go` — copy it, then apply the following (every changed region shown):

1. Package doc:

```go
// Package iamsts authenticates HTTP requests signed with AWS Signature
// Version 4A by verifying them locally against a cached ECDSA public key
// fetched from IAM's key service.
//
// The verifying service holds verification-only material: even a full
// compromise of its cache cannot mint a signature, unlike the classic SigV4
// derived-key scheme this replaces. It fetches the PKIX-encoded public key
// for an access key id once, caches it for the server-advised TTL, and
// verifies signatures itself. When the cache is warm, verification is a pure
// function of the request bytes and the cached key — no IAM RPC on the hot
// path.
//
// Caching rules: entries are keyed by access key id (SigV4A keys have no
// date/region/service scoping), honor the response's cache_until, collapse
// concurrent misses into a single RPC, and remember refusals (unknown or
// disabled keys) briefly so a flood of bad credentials cannot hammer IAM.
//
// Authenticate the client the same way as any other iamd caller: give
// Config.HTTPClient a sigv4aclient transport signing with the verifier's own
// IAM credential.
package iamsts
```

2. Imports: `sigv4` → `sigv4a` (`within.website/x/web/middleware/sigv4a`), plus `"crypto/ecdsa"`, `"crypto/elliptic"`, `"crypto/x509"`, `"fmt"`.
3. Delete the `scopeKey` type; `cache` becomes `map[string]*entry` keyed by access key id, and `entry`:

```go
// entry is one cache slot: either a public key with its identity, or a
// remembered refusal (err set). expiresAt of zero means "use once, do not
// serve from cache".
type entry struct {
	pub       *ecdsa.PublicKey
	identity  *stsv1.TokenIdentity
	err       error
	expiresAt time.Time
}
```

4. In `New`, the inner verifier is `*sigv4a.Verifier` (same fields), and `KeyLookup: v` now satisfies `sigv4a.PublicKeyLookuper`.
5. `Verify` simplifies — the verified key id is the cache key, no credential re-parse:

```go
func (v *Verifier) Verify(r *http.Request) (*Identity, error) {
	keyID, err := v.inner.Verify(r)
	if err != nil {
		return nil, err
	}

	// The signature checked out, so this entry was just used; re-read it for
	// the identity. In the unlikely case it expired in between, this
	// refetches — still off the hot path.
	e, err := v.entry(r.Context(), keyID)
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
```

6. `LookupSigningKey` becomes:

```go
// LookupPublicKey implements sigv4a.PublicKeyLookuper through the cache. The
// inner verifier only calls it after the clock-skew and scope checks pass,
// so unverifiable garbage never triggers an RPC.
func (v *Verifier) LookupPublicKey(ctx context.Context, accessKeyID string) (*ecdsa.PublicKey, error) {
	e, err := v.entry(ctx, accessKeyID)
	if err != nil {
		return nil, err
	}
	return e.pub, nil
}
```

7. `entry` and `store` take `accessKeyID string`; the singleflight key is `accessKeyID` directly. The `store` sweep comment now reads: evicts TTL-expired credentials so stale entries do not accumulate.
8. `fetch` becomes:

```go
// fetch performs the GetPublicKey RPC and stores the result. NOT_FOUND and
// PERMISSION_DENIED become cached refusals surfacing as ErrUnknownKey, so a
// probe cannot distinguish unknown from disabled credentials; the
// distinction is logged here. Any other failure (iamd down, transport fault,
// malformed key) is returned uncached and surfaces as an internal error — an
// IAM outage must read as an outage, not as a denial.
func (v *Verifier) fetch(ctx context.Context, accessKeyID string) (*entry, error) {
	resp, err := v.client.GetPublicKey(ctx, &stsv1.GetPublicKeyRequest{AccessKeyId: accessKeyID})
	if err != nil {
		var te twirp.Error
		if errors.As(err, &te) {
			switch te.Code() {
			case twirp.NotFound, twirp.PermissionDenied:
				slog.InfoContext(ctx, "iamsts: public key refused",
					"code", string(te.Code()),
					"access_key_id", accessKeyID,
				)
				e := &entry{err: sigv4a.ErrUnknownKey, expiresAt: v.now().Add(v.negTTL)}
				v.store(accessKeyID, e)
				return e, nil
			}
		}
		return nil, err
	}

	parsed, err := x509.ParsePKIXPublicKey(resp.GetPublicKey())
	if err != nil {
		return nil, fmt.Errorf("iamsts: response public key does not parse: %w", err)
	}
	pub, ok := parsed.(*ecdsa.PublicKey)
	if !ok || pub.Curve != elliptic.P256() {
		return nil, fmt.Errorf("iamsts: response public key is %T, want ECDSA P-256", parsed)
	}

	e := &entry{pub: pub, identity: resp.GetIdentity()}
	// Honor the server's caching bound exactly: a response without
	// cache_until is used for this request but never cached — the server
	// declined to grant a TTL and we do not invent one.
	if cu := resp.GetCacheUntil(); cu != nil {
		e.expiresAt = cu.AsTime()
	}
	if e.expiresAt.After(v.now()) {
		v.store(accessKeyID, e)
	}
	return e, nil
}
```

9. `Middleware` uses `sigv4a.TwirpError`. `Identity`, `Caller`, `withCaller`, `cacheLen` are unchanged. `Config`'s `HTTPClient` doc recommends a `sigv4aclient` transport.

- [ ] **Step 2: Write the tests**

Create `iamsts_test.go` and `e2e_test.go` ported from `web/middleware/sigv4/iamsts/`'s test files, with these systematic substitutions — keep every test's name and assertion intent:

- The fake key service implements `GetPublicKey` with real key material; give it an Unimplemented `GetSigningKey` stub to satisfy the widened interface (this fake only serves the public-key flow):

```go
func (f *fakeKeys) GetPublicKey(ctx context.Context, req *stsv1.GetPublicKeyRequest) (*stsv1.GetPublicKeyResponse, error) {
	f.mu.Lock()
	f.calls++
	f.mu.Unlock()
	if f.refuse != nil {
		return nil, f.refuse
	}
	if req.GetAccessKeyId() != testKey {
		return nil, twirp.NotFoundError("unknown access key id")
	}
	priv, err := sigv4a.DeriveKeyPair(testKey, testSecret)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return nil, twirp.InternalErrorWith(err)
	}
	return &stsv1.GetPublicKeyResponse{
		PublicKey:  der,
		Identity:   &stsv1.TokenIdentity{AccessKeyId: testKey, PrincipalId: "user-uuid", DisplayName: "tester"},
		CacheUntil: timestamppb.New(f.now().Add(f.cacheTTL)),
	}, nil
}
```

(adapt field names to whatever the existing fake uses — read it first; the shape above is the contract.)

- The `signedGET` helper signs with `sigv4a.NewSigner(testKey, testSecret, testRegion, testSvc)` and a pinned `Now` instead of the AWS SDK signer.
- The midnight-rollover test is replaced by a TTL-expiry sweep test: fetch keys for two different access key ids (extend the fake to serve a second credential), advance `now` past the first entry's `cache_until`, trigger an insert, assert `cacheLen()` shows the expired slot evicted.
- All other tests port mechanically: cache hit counts RPCs, negative caching, singleflight collapse, revocation after cache_until, skew-rejected-without-RPC, IAM-outage-is-500, fetch-survives-caller-cancel, identity threading.
- `e2e_test.go` drives the full deployment shape with a `sigv4aclient.NewSigV4ARoundTripper` transport against `stsv1.NewSigningKeyServiceServer(fake)`.

- [ ] **Step 3: Run, fix, pass, commit**

```bash
go test ./web/middleware/sigv4a/... && go build ./...
git add web/middleware/sigv4a/iamsts
git commit --signoff -m "feat(sigv4a): add iamsts public-key caching verifier"
```

---

### Task 8: Move iamd onto sigv4a

After this task iamd authenticates callers with SigV4A only; the CLI client and tests switch in the same commit. The old `sigv4` package remains fully functional for other consumers.

**Files:**

- Modify: `cmd/iamd/auth.go`
- Modify: `cmd/iamd/integration_test.go`
- Modify: `cmd/iamd/pub/iam/iamclient.go`
- Modify: `cmd/iamd/main.go` (doc comments only)

**Interfaces:**

- Consumes: `sigv4a.Verifier`, `sigv4a.LookuperFunc`, `sigv4a.KeyID`, `sigv4a.WithUser`, `sigv4a.ErrUnknownKey` (Task 4); `sigv4aclient` (Task 5).
- Produces: iamd verifying SigV4A end to end.

- [ ] **Step 1: Switch the middleware (`cmd/iamd/auth.go`)**

Change the import `"within.website/x/web/middleware/sigv4"` → `"within.website/x/web/middleware/sigv4a"` and every `sigv4.` reference to `sigv4a.` — the API names were kept identical on purpose (`Verifier`, `LookuperFunc`, `ErrUnknownKey`, `KeyID`, `WithUser`). The resulting `newVerifier` body:

```go
func newVerifier(dao *models.DAO, region, service string, maxBodySize int64) *sigv4a.Verifier {
	return &sigv4a.Verifier{
		Region:      region,
		Service:     service,
		MaxBodySize: maxBodySize,
		Lookup: sigv4a.LookuperFunc(func(accessKeyID string) (string, error) {
			secret, err := dao.SecretFor(context.Background(), accessKeyID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return "", sigv4a.ErrUnknownKey
				}
				return "", err
			}
			return secret, nil
		}),
	}
}
```

`UserMiddleware` likewise swaps `sigv4.KeyID`/`sigv4.WithUser` → `sigv4a.KeyID`/`sigv4a.WithUser`. In `main.go`, the `verifier *sigv4.Verifier` parameter type in `newMux` (line 59) becomes `*sigv4a.Verifier` and its import updates; the `-region` flag help text (line 31) becomes "SigV4A X-Amz-Region-Set region all clients must sign for" and `-service` (line 32) "SigV4A credential-scope service all clients must sign with".

Also grep for other in-repo services wrapping requests to iamd: `grep -rln "sigv4client" --include='*.go' | grep -v web/middleware` — any hit that signs _to iamd_ (not to real AWS) must switch to `sigv4aclient` in this commit.

- [ ] **Step 2: Switch the integration tests**

In `cmd/iamd/integration_test.go`, replace `signedTransport` (lines 57–70):

```go
// signedTransport builds a round tripper that signs requests with the given
// credentials for the test region/service.
func signedTransport(t *testing.T, akid, secret string) http.RoundTripper {
	t.Helper()
	rt, err := sigv4aclient.NewSigV4ARoundTripper(&sigv4aclient.Config{
		Region:      intRegion,
		AccessKey:   akid,
		SecretKey:   secret,
		ServiceName: intService,
	}, nil)
	if err != nil {
		t.Fatalf("round tripper: %v", err)
	}
	return rt
}
```

Imports: add `within.website/x/web/middleware/sigv4a/sigv4aclient`. Only transports that authenticate **to iamd** switch to `sigv4aclient`. The signing-key-verifier-chain test keeps `sigv4/iamsts` and classic `sigv4client` for its downstream leg — the classic chain is retained deliberately and this test remains its end-to-end illustration. Read that test carefully before editing: the transport fetching keys _from iamd_ must sign SigV4A, while the client signing requests _to the downstream sigv4/iamsts-protected service_ must keep signing classic SigV4. If both legs currently share the `signedTransport` helper, split it into `signedTransport` (sigv4aclient, for iamd) and `classicSignedTransport` (sigv4client, for the downstream leg).

- [ ] **Step 3: Switch the CLI client**

In `cmd/iamd/pub/iam/iamclient.go`, replace the transport construction (lines 20–28):

```go
	rt, err := sigv4aclient.NewSigV4ARoundTripper(&sigv4aclient.Config{
		Region:      region,
		AccessKey:   accessKeyID,
		SecretKey:   secretAccessKey,
		ServiceName: "iam",
	}, ua)
	if err != nil {
		return nil, err
	}
```

(import swap `sigv4client` → `sigv4aclient`; everything else in the file is unchanged.)

- [ ] **Step 4: Whole-repo check and commit**

```bash
go build ./... && npm test
git add -A
git commit --signoff -m "feat(iamd)!: authenticate callers with sigv4a

BREAKING CHANGE: iamd's middleware now accepts only
AWS4-ECDSA-P256-SHA256 signatures; clients must sign with sigv4aclient
(or sigv4a.Signer) instead of sigv4client."
```

---

### Task 9: Documentation

No code changes. `GetSigningKey`, `SigningKeyService`, `cmd/iamd/services/iam/sts/signingkey.go`, and `web/middleware/sigv4/iamsts/` all remain in place — the classic derived-key chain is kept deliberately as a working illustration of the SigV4 flow.

**Files:**

- Modify: `docs/plans/2026-06-29-sigv4-auth.md`
- Create: `web/middleware/sigv4a/iamsts/integration.md`
- Modify: `web/middleware/sigv4/iamsts/integration.md` (pointer note only)

**Interfaces:**

- Produces (final shape): `SigningKeyService` serves both `GetSigningKey` (classic, illustrative) and `GetPublicKey` (SigV4A, what iamd's own auth chain uses); docs describe both flows and which trust model each carries.

- [ ] **Step 1: Update the trust-model record**

`docs/plans/2026-06-29-sigv4-auth.md`: append a dated update section (do not rewrite the historical record) stating: iamd's own authentication migrated to SigV4A on 2026-07-07 (this plan, `docs/superpowers/plans/2026-07-07-sigv4a-migration.md`); verifiers using `sigv4a/iamsts` receive only ECDSA public keys and cannot mint signatures, so the trust trade-off recorded in the previous update — any authenticated principal could fetch forgery-capable derived keys — no longer applies to the SigV4A chain; `GetSigningKey` and `sigv4/iamsts` are retained deliberately as a working illustration of the classic flow, so that recorded trade-off still applies to any deployment using them; revocation latency for both chains remains bounded by the cache TTL (`-signing-key-cache-ttl`); the classic `sigv4` middleware remains available for AWS-SDK-signed clients.

- [ ] **Step 2: Write the SigV4A integration guide**

Port `web/middleware/sigv4/iamsts/integration.md` (which stays in place) to a new `web/middleware/sigv4a/iamsts/integration.md`, updating every reference: `GetSigningKey` → `GetPublicKey`, "derived signing key" → "public verification key", `sigv4client` transport examples → `sigv4aclient.NewSigV4ARoundTripper`, import paths → `sigv4a/iamsts`, and delete/replace any claim that a leaked response is bounded to one day/region/service (the new statement: a leaked response contains only public material). Then add a two-line note at the top of the _old_ guide: it documents the classic SigV4 derived-key chain, kept for illustration; new services should use `web/middleware/sigv4a/iamsts` (link to the new guide).

- [ ] **Step 3: Format, final verification, commit**

```bash
npm run format && npm test
git add docs/plans/2026-06-29-sigv4-auth.md web/middleware/sigv4a/iamsts/integration.md web/middleware/sigv4/iamsts/integration.md
git commit --signoff -m "docs(sigv4a): record public-key trust model and add iamsts guide"
```

---

## Self-Review (performed while writing)

- **Spec coverage:** algorithm string, KDF (counter layout, N−2 rejection sampling), scope change, X-Amz-Region-Set (signed-header enforcement + wildcard matching), DER/hex signatures, non-deterministic-signature testing strategy, shared-internals extraction (`awssig`, placed under `web/middleware/internal/` for sibling visibility), parallel `sigv4a` package with untouched `sigv4`, public-key distribution (proto, iamd, `sigv4a/iamsts`), `sigv4aclient` transport mirroring `sigv4client`, iamd cutover, classic chain deliberately retained for illustration, docs — each maps to a task.
- **Green-tree check:** T1 is a proven-invisible refactor; T2–T5 and T7 are purely additive; T6 is additive plus the old-iamsts fake stub (Step 3) for the widened twirp interface; T8 flips iamd and all its clients atomically; T9 is docs-only. `npm test` passes at every commit boundary.
- **Type consistency:** `DeriveKeyPair` signature identical across T2/T4/T6; `awssig.BuildCanonicalRequest(r, sortedSignedHeaders, payloadHash, disablePathEscaping)` used by sigv4 (T1), the sigv4a verifier (T4), signer and its tests (T5); `PublicKeyLookuper.LookupPublicKey(ctx, accessKeyID) (*ecdsa.PublicKey, error)` identical in lookuper.go, Verifier, and iamsts; proto `public_key` (PKIX DER) matches `MarshalPKIXPublicKey`/`ParsePKIXPublicKey` on both ends; `sigv4aclient.Config` field names mirror `sigv4client.Config` (`Region/AccessKey/SecretKey/ServiceName`) so T8's swaps are renames.
- **Error-value sharing:** `ErrStreamingUnsupported`/`ErrBodyHash`/`ErrBodyTooLarge` are single values in `awssig` aliased by both packages, so `errors.Is` keeps working for existing `sigv4` callers after T1 and behaves identically in `sigv4a`.
- **Known risk, documented in-line:** individual AWS vectors may trip over Go's `http.ReadRequest` URL handling; the rule (T1 curation note, T4 Step 5) is to remove the vector with a README note, never to add skip logic.
