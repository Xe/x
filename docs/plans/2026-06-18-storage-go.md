# Migrate AWS S3 SDK client construction to `tigrisdata/storage-go`

## Context

The repo constructs raw AWS S3 SDK v2 clients in 8 places and ships its own
`within.website/x/tigris` helper package that wraps the S3 SDK for Tigris. The
official `github.com/tigrisdata/storage-go` library (already an _indirect_
dependency at v0.7.0) now supersedes that custom code: its `storage.Client`
**embeds `*s3.Client`** (so every standard S3 op works unchanged) and its
`tigrisheaders` subpackage contains verbatim copies of every helper in our
`tigris` package, including the `Region` type.

Goal: route all S3 client _construction_ through `storage-go`, delete the
now-redundant `tigris` package, and keep behavior equivalent. Per the user:
migrate **all** clients (including non-Tigris ones — Backblaze B2 and
future-sight's configurable endpoint), and for Tigris clients that currently
resolve their endpoint from the `AWS_ENDPOINT_URL_S3` env var, pin the
**global** endpoint (`storage.WithGlobalEndpoint()` → `t3.storage.dev`).

Most files use the low-level `storage.Client` (embeds `*s3.Client`).
**`cmd/stickers` is the exception**: it switches to the high-level
`storage-go/simplestorage` package and drops every AWS SDK import (see below).

## Key facts that shape the migration

- `storage.Client` embeds `*s3.Client`. Replacing a `*s3.Client` field/var with
  `*storage.Client` is drop-in for `PutObject`/`GetObject`/`HeadObject`/
  `DeleteObject`/`ListObjectsV2` calls — **no call-site changes** beyond the type.
- The `aws-sdk-go-v2/service/s3` and `aws-sdk-go-v2/aws` imports must **stay** in
  files that build `s3.*Input` structs or call `aws.String`/`aws.Int64`.
  Only the construction imports go away: `aws-sdk-go-v2/config` and
  `aws-sdk-go-v2/credentials`.
- `storage.New` credential resolution is backward-compatible: it uses
  `TIGRIS_STORAGE_ACCESS_KEY_ID`/`_SECRET_ACCESS_KEY` if set, otherwise falls
  back to `LoadDefaultConfig` (the existing `AWS_*` env behavior).
- No code in the repo uses the `tigris.WithX` helpers or `Region` type — only
  `tigris.Client`. So no `tigrisheaders` import is needed anywhere; those
  helpers simply move to their library home if ever needed later.

## Per-file changes

### Delete: `tigris/tigris.go` (and the empty `tigris/` dir)

Fully redundant. Its only importer is `cmd/xedn/uplodr/main.go` (updated below).

### `cmd/xedn/uplodr/main.go`

- Imports: drop `within.website/x/tigris` and `aws-sdk-go-v2/credentials`; add
  `storage "github.com/tigrisdata/storage-go"`. Keep `service/s3` + `aws`.
- Struct fields `tc`, `b2c`: `*s3.Client` → `*storage.Client`.
- Tigris client: `tc, err := storage.New(ctx, storage.WithFlyEndpoint())`
  (preserves the existing fly endpoint that the `tigris` package hardcoded;
  xedn deploys on fly.io — this client was never env-driven).
- B2 client: replace `mkB2Client()` with
  `storage.New(ctx, storage.WithEndpoint("https://s3.us-west-001.backblazeb2.com"), storage.WithRegion("us-west-001"), storage.WithPathStyle(true), storage.WithAccessKeypair(*b2KeyID, *b2KeySecret))`.
  This now takes `ctx` and returns an error — fold into `New(ctx)` accordingly.
- `s.tc.PutObject(...)` / `s.b2c.PutObject(...)` call sites unchanged.

### `store/s3api.go`

- Replace `LoadDefaultConfig` + `s3.NewFromConfig` in `NewS3API` with
  `storage.New(ctx, storage.WithGlobalEndpoint())` (path-style stays false /
  vhost, the storage-go default). Drop `aws-sdk-go-v2/config` import.
- `S3API.s3`: `*s3.Client` → `*storage.Client`. Keep `s3`/`aws` imports
  (input structs, `aws.String`).

### `autocert/s3cache/s3cache.go`

- `New`: `storage.New(ctx, storage.WithGlobalEndpoint(), storage.WithPathStyle(true))`
  (preserves the existing path-style=true). Drop `config` import.
- `impl.cli`: `*s3.Client` → `*storage.Client`. (Consumer:
  `cmd/sakurajima/internal/entrypoint/router.go` — no change needed.)

### `cmd/relayd/telemetry.go`

- Replace construction with
  `storage.New(ctx, storage.WithGlobalEndpoint(), storage.WithPathStyle(true))`.
  Drop `config` import; keep `s3`/`aws`.
- `TelemetrySink.s3c`: `*s3.Client` → `*storage.Client`.

### `cmd/uploud/main.go`

- Replace construction with
  `storage.New(ctx, storage.WithGlobalEndpoint(), storage.WithPathStyle(true))`.
  Drop `config` import; keep `s3`/`aws` (PutObjectInput, checksums).
- Local `s3c` var type changes to `*storage.Client`; call sites unchanged.

### `cmd/future-sight/main.go`

- Replace `s3.New(s3.Options{...})` with
  `storage.New(ctx, storage.WithEndpoint(*awsEndpointS3), storage.WithRegion(*awsRegion), storage.WithPathStyle(*usePathStyle), storage.WithAccessKeypair(*awsAccessKeyID, *awsSecretKey))`.
  Keep the existing flags (`--aws-endpoint-url-s3` default `localhost:9000`, etc.).
- Drop `aws-sdk-go-v2/credentials` import. Keep `s3`/`aws` (`aws.String` for keys).
- **Behavior note:** the custom `AppID` user-agent and `ClientLogMode`
  (`LogRetries|LogRequest|LogResponse`) are not exposed by `storage.New` and
  will be dropped — these are cosmetic (UA string + verbose request logging).
  Call this out at implementation; acceptable per "everything via storage-go".

### `cmd/stickers/main.go` — use the high-level `simplestorage` package

This program is the exception: instead of the low-level `storage.Client`, it
uses `github.com/tigrisdata/storage-go/simplestorage`, whose string-keyed `Head`
and `PresignURL` calls replace everything stickers does. This lets it drop
**all** AWS SDK imports.

- Imports: remove `aws-sdk-go-v2/config`, `aws-sdk-go-v2/service/s3`,
  `aws-sdk-go-v2/aws`, and `aws-sdk-go-v2/aws/signer/v4` (`v4`). Add
  `simplestorage "github.com/tigrisdata/storage-go/simplestorage"`. `net/http`
  and `time` are already imported.
- Construction (in `main`):
  `sc, err := simplestorage.New(ctx, simplestorage.WithBucket(*bucketName), simplestorage.WithFlyEndpoint())`.
  **Fly endpoint is required, not global** — the handler rebrands the presigned
  URL via `strings.ReplaceAll(url, "xedn.fly.storage.tigris.dev", "files.xeiaso.net")`,
  which only matches the fly vhost host. `WithBucket(*bucketName)` binds the
  bucket the flag used to pass on every call (default still `xedn`).
- Existence probe: replace the `s3c.HeadObject(&s3.HeadObjectInput{...})` block
  with `if _, err := sc.Head(r.Context(), key); err != nil { ... }` — same
  not-found fallback logic.
- Presigned URL: replace `presigner.GetObject(r.Context(), key, 3600)` with
  `url, err := sc.PresignURL(r.Context(), http.MethodGet, key, time.Hour)`.
  `PresignURL` returns a plain `string`, so `req.URL` becomes `url` directly in
  the `ReplaceAll`/`http.Redirect`.
- **Delete** the `Presigner` struct and its `GetObject` method (lines ~164-190)
  and the `_ = presigner` line — fully superseded by `PresignURL`.
- Net result: `cmd/stickers/main.go` has zero AWS SDK imports.

### `go.mod` / `go.sum`

- `github.com/tigrisdata/storage-go` moves from indirect to direct.
- Run `go mod tidy` — the AWS `config`/`credentials` modules may drop to indirect
  (or out) automatically; `service/s3` and `aws` remain direct.

## Verification

1. `go build ./...` — compiles cleanly.
2. `npm run generate` then `go test ./...` (the repo's `npm test`) — all pass.
   Pay attention to `store`, `autocert/s3cache`, and `cmd/...` packages.
3. `go vet ./...` — no vet issues from the type swaps.
4. `npm run format` — goimports tidies any leftover unused imports + Prettier.
5. Confirm `tigris/` directory is gone and `grep -rn "within.website/x/tigris"`
   returns nothing.
6. Sanity check that no `s3.NewFromConfig` / `s3.New(` / `awsConfig.LoadDefaultConfig`
   construction remains: `grep -rn "NewFromConfig\|LoadDefaultConfig\|s3.New(" --include=*.go`.
7. Confirm `cmd/stickers/main.go` no longer imports any `aws-sdk-go-v2` package,
   and that its presigned-URL rebrand still matches (fly endpoint → `xedn.fly.storage.tigris.dev`).

## Commit (when approved)

Conventional Commit, signed off:

```
refactor(storage): replace AWS S3 SDK client setup with tigrisdata/storage-go

Migrate all S3 client construction to github.com/tigrisdata/storage-go and
remove the redundant within.website/x/tigris package.

Signed-off-by: Xe Iaso <xe@tigrisdata.com>
```
