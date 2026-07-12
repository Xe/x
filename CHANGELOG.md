# [1.31.0](https://github.com/Xe/x/compare/v1.30.0...v1.31.0) (2026-07-12)

### Bug Fixes

- kube ([af5066f](https://github.com/Xe/x/commit/af5066f760329f75d8c929218a7cdad94f3cf94e))
- **kube:** add services lol ([523adfe](https://github.com/Xe/x/commit/523adfe94e1131d6de2d5de1a963ca680fe1ef6b))
- **lint-staged:** remove unnecessary braces from go glob pattern ([29c1e32](https://github.com/Xe/x/commit/29c1e329774cefe93665caa882008911af17f22e))
- **mi:** resolve multiple bugs in cmd/mi ([b8d7138](https://github.com/Xe/x/commit/b8d7138d6c37105e99fc6d5b56b236e562089e04))
- **within.website:** use <pre><code> for go get snippet ([1344e4b](https://github.com/Xe/x/commit/1344e4bd074832fb83e7af74f13bcd8ac8df7869))
- **xess:** drop blockquote left border ([b4536b9](https://github.com/Xe/x/commit/b4536b9baeb4a535561dcd38293c747fe24b6122))

### Features

- add design.within.website demo site for xess ([2331f46](https://github.com/Xe/x/commit/2331f468ca0e669821ea8551f8e32c1ffec48551))
- **kube/alrest:** add lurker ([57e00fe](https://github.com/Xe/x/commit/57e00fe6385287ebe070db4b5988b2b97f29b16d))
- **kube:** import base saga cluster config ([d145106](https://github.com/Xe/x/commit/d14510638a58c9cf05e0d681264dee7d0018a73c))
- **license:** add non-ai licenses ([8ae3e94](https://github.com/Xe/x/commit/8ae3e9447f4799c7e4fb24ce55dda2b9c9861a37))
- **mi/mcp:** add event management tools ([c0c0ef5](https://github.com/Xe/x/commit/c0c0ef585a6c6f7e12bcdb73792fd34af46cde34))
- **mi:** add member name alias support to switch tracker ([#863](https://github.com/Xe/x/issues/863)) ([d3f7e04](https://github.com/Xe/x/commit/d3f7e046ec671a2629991ec839a1d6a213f0e1f9))
- **mi:** add Zoe as nickname for W'zamqo ([585f90f](https://github.com/Xe/x/commit/585f90fe3b019bea930fccbbd551d5f3e7e09318))
- **mimi:** remove falin image generation service ([a3a3e35](https://github.com/Xe/x/commit/a3a3e35fcbfc3e1aeb8cf4f527a7388d7c900929))
- **sigv4:** add AWS SigV4 request authentication and IAM daemon ([#957](https://github.com/Xe/x/issues/957)) ([d2f2e00](https://github.com/Xe/x/commit/d2f2e00f8bddbe7b6bdd36a7354d5f96d8bbed13))
- **skills:** xe-go-style skill added ([c79a11b](https://github.com/Xe/x/commit/c79a11b7345ad5484cb4dfcbf1a3c41fb0f18f14))
- **web:** add alpine.js package ([c0d23b4](https://github.com/Xe/x/commit/c0d23b41d4eb77d0cb9f825b4bd9096897391d1f))
- **xess:** adopt xe-design-system tokens and add button/card/tag components ([1c62860](https://github.com/Xe/x/commit/1c62860923602683cebe8d9eaf5e9a83aba76074))

### BREAKING CHANGES

- **sigv4:** iamsts.NewVerifier is replaced by iamsts.New(Config);
  iamsts.Identity now carries TokenIdentity fields (PrincipalID) instead
  of an iamv1.User.

Signed-off-by: Xe Iaso <me@xeiaso.net>

- test(iamd): cover the signing-key local-verification chain end to end

Signed-off-by: Xe Iaso <me@xeiaso.net>

- refactor(iam)!: delete the per-request STS verification flow

Removes STSService/GetCallerIdentity (proto, generated stubs, iamd
handler, and sigv4.VerifySignature) now that downstream services verify
locally with cached derived signing keys.

The deleted double-slash-path test guarded VerifySignature's synthetic
request construction, which is removed with it; the local path reads
r.URL from net/http's own parser.

- **sigv4:** the STSService Twirp/gRPC/Connect APIs no longer
  exist; downstream verifiers must upgrade to iamsts.New + SigningKeyService.

Signed-off-by: Xe Iaso <me@xeiaso.net>

- docs: remove stale references to the deleted central STS verify flow

Signed-off-by: Xe Iaso <me@xeiaso.net>

- docs(sigv4): record the move to derived signing key caching

Signed-off-by: Xe Iaso <me@xeiaso.net>

- fix(iamsts): detach signing-key fetch from the caller's request context

The singleflight leader in entry() ran the GetSigningKey RPC on the
first caller's request context, so a client disconnecting mid-RPC for
an uncached scope canceled the fetch and failed every request
collapsed onto it with a 500 — an attacker could trigger this burst
deliberately by opening a request and dropping the connection. The
fetch now runs on a context detached from the caller via
context.WithoutCancel (keeping trace/log values) with its own
10s timeout replacing the dropped deadline.

Signed-off-by: Xe Iaso <me@xeiaso.net>

- fix(iamd): reject non-positive -signing-key-cache-ttl at startup

A zero or negative TTL makes every GetSigningKey response effectively
use-once, silently degrading downstream verifiers from a cached
warm-path lookup to an RPC on every request. Fail startup instead of
letting this misconfiguration degrade quietly under load.

Signed-off-by: Xe Iaso <me@xeiaso.net>

- docs(sigv4): record signing-key distribution trust model and current iamd role

Update the components table to reflect that iamd now serves the
signing-key distribution server rather than a planned STS validation
server, mark the superseded forwarded-material security-model section
as historical, and record the trust trade-off the final review
surfaced: any authenticated IAM principal can currently fetch any
principal's derived signing key for the fleet scope, attributed by
caller in logs/metrics, with a verifier allowlist noted as a follow-up
if trust tiers diverge.

Signed-off-by: Xe Iaso <me@xeiaso.net>

- build(iamd): add docker bake target and Dockerfile

Two-stage build matching the repo's existing service images
(golang:1.25 build with cache mounts, debian:bookworm runtime).
The SQLite database lives on a /data volume so state survives
container replacement; built for amd64 and arm64 since the
ncruces SQLite driver is CGO-free. Added to the default bake
group.

Assisted-by: Fable 5 via Claude Code
Signed-off-by: Xe Iaso <me@xeiaso.net>

- docs(iamsts): add twirp integration guide

Shows how to wire the caching verifier into a Twirp API mux:
signed transport for the key fetches, one shared verifier per
process, per-route middleware wrapping, reading the caller
identity, and the operational behaviors (error mapping, outage
semantics, revocation latency, midnight rollover).

Assisted-by: Fable 5 via Claude Code
Signed-off-by: Xe Iaso <me@xeiaso.net>

- chore: commit stuff

Signed-off-by: Xe Iaso <me@xeiaso.net>

- refactor(sigv4): extract shared signing internals into awssig

Signed-off-by: Xe Iaso <me@xeiaso.net>

- feat(sigv4a): add ecdsa p-256 key derivation with aws test vectors

Signed-off-by: Xe Iaso <me@xeiaso.net>

- feat(sigv4a): add x-amz-region-set matching

Signed-off-by: Xe Iaso <me@xeiaso.net>

- fix(awssig): match aws canonicalization for quoted spaces and raw utf-8 paths

collapseSpaces preserved runs of spaces inside double-quoted header
values, but neither aws-sdk-go-v2's v4 signer nor the aws-c-auth v4a
test vectors special-case quotes; both collapse whitespace runs
unconditionally.

CanonicalURI re-percent-encoded r.URL.EscapedPath(), which reproduces
AWS's double-encoding correctly when the wire path is already
percent-encoded ASCII, but fabricates a fresh single-encoded path for
raw, unescaped UTF-8 wire bytes -- encoding that a second time
double-encodes it. Canonicalize the literal wire path (r.RequestURI)
instead, falling back to EscapedPath() for client-constructed
requests that never went through the wire.

Caught by the sigv4a vector suite (get-header-value-trim, get-utf8);
sigv4's SDK round-trip tests and iamd's integration tests pass
unchanged, confirming classic SigV4 compatibility holds.

Signed-off-by: Xe Iaso <me@xeiaso.net>

- feat(sigv4a): add request verifier middleware

Signed-off-by: Xe Iaso <me@xeiaso.net>

- fix(sigv4a): treat nil public key from lookup as unknown key

A PublicKeyLookuper returning (nil, nil) left pub == nil, err == nil,
reaching ecdsa.VerifyASN1(nil, ...), which panics because it
dereferences pub.Curve unconditionally -- a DoS of this auth
middleware once a real KeyLookup is wired in. Treat a nil key as
ErrUnknownKey instead.

Adds coverage for the KeyLookup branch (previously untested) and for
a malformed-hex Signature= value.

Signed-off-by: Xe Iaso <me@xeiaso.net>

- feat(sigv4a): add request signer and sigv4aclient transport

Signed-off-by: Xe Iaso <me@xeiaso.net>

- feat(iamd): serve sigv4a public verification keys via GetPublicKey

Signed-off-by: Xe Iaso <me@xeiaso.net>

- chore: exempt sigv4a test vectors from prettier

npm run format was reformatting the aws-c-auth vector fixtures, which
must stay byte-identical to upstream.

Signed-off-by: Xe Iaso <me@xeiaso.net>

- feat(sigv4a): add iamsts public-key caching verifier

Signed-off-by: Xe Iaso <me@xeiaso.net>

- test(sigv4a/iamsts): make TTL-sweep test table-driven

Restructure the fresh (non-ported) TTL-expiry sweep test into a
table of ordered steps per the repo's go-table-driven-tests
conventions, since it wasn't inherited from the classic package.

Signed-off-by: Xe Iaso <me@xeiaso.net>

- feat(iamd)!: authenticate callers with sigv4a
- **sigv4:** iamd's middleware now accepts only
  AWS4-ECDSA-P256-SHA256 signatures; clients must sign with sigv4aclient
  (or sigv4a.Signer) instead of sigv4client.

Signed-off-by: Xe Iaso <me@xeiaso.net>

- docs(sigv4a): record public-key trust model and add iamsts guide

Signed-off-by: Xe Iaso <me@xeiaso.net>

- fix(twirpslog): restore caller attribution for sigv4a-authenticated services

iamd's UserMiddleware now stores the verified caller via
sigv4a.WithUser, a different context key than the classic
sigv4.WithUser the interceptor checked. Twirp logs and the per-user
billing counter silently lost attribution for every sigv4a-verified
call. Check the sigv4a sources first, falling back to the classic
sigv4/sigv4a.iamsts pair kept for the retained classic chain.

Signed-off-by: Xe Iaso <me@xeiaso.net>

- docs(sigv4a): note twirpslog attribution works for sigv4a callers

Signed-off-by: Xe Iaso <me@xeiaso.net>

- fix(sigv4aclient): set GetBody so redirects and retries can rewind the body

Signed-off-by: Xe Iaso <me@xeiaso.net>

- fix(iamd): read the caller from the sigv4a context in KeyService

The SigV4A cutover left KeyService reading the caller from the classic
sigv4 context key that UserMiddleware no longer populates, so every
KeyService RPC failed closed with an unauthenticated caller.

Signed-off-by: Xe Iaso <me@xeiaso.net>

- feat(iamd): accept both sigv4 and sigv4a signatures

Dispatch on the Authorization header's algorithm token so classic
SigV4 and SigV4A callers both authenticate through the same DAO-backed
credentials; classic traffic stays observable via
iamd_auth_requests_total while it drains.

Signed-off-by: Xe Iaso <me@xeiaso.net>

- refactor(middleware): unify caller identity context in authctx

Two real bugs (a KeyService handler failing closed on the wrong family's key, and a twirpslog attribution regression) came from sigv4 and sigv4a each keeping their own unexported context keys for the same caller identity; this introduces web/middleware/authctx as the single canonical storage that all four packages delegate to, with no exported API changes.

Signed-off-by: Xe Iaso <me@xeiaso.net>

- feat(sigv4any): add dual-algorithm dispatch middleware

Lift the Authorization-header dispatch pattern (classic SigV4 vs SigV4A)
into a reusable middleware so any service can accept either algorithm
without hand-rolling the switch, while keeping metric wiring pluggable
via an Observe hook instead of a hard prometheus dependency.

Signed-off-by: Xe Iaso <me@xeiaso.net>

- refactor(iamd): use sigv4any for dual-algorithm auth

Replace iamd's hand-rolled Authorization-header dispatch with the new
sigv4any.Verifier, keeping the iamd_auth_requests_total counter wired
through Observe. cmd/iamd/integration_test.go is the equivalence proof
and passes unchanged.

Signed-off-by: Xe Iaso <me@xeiaso.net>

- fix(iam): wrap errors from DisableKey and ListKeys

The disable command returned the raw DisableKey error and had a
duplicated dead err check left over from a copy-paste; the list
command never checked the ListKeys error at all. Wrap both so
failures surface a descriptive message.

Assisted-by: Claude Opus 4.8 via Claude Code
Signed-off-by: Xe Iaso <me@xeiaso.net>

- fix(iam): wrap error from ListUsers

The user list command ignored the ListUsers error and proceeded
to render an empty result. Check and wrap it so failures surface
a descriptive message, matching the keys command.

Assisted-by: Claude Opus 4.8 via Claude Code
Signed-off-by: Xe Iaso <me@xeiaso.net>

- chore(saga): add iamd and route53 dyndns manifests plus sigv4a plan

Deploy iamd to the saga cluster with sigv4a signing-key cache
config, add a route53 dynamic DNS deployment, and check in the
sigv4a migration plan doc.

Assisted-by: Claude Opus 4.8 via Claude Code
Signed-off-by: Xe Iaso <me@xeiaso.net>

- fix(saga): remove invalid value on AWS_SECRET_ACCESS_KEY env var

The env var set both valueFrom.secretKeyRef and a literal
value, which Kubernetes rejects. Drop the leftover placeholder
value so the manifest applies.

Assisted-by: Claude Opus 4.8 via Claude Code
Signed-off-by: Xe Iaso <me@xeiaso.net>

- chore: nuke sins

Signed-off-by: Xe Iaso <me@xeiaso.net>

# [1.30.0](https://github.com/Xe/x/compare/v1.29.0...v1.30.0) (2026-02-16)

### Bug Fixes

- **falin:** resolve npm ci dependency conflict ([#835](https://github.com/Xe/x/issues/835)) ([683bff8](https://github.com/Xe/x/commit/683bff8e5a484f7b4f2a4b506380501fda204720))
- **mi:** use blog post summary in Bluesky embed description ([#848](https://github.com/Xe/x/issues/848)) ([7491810](https://github.com/Xe/x/commit/7491810e2481fd6be8f4baec97a08e4fd758b2f8))
- **nguh:** return error for unsupported tokens ([d1a50e7](https://github.com/Xe/x/commit/d1a50e7a1604c1ecb831be9199365f39c22218a2))
- **skills/xe-writing-style:** update details about successive paragraph starting letter rule ([5808b2b](https://github.com/Xe/x/commit/5808b2b2100dbcac2f1ec63800909d9cc6588ef2))
- **skills/xe-writing-style:** wumbofy this with Opus ([cea6609](https://github.com/Xe/x/commit/cea6609b7f020b682136817604b40ebd8bf1ce61))
- **useragent:** use filepath.Base for os.Args[0] in GenUserAgent ([#830](https://github.com/Xe/x/issues/830)) ([3ef21d9](https://github.com/Xe/x/commit/3ef21d9a605ac0f1607a71677d233d11f1b64187))
- **web:** replace deprecated io/ioutil with io ([#829](https://github.com/Xe/x/issues/829)) ([fee5e4f](https://github.com/Xe/x/commit/fee5e4f1ce31e9bd5b6e2331eddebfd7fa015a3e))

### Features

- **cmd/x:** add ai-add-provider and ai-list-models subcommands ([#850](https://github.com/Xe/x/issues/850)) ([bba7f41](https://github.com/Xe/x/commit/bba7f419c26bc3a099f306cd4006f9e5a86dd8e6))
- **python:** accept io/fs.FS as root filesystem parameter ([#813](https://github.com/Xe/x/issues/813)) ([87b97e8](https://github.com/Xe/x/commit/87b97e8fadaecb2efcbd1e87d6990440b61ddd8a))
- **reviewbot:** add Python interpreter with repo filesystem ([#814](https://github.com/Xe/x/issues/814)) ([b40ff1c](https://github.com/Xe/x/commit/b40ff1cadba0c5aeec5936ef5134ec900d80f310))
- **sakurajima:** add HTTP request timeouts to prevent hanging connections ([#837](https://github.com/Xe/x/issues/837)) ([d50a792](https://github.com/Xe/x/commit/d50a7922e614feaa51d8d593f17d7cc03ffbcc1e))
- **sakurajima:** add request size limits to prevent DoS attacks ([#838](https://github.com/Xe/x/issues/838)) ([f207855](https://github.com/Xe/x/commit/f2078551e8f2cf28d7450e396715873f5acb1607))
- **sakurajima:** add request size limits to prevent DoS attacks ([#839](https://github.com/Xe/x/issues/839)) ([80dd84a](https://github.com/Xe/x/commit/80dd84ad36e8acd028812ae9bd032b901b2524a1))
- **sakurajima:** production readiness fixes and enhancements ([#834](https://github.com/Xe/x/issues/834)) ([4368e6f](https://github.com/Xe/x/commit/4368e6fd401f7d883a26e3eadec1eb685e5aa4fd))
- **sapientwindex:** add state to prevent double-posts ([#825](https://github.com/Xe/x/issues/825)) ([6ba9223](https://github.com/Xe/x/commit/6ba922341741c20bac2beb48e4103bc06a3c2036))
- **skills:** add experimental Xe writing style skill ([baed3bd](https://github.com/Xe/x/commit/baed3bd228a40d5be584dbdff4d8b300be85e10e))
- **skills:** add Go table-driven tests skill ([#817](https://github.com/Xe/x/issues/817)) ([a2e35ea](https://github.com/Xe/x/commit/a2e35ead52b7d3b0849129a9b399b0f2f7e4e283))
- **store:** add filesystem backends (DirectFile, JSONMutexDB, CAS) ([#824](https://github.com/Xe/x/issues/824)) ([4f694cf](https://github.com/Xe/x/commit/4f694cfaa9efe8754f68e2871aeb5d27576c5433))
- **totpgen:** add TOTP code generator command ([#833](https://github.com/Xe/x/issues/833)) ([d0a556d](https://github.com/Xe/x/commit/d0a556d6aa204eec74ffe5cb4b8fbe5496452d89))

### BREAKING CHANGES

- **python:** llm/codeinterpreter/python.Run() now takes fs.FS as first parameter

Assisted-by: GLM 4.6 via Claude Code
Reviewbot-request: yes

Signed-off-by: Xe Iaso <me@xeiaso.net>

# [1.29.0](https://github.com/Xe/x/compare/v1.28.0...v1.29.0) (2026-01-13)

### Features

- add cmd/readability ([8043d20](https://github.com/Xe/x/commit/8043d209826d0e038ebd991b32806de9424a4cae))

# [1.28.0](https://github.com/Xe/x/compare/v1.27.0...v1.28.0) (2026-01-10)

### Bug Fixes

- **reviewbot:** add desired output format to the system prompt ([7cb3f98](https://github.com/Xe/x/commit/7cb3f98b2e68f975a4902902c0f5c2987bcbecc1))

### Features

- add reviewbot prototype ([1ea29dd](https://github.com/Xe/x/commit/1ea29dde9e7f984e745d07b4343b6b8366c4b9ae))
- **reviewbot:** add auto-trigger via commit footer ([91726b2](https://github.com/Xe/x/commit/91726b2e5827b2dfe8ae11cedee59bf4a4e96d0b))
- **reviewbot:** if no tool call, default to leaving a PR comment ([#809](https://github.com/Xe/x/issues/809)) ([c530dd7](https://github.com/Xe/x/commit/c530dd777dde5fdc4efdd79a4cf0ac509d9128ab))
- **skills:** add templ-htmx skill using local htmx package ([ad16a19](https://github.com/Xe/x/commit/ad16a19c9810cfb0ae52d122158738cea7eec154))

# [1.27.0](https://github.com/Xe/x/compare/v1.26.1...v1.27.0) (2026-01-08)

### Bug Fixes

- **package.json:** use Signed-off-by instead of Signed-Off-By ([c2406e3](https://github.com/Xe/x/commit/c2406e3de87ab486e37b68e33b044288a704e5bd))
- **yeetfile:** build sysexts ([c9b3ab7](https://github.com/Xe/x/commit/c9b3ab7f70954de56174ed1b634c0d224a8bdbee))

### Features

- add 7-day cooldown to dependabot updates ([#791](https://github.com/Xe/x/issues/791)) ([1a5d9d3](https://github.com/Xe/x/commit/1a5d9d3fa49558fd483c1d2b29c40f842f1f3186))
- add attention attenuator package ([6ca9c01](https://github.com/Xe/x/commit/6ca9c014acb21fa808f20222f9383be8a90c44cb))
- add claude-code-cfg package ([73798a3](https://github.com/Xe/x/commit/73798a312ee21cda529f527370d7c1ca60a8bc64))
- **kube/alrest:** add mcp-auth sidecar ([e56238b](https://github.com/Xe/x/commit/e56238bb3bcc0870b84e4bfbc10e04c99715ccdc))
- **mi:** add birthday field to Member model ([#784](https://github.com/Xe/x/issues/784)) ([69bff8e](https://github.com/Xe/x/commit/69bff8e99e082cfc3e859cfc03bfc597cc36c16a))
- require Signed-off-by in commit messages ([#785](https://github.com/Xe/x/issues/785)) ([7e1c910](https://github.com/Xe/x/commit/7e1c910b5d458f5b7a9e21ebe7272ddb1aae68d1))

## [1.26.1](https://github.com/Xe/x/compare/v1.26.0...v1.26.1) (2025-11-11)

### Bug Fixes

- **mcp:** fix list-system-members output schema ([2a24290](https://github.com/Xe/x/commit/2a2429068fcdbd7dd5f22e0a47caa7037fdec4fb))

# [1.26.0](https://github.com/Xe/x/compare/v1.25.0...v1.26.0) (2025-11-09)

### Features

- add list-system-members MCP tool ([#781](https://github.com/Xe/x/issues/781)) ([3d14b73](https://github.com/Xe/x/commit/3d14b73136855e4444549f6cce866547625fd62e))

# [1.25.0](https://github.com/Xe/x/compare/v1.24.0...v1.25.0) (2025-10-30)

### Bug Fixes

- a bunch of things ([113cde3](https://github.com/Xe/x/commit/113cde35bba091dada6c83f7e40c74dcd0d181a0))
- bump versions of things in kube ([94ef124](https://github.com/Xe/x/commit/94ef124e9eae053d1b9874208eed99854a613196))
- **httpdebug:** log request info ([b7c3bcb](https://github.com/Xe/x/commit/b7c3bcb18d682e30935b6c803b480b2b8fb9c7e6))
- oops ([1796967](https://github.com/Xe/x/commit/17969677917010bdfe3d7aa99cfd8ead189a00fc))
- **sakurajima:** fix tests ([5549686](https://github.com/Xe/x/commit/55496863add61e24c7e6edc1d05288babd52cfc9))

### Features

- add httpdebug Docker target and Dockerfile ([b428337](https://github.com/Xe/x/commit/b428337f8e177ba51110c9895d141542c66c62b2))
- add shiroko k8s ([0172f70](https://github.com/Xe/x/commit/0172f7082b11b4b177ba634461d590296d2a5d53))
- add W'zamqo as valid member name and update suggestions ([d0fff60](https://github.com/Xe/x/commit/d0fff604962700ba5ed185c4828f81db3443b07d))
- **cmd/httpdebug:** enhance security and functionality ([0c0f8ea](https://github.com/Xe/x/commit/0c0f8ea72feab23cbe5ed667e0dd283b1fa77914))
- **cmd/mi:** add MCP server for completely bad ideas ([673dbbd](https://github.com/Xe/x/commit/673dbbde5dd0f61ad7d06239186d49aef732227c))
- cmd/sakurajima ([86de439](https://github.com/Xe/x/commit/86de439c6da51f1751a2c72b79a584ceba72d3ef))
- **cmd/sakurajima:** add autocert config settings ([0017dc0](https://github.com/Xe/x/commit/0017dc0eeab6e42dbc1735ea6a18bf509d36a28f))
- **cmd/x:** fix switch command ([616d8b4](https://github.com/Xe/x/commit/616d8b452be147e9994f292f446ddf49d7f6773d))
- **cmd/x:** port list-switches command over ([edd2e48](https://github.com/Xe/x/commit/edd2e48be7437fe43ea053d11372840199b0caab))
- **cmd:** add x-browser-validation test program ([8c0154e](https://github.com/Xe/x/commit/8c0154e9fd798cdeec0e3f6ad66f287bdae9e7b7))
- **sakurajima:** add the rigging for log filtering logic ([0274cf3](https://github.com/Xe/x/commit/0274cf3e4cdfbe908fb8602d980b5b1636f5e6fb))
- **sakurajima:** implement access logs / logrotate logic ([8b68476](https://github.com/Xe/x/commit/8b68476147a011a8deebf6aa0b8f82770397ae67))
- **sakurajima:** log filters ([b4a5f77](https://github.com/Xe/x/commit/b4a5f77dc13da12437865bf8b5dbed60e0611685))
- **stickers:** use branded presigned URLs ([5223d77](https://github.com/Xe/x/commit/5223d77df6b10ff75d6a0dce4b674a989562b833))

# [1.24.0](https://github.com/Xe/x/compare/v1.23.0...v1.24.0) (2025-07-15)

### Features

- **cmd/httpdebug:** add HTTP method and URI to output ([158fb5b](https://github.com/Xe/x/commit/158fb5b9c4883aed25bf38d1893cf1a84d0db76c))
- **cmd/relayd:** ja4t / ja4h fingerprinting ([e810bdb](https://github.com/Xe/x/commit/e810bdb0c432df273e7cc6f0885273daae5faea6))
- **fish-config:** build a deb too I guess ([063d5c0](https://github.com/Xe/x/commit/063d5c0dd6cefbf24aab2bcebd02003f65994c5d))
- **fish-config:** dotenv.fish ([7c8b929](https://github.com/Xe/x/commit/7c8b92988925a9db748a653ef62fde5fb16f36d3))
- **pb/relayd:** add fingerprints object ([2fc470c](https://github.com/Xe/x/commit/2fc470cb7ed3c887f5feaf3d10a88265dda1d685))

# [1.23.0](https://github.com/Xe/x/compare/v1.22.0...v1.23.0) (2025-06-28)

### Bug Fixes

- **fish-config:** make fish prompt compatible with vs code ([484b614](https://github.com/Xe/x/commit/484b614b607f0a5b610dd6186a07aaa547873ea1))
- **fish-prompt:** clean up theme ([d00dabd](https://github.com/Xe/x/commit/d00dabd9e9962a0950837a9a0eb2d419471ad00c))
- **mi:** enable reflection ([b264671](https://github.com/Xe/x/commit/b264671471cc62a4696442de204085808e628130))

### Features

- **pkgs/fish-config:** add autols, autopair, fisher, and nvm ([2a8ccb5](https://github.com/Xe/x/commit/2a8ccb5e9235262aa0db262caa46061b174472c9))

# [1.22.0](https://github.com/Xe/x/compare/v1.21.0...v1.22.0) (2025-06-11)

### Bug Fixes

- **cmd/relayd:** set the right Host header ([6db9999](https://github.com/Xe/x/commit/6db99999b5de18ae9da2cb995473141b96a46fae))
- **mi:** make old routes work to avoid breaking tools ([b51ae17](https://github.com/Xe/x/commit/b51ae174ba2eb47a4d2d06db706dbee6969a5444))
- **yeetfile:** only build uploud where purego is supported ([8c12cfa](https://github.com/Xe/x/commit/8c12cfa67be87d5f1cfd8a5315160a5197f22ac1))

### Features

- **ingressd:** grpc health checking ([0870e5d](https://github.com/Xe/x/commit/0870e5df3513789b16fc33a588f01370dad55a76))

# [1.21.0](https://github.com/Xe/x/compare/v1.20.0...v1.21.0) (2025-05-31)

### Bug Fixes

- **pvfm/bot:** cache seen messages for up to 10 minutes ([76268a9](https://github.com/Xe/x/commit/76268a97a921358579b8c96feefd4a36f01f6478))
- **recording:** don't delete temporary files ([472754d](https://github.com/Xe/x/commit/472754d2f38c51e6dd77b29c4faf645fb2311caf))

### Features

- **aura:** build with docker buildx bake ([126a3cf](https://github.com/Xe/x/commit/126a3cf9934d4e07647ec08cd6d96ab27f1f9eca))
- **x:** grpchc can do TLS now ([cc4be82](https://github.com/Xe/x/commit/cc4be8203f35b67d58b9d1edf0c2c4ab363e0a96))

# [1.20.0](https://github.com/Xe/x/compare/v1.19.0...v1.20.0) (2025-05-30)

### Features

- **yeetfile:** build riscv64 binaries ([aa0c069](https://github.com/Xe/x/commit/aa0c069cd638b4d3670385f3b9b0934eb186b720))

# [1.19.0](https://github.com/Xe/x/compare/v1.18.0...v1.19.0) (2025-05-29)

### Bug Fixes

- **cmd/relayd:** use X-Http-Protocol for HTTP protocol version ([e73bf5f](https://github.com/Xe/x/commit/e73bf5ffe9c36d7ffb2d8021795503edf0a3a156))
- **yeetfile:** new docker image ([5172ad9](https://github.com/Xe/x/commit/5172ad944aebe6ff44962848333d4912c744d1aa))

### Features

- **cmd/license:** move licenses to go templates ([c01ca48](https://github.com/Xe/x/commit/c01ca4894a0b04ac1cc12d24eda85670f4eaa625))
- **fish-config:** kubernetes info in prompt ([66924c2](https://github.com/Xe/x/commit/66924c2e03d4c0f8860c73875caa24561097aefe))
- **pkgs:** add fish config as a package ([93fd623](https://github.com/Xe/x/commit/93fd6232eb3b5ea22ed4666ec4e2c18b4765464a))
- **x:** grpc health checking ([a78e6cf](https://github.com/Xe/x/commit/a78e6cf59287d7bdf16d21272031d58bfdd38583))
- **yeetfile:** ppc64le binaries ([2fbd92f](https://github.com/Xe/x/commit/2fbd92f092075a26bfd5496742a51a421d72cf6e))

# [1.18.0](https://github.com/Xe/x/compare/v1.17.0...v1.18.0) (2025-05-05)

### Bug Fixes

- **gitea:** fix deployment ([b418d50](https://github.com/Xe/x/commit/b418d50b5d444916e71b06f295c79a10731e6211))
- **relayd:** increase correlation potential ([e047ce3](https://github.com/Xe/x/commit/e047ce314d7ea8b6ec398c1cb633e60ea61f06dc))
- **relayd:** make max bundle size 512 ([915e301](https://github.com/Xe/x/commit/915e3019e03ecb58acc833a503a3708c10b4456d))
- **relayd:** move request ID creation to a variable ([d030f4f](https://github.com/Xe/x/commit/d030f4fcb286791bb1b8be0bdf0d9f6193311b56))

### Features

- **relayd:** change log persistence method ([b3bb404](https://github.com/Xe/x/commit/b3bb404331ddf11b94b1d46a8567308903f94e4c))
- **web:** add iptoasn client ([c53fd2a](https://github.com/Xe/x/commit/c53fd2aad2e65fa362cab2a784488e83d6a9bfb3))

# [1.17.0](https://github.com/Xe/x/compare/v1.16.0...v1.17.0) (2025-04-27)

### Features

- **httpdebug:** quiet mode and function as a systemd service ([1d9fa34](https://github.com/Xe/x/commit/1d9fa34fa84cc125c68ab486d8bbc2dbe7a51f0e))
- **relayd:** autocert support for automatic TLS cert minting ([c9136cc](https://github.com/Xe/x/commit/c9136cc167ca0bbabce1196f88cfc1b302350f0a))
- **xess:** add fancy 404 page ([10e176a](https://github.com/Xe/x/commit/10e176a023ee1a4955160c86f0dc71a435bdf866))

# [1.16.0](https://github.com/Xe/x/compare/v1.15.0...v1.16.0) (2025-04-27)

### Bug Fixes

- **gitea:** use >- instead of > ([972cc99](https://github.com/Xe/x/commit/972cc990716c8593fc1f1d7061e6b707c6bccc51))

### Features

- **relayd:** store and query TLS fingerprints ([ef94cbc](https://github.com/Xe/x/commit/ef94cbcc7f9f90ef5c238413ee3305c305743a42))

# [1.15.0](https://github.com/Xe/x/compare/v1.14.1...v1.15.0) (2025-04-27)

### Features

- **ci:** allow automatically cutting a new release via messages ([b12801a](https://github.com/Xe/x/commit/b12801a2445bbaa8840acd00d76653100a4f6bbe))

## [1.14.1](https://github.com/Xe/x/compare/v1.14.0...v1.14.1) (2025-04-27)

# [1.14.0](https://github.com/Xe/x/compare/v1.13.6...v1.14.0) (2025-04-27)

### Bug Fixes

- **relayd:** disable TCP fingerprinting on Linux for now ([6aa26b7](https://github.com/Xe/x/commit/6aa26b7defa02515fcc8473b8c8603e5fbe45f3f))
- **relayd:** rename HTTP headers for fingerprints ([b64f843](https://github.com/Xe/x/commit/b64f8430190d0a49f8ec6a105e2978714342dd3e))

### Features

- **anubis:** replace with tombstone ([929e2de](https://github.com/Xe/x/commit/929e2debb8b9a63c44e3bb02387a6774821ccb99))
- cmd/aws-secgen for generating fake AWS secrets ([7b8662a](https://github.com/Xe/x/commit/7b8662a0a877fd708afc679b4898e0a54343fe7a))
- **relayd:** add standard reverse proxy headers ([33ebd25](https://github.com/Xe/x/commit/33ebd254071288ae5925b39cc59c3aba67cce499))
- **relayd:** ja4t fingerprinting ([8ecbe6f](https://github.com/Xe/x/commit/8ecbe6f42e0eed79e899178570690aab1ce67c3f))
