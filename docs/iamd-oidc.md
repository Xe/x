# OIDC in iamd

`iamd` issues OIDC ID tokens, JWT access tokens, and rotating refresh tokens.
It uses `https://idp.xeserv.us` to authenticate people.
The public issuer is `https://iamd.xeserv.us`.

All provider, browser, and administrative operations run through generated Connect handlers.
Vanguard maps standard HTTP routes to those handlers.
The HTTP bridge preserves OAuth forms, JSON bodies, redirects, cookies, status codes, and errors.

Existing IAM and STS routes retain SigV4/SigV4A authentication.
OIDC access tokens authorize downstream services, not IAM administration.

## Enable the provider

1. Create a confidential OIDC client in the parent provider.
2. Register `https://iamd.xeserv.us/upstream/callback` as its redirect URI.
3. Configure `https://iamd.xeserv.us/upstream/backchannel-logout` as its back-channel logout URI.
4. Permit the `openid`, `profile`, and `email` scopes.
5. Save its secret in a file readable only by the iamd process.
6. Back up the existing SQLite database.
7. Start iamd with these flags:

```sh
iamd \
  -db-loc /data/iamd.db \
  -oidc-enabled \
  -oidc-issuer https://iamd.xeserv.us \
  -oidc-upstream-client-id YOUR_UPSTREAM_CLIENT_ID \
  -oidc-upstream-client-secret-file /run/secrets/iamd/upstream-client-secret
```

Equivalent environment variables use uppercase names and underscores:

```text
OIDC_ENABLED=true
OIDC_ISSUER=https://iamd.xeserv.us
OIDC_UPSTREAM_CLIENT_ID=YOUR_UPSTREAM_CLIENT_ID
OIDC_UPSTREAM_CLIENT_SECRET_FILE=/run/secrets/iamd/upstream-client-secret
```

Mount the secret file through the deployment's secret mechanism.
Do not put its contents in the App manifest or an environment variable.
The supplied App manifest leaves OIDC disabled until this configuration exists.

When enabled, invalid configuration or failed upstream discovery stops startup.
When disabled, iamd does not create or mount its OIDC provider.

Use one replica with the existing SQLite volume.
The database contains signing keys, the token HMAC secret, and authentication state.
Protect database files and backups as credentials.
Back up the entire database, including its SQLite WAL when applicable.
Use SQLite's backup API or stop the process before a filesystem copy.

## HTTP and Connect endpoints

| Standard route                                | Purpose                       |
| --------------------------------------------- | ----------------------------- |
| `GET /.well-known/openid-configuration`       | OIDC discovery                |
| `GET /.well-known/oauth-authorization-server` | OAuth discovery               |
| `GET /jwks.json`                              | Public signing keys           |
| `GET` or `POST /authorize`                    | Authorization code flow       |
| `POST /token`                                 | Code exchange or refresh      |
| `GET` or `POST /userinfo`                     | Scoped user claims            |
| `POST /register`                              | Dynamic client registration   |
| `POST /revoke`                                | Token revocation              |
| `GET /login`                                  | Parent login                  |
| `GET /upstream/callback`                      | Parent callback               |
| `POST /upstream/backchannel-logout`           | Parent logout signal          |
| `GET` or `POST /consent`                      | Consent display or decision   |
| `GET` or `POST /account`                      | Sessions and connected apps   |
| `GET` or `POST /logout`                       | Logout confirmation or action |

OAuth POST requests use `application/x-www-form-urlencoded`.
Registration and administrative document requests use `application/json`.
Public API routes permit CORS without browser credentials for SPA clients.
Browser session and administrative routes do not permit cross-origin API access.

The protobuf defines `ProviderService`, `BrowserService`, and `AdministrationService` in `within.website.x.iam.oidc.v1`.
Their canonical Connect paths remain available beside the standard routes.
`ProtocolRequest` contains OAuth parameters; headers carry authentication and browser cookies.
`DocumentRequest.document` contains registration or administrative JSON.

Native browser read requests return structured page data, including the CSRF value.
Account actions, consent decisions, and logout with a CSRF value perform mutations.

Native Connect responses contain `http_status`, structured JSON in `result`, and the raw representation in `body`.
OAuth failures remain protocol responses, such as `result.error = "invalid_grant"`.
Transport failures use Connect errors.
HTTP clients receive the raw response with its standard status and headers.

See the [protobuf](../pb/within/website/x/iam/oidc/v1/oidc.proto) and [HTTP OpenAPI](api/iamd-oidc.openapi.yaml), and [Connect OpenAPI](api/iamd-oidc-connect.openapi.yaml).
Regenerate them with `npm run generate:buf`.

## Register a client

Registration is open.
It does not grant access to a user's data or create API audiences.
Users approve access before a new client receives tokens.
Registration limits are 10 requests per source address and 100 requests globally per minute.
Behind a proxy, the immediate peer address determines the source limit; forwarded headers are not trusted.

This example registers a public browser client:

```sh
curl --fail-with-body https://iamd.xeserv.us/register \
  -H 'Content-Type: application/json' \
  --data '{
    "client_name": "Example application",
    "redirect_uris": ["https://app.example.com/callback"],
    "token_endpoint_auth_method": "none",
    "grant_types": ["authorization_code", "refresh_token"],
    "response_types": ["code"],
    "scope": "openid profile email offline_access"
  }'
```

Public clients must explicitly request authentication method `none`.
The registration default is `client_secret_basic` for confidential clients.
Save a generated client secret when registration returns it; iamd stores only its hash.
Clients can also use `private_key_jwt` with supported public keys.

Every authorization requires PKCE with `S256`.
Clients must use the same verifier during code exchange.
Use `openid` for OIDC and `offline_access` for refresh tokens.
The authorization flow supports only the `code` response type and `query` response mode.

Native clients use `application_type: "native"`.
They can register HTTPS, IP-loopback HTTP, or reverse-domain private-use redirect schemes.
Loopback ports can vary; all other redirect components must match.

## Define downstream resources

Resource administration requires a signed request from an enabled iamd administrator.
Send signatures for this deployment's configured region and `iam` service.
For alrest, the region is currently `yow`.

`POST /admin/resources` creates or updates a resource:

```json
{
  "audience": "https://api.example.com",
  "name": "Example API",
  "scopes": "read write"
}
```

`GET /admin/resources` lists the catalog.
Clients request a resource with the OAuth `resource` parameter.
Each access token has one resource audience.
The requested scopes must belong to that resource and the client's allowed scopes.
Without an explicit resource, identity-only requests use iamd's UserInfo audience.

Downstream services must check the RS256 signature, `typ: at+JWT`, exact issuer, expected audience, expiry, and required scopes.
Use discovery and JWKS to obtain the issuer's public keys.
An ID token is not an API credential.
Do not derive local privileges from client metadata, profile names, or email addresses.

## Client ID Metadata Documents

CIMD clients use a public HTTPS metadata URL as `client_id` instead of registering first.
The implementation targets [CIMD draft-03](https://datatracker.ietf.org/doc/html/draft-ietf-oauth-client-id-metadata-document-03).
This protocol is an Internet-Draft.

The document must identify the exact client URL and declare supported redirects and authentication.
CIMD supports public clients and asymmetric client authentication; it never accepts shared client secrets.

Metadata fetching rejects redirects and special-use IP addresses.
DNS results are checked before the connection uses a pinned public IP.
Requests have a five-second timeout and a 64-KiB response limit.
The bounded cache defaults to five minutes and caps freshness at one hour.
It respects restrictive cache directives and never stores invalid documents or error responses.
Referenced JWKS documents use the same network restrictions.

Any public HTTPS metadata host can identify a client.
Consent pages label client metadata as unverified.
The server does not fetch client logos.

## Users, consent, and revocation

The stable upstream identity is the verified `(issuer, subject)` pair.
The first successful login creates a non-admin local user without signing keys.
Later logins update upstream profile fields for that same UUID.
Email addresses and display names never link accounts automatically.
Disabled users remain disabled after parent login.

To link an existing user explicitly, an administrator sends `POST /admin/identities`:

```json
{
  "issuer": "https://idp.xeserv.us",
  "subject": "UPSTREAM_SUBJECT",
  "user_id": "LOCAL_USER_UUID"
}
```

Users approve new grants and expanded access.
`/account` lists active sessions and connected applications.
Users can revoke a session or an application's grant there.
Browser mutations require a session-bound CSRF value.

| Credential           | Lifetime                       |
| -------------------- | ------------------------------ |
| Access JWT           | 10 minutes                     |
| ID token             | 10 minutes                     |
| Authorization code   | 5 minutes, single use          |
| Pending login        | 5 minutes, single use          |
| Browser session      | 12 hours                       |
| Refresh token family | 30 days idle; 90 days absolute |

Refresh tokens rotate on every successful use.
Reuse revokes the token family, including tokens from concurrent exchanges.
Clients must serialize refresh requests and save the replacement token before another refresh.

Local logout revokes its browser session and refresh grants.
Validated parent logout signals revoke matching sessions.
Neither operation logs the browser out of the parent provider.
Offline refresh can continue after browser-session expiry until refresh expiry or explicit revocation.
Downstream services can accept already-issued access JWTs until their ten-minute expiry.
iamd checks local revocation state for UserInfo requests.

`GET /admin/clients` lists registered clients without secret hashes.
`POST /admin/clients` disables a registered or CIMD client with this body:

```json
{ "client_id": "CLIENT_ID_OR_METADATA_URL", "disabled": true }
```

Disabling a client revokes its grants.
`POST /admin/keys/rotate` rotates the signing key.
Automatic rotation occurs after 30 days; JWKS retains retired public keys during the token overlap period.

## Operations and rollout

Check discovery, JWKS, a real browser login, consent, token refresh, and logout before routing production clients to this issuer.
The test suite includes a local upstream provider and does not require production credentials.

Metrics include:

- `iamd_oidc_requests_total`: protocol operation and HTTP status.
- `iamd_oidc_request_duration_seconds`: protocol duration.
- `iamd_oidc_refresh_reuse_total`: refresh replay detection.
- `iamd_oidc_registration_rejections_total`: rejected registrations.
- `iamd_oidc_metadata_failures_total`: CIMD lookup failures.

Never log tokens, authorization codes, cookies, or secret headers.
Existing IAM routes remain available when OIDC is disabled.
To stop new OIDC issuance, disable OIDC and restart iamd.
Previously issued JWTs can remain valid until expiry in downstream services that cache JWKS.
Keep the database and key material during rollback.

MCP discovery, token introspection, device grants, client-credentials grants, and multiple replicas are outside this release.
