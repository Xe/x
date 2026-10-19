# Remove Xev API infrastructure

## Context

Xev is discontinued. The `xev` subcommands in `cmd/x` are explicitly retained: they are the CLI for the System One decision API through `within.website/x/web/systemone`. "Xev supporting infrastructure" therefore means the Xev Decision API definition, its generated Go clients/servers, and the dependencies used only by that API. Remove that infrastructure and leave the commands fully functional.

The end state: no `pb/xeiaso`, no `gen/xeiaso`, no dependencies used only by Xev; `cmd/x/cmd/xev` and `cmd/x/main.go` unchanged and still compiling against `web/systemone`.

## Approach

1. Delete the Xev API definition:
   - `pb/xeiaso/net/xev/v1/xev.proto`
   - the now-empty `pb/xeiaso/` directory tree.

2. Delete the generated Xev API package:
   - `gen/xeiaso/net/xev/v1/xev.pb.go`
   - `gen/xeiaso/net/xev/v1/xev_grpc.pb.go`
   - `gen/xeiaso/net/xev/v1/xev.twirp.go`
   - `gen/xeiaso/net/xev/v1/xevv1connect/xev.connect.go`
   - the now-empty `gen/xeiaso/` directory tree.

3. Remove Xev-only buf dependencies:
   - `buf.yaml`: delete `buf.build/googleapis/googleapis` and `buf.build/gnostic/gnostic` from `deps`. The list must contain only `buf.build/bufbuild/protovalidate:v0.10.7`.
   - `buf.gen.yaml`: delete the `disable` entries whose `module` is `buf.build/googleapis/googleapis` and `buf.build/gnostic/gnostic`. Keep the `buf.build/bufbuild/protovalidate` disable entry and all plugin definitions unchanged.
   - `buf.lock`: delete the `- name: buf.build/gnostic/gnostic` block and the `- name: buf.build/googleapis/googleapis` block, including their `commit` and `digest` lines. Keep the `buf.build/bufbuild/protovalidate` block byte-for-byte unchanged.
   - Do not run `buf dep update` unless a later verification step reports the lockfile is inconsistent; the trimmed lockfile is the expected state.

4. Run `go mod tidy` from the repository root. It is authoritative for the module graph:
   - Expected to drop `github.com/google/gnostic`, which was imported only by the deleted `gen/xeiaso/net/xev/v1/xev.pb.go`.
   - `google.golang.org/genproto/googleapis/api` stays in the graph as an indirect requirement because `buf.build/go/protovalidate`, `cel-go`, and gRPC still require it. `github.com/google/gnostic-models` stays as an indirect requirement of the `protoc-gen-connect-openapi` tool. Tidy may move `google.golang.org/genproto/googleapis/api` from the direct block to the indirect block; accept that.
   - It will also apply the repository's pre-existing tidy normalization (`modernc.org/quickjs` direct/indirect placement and stale `go.sum` entries). Keep tidy's output; do not hand-edit `go.mod` or `go.sum` afterward.

5. Do not modify these retained paths:
   - `cmd/x/cmd/xev/noul.go`, `pick.go`, `score.go`, `xev.go`, `xev_test.go`
   - `cmd/x/main.go` (including the `xev` import and the three `subcommands.Register` calls)
   - `web/systemone/` (the client/handler the commands use)

## Critical files & anchors

- `cmd/x/main.go:13,34-37` — the `within.website/x/cmd/x/cmd/xev` import and `Pick`/`Noul`/`Score` registrations that MUST remain.
- `cmd/x/cmd/xev/xev.go:1-28` — package comment and `New()`; the package is backed by `web/systemone`, not the deleted proto.
- `web/systemone/systemone.go:1` — client/handler used by the retained commands; MUST remain.
- `pb/xeiaso/net/xev/v1/xev.proto:1-75` — Xev Decision API; delete the file.
- `buf.yaml:5-8`, `buf.gen.yaml:7-13`, `buf.lock:7-12` — Xev-only buf dependency wiring; trim exactly as described.
- `go.mod:40` — `github.com/google/gnostic` is imported only by the deleted generated package; `go mod tidy` removes it. `google.golang.org/genproto/googleapis/api` remains in the graph through other modules, so it is not expected to disappear.

## Verification

Run from the repository root, in order:

1. `buf build` → exits 0.
2. `buf generate` → exits 0. Then `git status --short` must not show recreated `pb/xeiaso/` or `gen/xeiaso/` paths. If local tool-version drift rewrites unrelated generated files (for example protoc-gen-go version comments or prettier whitespace), restore those paths with `git restore -- <paths>`; they are not part of this change.
3. `go build ./cmd/x` → exits 0.
4. `go test ./cmd/x/cmd/xev ./web/systemone` → passes. This proves the retained commands and their System One client still work.
5. `go run ./cmd/x commands` → output still includes `pick`, `noul`, and `score` alongside the other commands. Expected current output:
   ```text
   help
   flags
   commands
   grpc-health
   ai-add-provider
   ai-list-models
   import-chatgpt
   import-deepseek
   list-switches
   switch
   who-is-front
   pick
   noul
   score
   ```
6. `go test ./...` → passes.
7. `git ls-files | grep -i xev` → lists only the retained CLI files under `cmd/x/cmd/xev/` (no `gen/` or `pb/` entries).
8. `git grep -nE 'xeiaso/net/xev|xeiaso\.net\.xev|xevv1|Xev Decision'` → no matches (exit status 1).
9. `grep -nE 'gnostic|googleapis' buf.yaml buf.gen.yaml buf.lock` → no matches (exit status 1).
10. `go mod why -m github.com/google/gnostic` → reports that the main module does not need the module. `go mod why -m google.golang.org/genproto/googleapis/api` → must not mention `within.website/x/gen/xeiaso`; any remaining path is through `buf.build/go/protovalidate`, `cel-go`, or gRPC.
11. After all checks pass, commit with `git commit --signoff -m "refactor(xev)!: remove Xev decision API"`. The `!` records the breaking removal for semantic-release; the repository requires the signoff.

## Assumptions & contingencies

- The `xev` command group is intentionally retained. Only the API definition, generated bindings, and their unique dependencies are removed. Do not add deprecation warnings or rename the commands/flags; they remain the supported System One CLI.
- `buf.lock` is hand-trimmed to avoid a registry round trip. If `buf build` or `buf generate` reports a lockfile/dependency inconsistency, run `buf dep update`; it should produce the same two-block-removed lockfile.
- If `go test ./...` reports a compile error from a package importing `within.website/x/gen/xeiaso/net/xev/v1`, that is an unaccounted Xev consumer: remove the import and its callsites before rerunning. No such consumer exists in the current tree.
