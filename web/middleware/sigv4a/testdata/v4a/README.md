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

`get-vanilla-query-order-key` was excluded and replaced with its sibling
`get-vanilla-query-order-key-case`: upstream ships `get-vanilla-query-order-key`
with only `context.json` and `request.txt`, missing `public-key.json` and every
signing artifact every other vector has, so it cannot be used to check a
derived key or a signature.
