# 🧰 Troubleshooting

> Last updated: 2026-02-20

Use this guide for common setup and runtime errors.

## 🧭 Decision Tree

Use this quick route before diving into detailed sections:

1. `curl http://localhost:8080/health` fails -> go to `Database connection failure` and `Migration failure`
2. Token endpoint (`POST /v1/token`) returns `401`/`403`/`429` -> go to `401 Unauthorized`, `429 Too Many Requests`, or `Token issuance fails with valid-looking credentials`
3. API requests return `403` with valid token -> go to `403 Forbidden` (policy/capability mismatch)
4. API requests return `422` -> go to `422 Unprocessable Entity` (payload/query format)
5. API requests return `429` -> go to `429 Too Many Requests` (rate limiting)
6. Browser calls fail before API handler -> go to `CORS and preflight failures`
7. After rotating keys behavior is stale -> go to `Rotation completed but server still uses old key context`
8. Startup fails with key config errors -> go to `Missing or Invalid Master Keys` and `KMS configuration mismatch`
9. Monitoring data is missing -> go to `Metrics Troubleshooting Matrix`
10. Tokenization endpoints fail after upgrade -> go to `Tokenization migration verification`
11. Master key loads but key-dependent crypto fails after mixed-version rollout -> go to `Master key load regression triage (historical v0.5.1 fix)`

## 📑 Table of Contents

- [401 Unauthorized](#401-unauthorized)
- [403 Forbidden](#403-forbidden)
- [409 Conflict](#409-conflict)
- [422 Unprocessable Entity](#422-unprocessable-entity)
- [429 Too Many Requests](#429-too-many-requests)
- [CORS and preflight failures](#cors-and-preflight-failures)
- [CORS smoke checks (copy/paste)](#cors-smoke-checks-copypaste)
- [Database connection failure](#database-connection-failure)
- [Migration failure](#migration-failure)
- [Missing or Invalid Master Keys](#missing-or-invalid-master-keys)
- [KMS configuration mismatch](#kms-configuration-mismatch)
- [Mode mismatch diagnostics](#mode-mismatch-diagnostics)
- [KMS authentication or decryption failures](#kms-authentication-or-decryption-failures)
- [Master key load regression triage (historical v0.5.1 fix)](#master-key-load-regression-triage-historical-v051-fix)
- [Missing KEK](#missing-kek)
- [Metrics Troubleshooting Matrix](#metrics-troubleshooting-matrix)
- [Tokenization migration verification](#tokenization-migration-verification)
- [Rotation completed but server still uses old key context](#rotation-completed-but-server-still-uses-old-key-context)
- [Token issuance fails with valid-looking credentials](#token-issuance-fails-with-valid-looking-credentials)
- [Policy matcher FAQ](#policy-matcher-faq)
- [Quick diagnostics checklist](#quick-diagnostics-checklist)

## 401 Unauthorized

- Symptom: API returns `401 Unauthorized`
- Likely cause: missing/invalid token, expired token, or bad client credentials
- Fix:
  - request a fresh token via `POST /v1/token`
  - ensure header format is `Authorization: Bearer <token>`
  - verify client is active

## 403 Forbidden

- Symptom: token is valid, but operation returns `403 Forbidden`
- Likely cause: policy does not grant required capability on requested path
- Fix:
  - verify capability mapping for endpoint (`read`, `write`, `delete`, `encrypt`, `decrypt`, `rotate`)
  - verify path pattern (`*`, exact path, trailing wildcard `/*`, or mid-path wildcard like `/v1/transit/keys/*/rotate`)
  - avoid unsupported wildcard patterns (partial-segment `prod-*`, suffix/prefix `*prod`/`prod*`, and `**`)
  - validate concrete matcher examples:
    - `/v1/transit/keys/*/rotate` matches `/v1/transit/keys/payment/rotate`
    - `/v1/transit/keys/*/rotate` does not match `/v1/transit/keys/payment/extra/rotate`
  - update client policy and retry

Common false positives (`403` vs `404`):

- `404 Not Found` usually means route shape mismatch (endpoint path does not exist).
- `403 Forbidden` usually means route exists but caller policy/capability denies access.
- Validate route shape first, then evaluate policy matcher and capability mapping.

## 409 Conflict

- Symptom: request returns `409 Conflict`
- Likely cause: resource already exists with unique key constraints

Common 409 case:

| Endpoint | Common cause | Fix |
| --- | --- | --- |
| `POST /v1/transit/keys` | transit key `name` already has initial `version=1` | use `POST /v1/transit/keys/:name/rotate` for a new version, or pick a new key name |

- Fix:
  - use create only for first-time key initialization
  - use rotate for subsequent key versions
  - migration note: if legacy automation retries create for existing names, update it to call rotate
    after receiving `409 Conflict`

## 422 Unprocessable Entity

- Symptom: request rejected with `422`
- Likely cause: malformed JSON, invalid query params, missing required fields

Common 422 cases:

| Endpoint | Common cause | Fix |
| --- | --- | --- |
| `POST /v1/secrets/*path` | `value` is missing or not base64 | Send `value` as base64-encoded bytes |
| `POST /v1/transit/keys/:name/encrypt` | `plaintext` is missing or not base64 | Send `plaintext` as base64-encoded bytes |
| `POST /v1/transit/keys/:name/decrypt` | `ciphertext` not in `<version>:<base64-ciphertext>` format | Pass `ciphertext` exactly as returned by encrypt |
| `GET /v1/audit-logs` | invalid `offset`/`limit`/timestamp format | Use numeric `offset`/`limit` and RFC3339 timestamps |

- Fix:
  - validate JSON body and required fields
  - for secrets/transit endpoints, send base64 values where required
  - for transit decrypt, pass `ciphertext` exactly as returned by encrypt (`<version>:<base64-ciphertext>`)
  - validate `offset`, `limit`, and RFC3339 timestamps on audit endpoints

## 429 Too Many Requests

- Symptom: authenticated requests return `429`
- Likely cause: per-client rate limit exceeded on authenticated endpoints, or per-IP token endpoint rate limit exceeded on `POST /v1/token`
- Fix:
  - check `Retry-After` response header and back off before retrying
  - implement exponential backoff with jitter in client retry logic
  - reduce request burst/concurrency from caller
  - tune `RATE_LIMIT_REQUESTS_PER_SEC` and `RATE_LIMIT_BURST` if traffic is legitimate
  - for `POST /v1/token`, tune `RATE_LIMIT_TOKEN_REQUESTS_PER_SEC` and `RATE_LIMIT_TOKEN_BURST` if callers share NAT/proxy egress

Trusted proxy checks for token endpoint (`POST /v1/token`):

- If many callers suddenly look like one IP, verify proxy forwarding and trusted proxy settings
- If `X-Forwarded-For` is accepted from untrusted sources, IP spoofing can bypass intended per-IP controls
- Compare application logs (`client_ip`) with edge proxy logs to confirm real source-IP propagation
- Use [Trusted proxy reference](../operations/trusted-proxy-reference.md) for a platform checklist

Quick note:

- Authenticated rate limiting applies to `/v1/clients`, `/v1/secrets`, `/v1/transit`, `/v1/tokenization`, and `/v1/audit-logs`
- IP-based rate limiting applies to token issuance (`POST /v1/token`)
- Rate limiting does not apply to `/health`, `/ready`, and `/metrics`

## CORS and preflight failures

- Symptom: browser requests fail on preflight (`OPTIONS`) or show CORS errors in console
- Likely cause: CORS disabled (default) or origin not listed in `CORS_ALLOW_ORIGINS`
- Fix:
  - keep `CORS_ENABLED=false` for server-to-server usage
  - if browser access is required, set `CORS_ENABLED=true`
  - configure explicit origins in `CORS_ALLOW_ORIGINS` (comma-separated, no wildcard in production)
  - confirm request origin exactly matches configured origin (scheme/host/port)

Quick checks:

- If token call succeeds from backend but browser fails before handler, this is usually CORS, not auth policy
- `403 Forbidden` indicates authorization policy denial; CORS failures usually happen at browser layer

### CORS behavior matrix

| Browser scenario | Expected result | Common misconfiguration |
| --- | --- | --- |
| `CORS_ENABLED=false`, same-origin app | Works (no cross-origin checks) | N/A |
| `CORS_ENABLED=false`, cross-origin app | Browser blocks request | Expecting browser access without enabling CORS |
| `CORS_ENABLED=true`, origin listed | Preflight and request succeed | Wrong scheme/port in origin list |
| `CORS_ENABLED=true`, origin missing | Browser blocks request | Origin not included in `CORS_ALLOW_ORIGINS` |
| `CORS_ENABLED=true`, wildcard in production | Works but insecure | Overly broad origin trust |

## CORS smoke checks (copy/paste)

Preflight request check:

```bash
curl -i -X OPTIONS http://localhost:8080/v1/clients \
  -H "Origin: https://app.example.com" \
  -H "Access-Control-Request-Method: GET" \
  -H "Access-Control-Request-Headers: Authorization,Content-Type"
```

Expected when CORS is enabled and origin is allowed:

- `204`/`200` preflight response
- `Access-Control-Allow-Origin: https://app.example.com`
- `Access-Control-Allow-Methods` includes requested method

Simple cross-origin request header check:

```bash
curl -i http://localhost:8080/health \
  -H "Origin: https://app.example.com"
```

If CORS is disabled or origin is not allowed, browser requests can fail even if raw curl succeeds.

## Database connection failure

- Symptom: app fails at startup or migration with DB connection errors
- Likely cause: wrong connection string, unreachable DB host, wrong credentials
- Fix:
  - check `DB_DRIVER` and `DB_CONNECTION_STRING`
  - ensure DB container/service is running and reachable
  - if Docker network is used, ensure host in connection string matches service/container name

## Migration failure

- Symptom: `migrate` command fails
- Likely cause: DB unavailable, bad credentials, schema conflict
- Fix:
  - verify DB connectivity first
  - run migration again with clean logs
  - if schema drift exists, align DB state before rerunning

## Missing or Invalid Master Keys

- Symptom: startup or key operations fail with master key configuration errors
- Likely cause: invalid format or wrong key length
- Fix:
  - format must be `id:base64key` (or comma-separated list)
  - decoded key must be exactly 32 bytes
  - ensure `ACTIVE_MASTER_KEY_ID` exists in `MASTER_KEYS`

## KMS configuration mismatch

- Symptom: startup fails with errors indicating `KMS_PROVIDER` or `KMS_KEY_URI` is missing
- Likely cause: only one KMS variable is set
- Fix:
  - KMS mode requires both `KMS_PROVIDER` and `KMS_KEY_URI`
  - legacy mode requires both values unset/empty
  - verify `.env` and deployment secret injection order

## Mode mismatch diagnostics

Use these quick checks when startup errors suggest key mode mismatch:

> Command status: verified on 2026-02-20

```bash
# 1) Check selected mode variables
env | grep -E '^(KMS_PROVIDER|KMS_KEY_URI|ACTIVE_MASTER_KEY_ID|MASTER_KEYS)='

# 2) Confirm MASTER_KEYS entry shape
# Legacy mode entries usually look like id:<base64-32-byte-key>
# KMS mode entries should be ciphertext values produced by your KMS provider

# 3) Check startup logs for mode and key load behavior
docker logs <secrets-container-name> 2>&1 | grep -E 'KMS mode enabled|master key decrypted via KMS|master key chain loaded'
```

Expected patterns:

- Legacy mode:
  - no `KMS mode enabled` log line
  - master key chain loads from local config
- KMS mode:
  - `KMS mode enabled provider=<provider>`
  - `master key decrypted via KMS key_id=<id>` for each configured key

## KMS authentication or decryption failures

- Symptom: startup fails while opening KMS keeper or decrypting master keys
- Likely cause: invalid KMS credentials, wrong key URI, missing decrypt permissions, or corrupted ciphertext
- Fix:
  - verify provider credentials are available in runtime environment
  - verify `KMS_KEY_URI` points to the key used to encrypt `MASTER_KEYS`
  - confirm KMS IAM/policy includes decrypt permissions
  - rotate/regenerate master key entries if ciphertext was truncated or malformed
  - use provider setup checks in [KMS setup guide](../operations/kms-setup.md)

## Master key load regression triage (historical v0.5.1 fix)

Historical note:

- This section is retained for mixed-version or rollback investigations involving pre-`v0.5.1` builds.
- For current rollouts, prioritize KMS mode diagnostics and the `v0.7.0` upgrade path.

- Symptom: startup succeeds, but key-dependent operations fail unexpectedly after a recent rollout
- Likely cause: running a pre-`v0.5.1` build where decoded master key buffers could be zeroed too early
- Mixed-version rollout symptom: some requests pass while others fail if old and new images are serving traffic together
- Version fingerprint checks:
  - local binary: `./bin/app --version`
  - pinned image check: `docker run --rm allisson/secrets:v0.7.0 --version`
  - running containers: `docker ps --format 'table {{.Names}}\t{{.Image}}'`
- Fix:
  - upgrade all instances to `v0.7.0` (or at minimum `v0.5.1+`)
  - restart API instances after deploy
  - run key-dependent smoke checks (token issuance, secrets write/read, transit round-trip)
  - review [v0.5.1 release notes](../releases/v0.5.1.md) and
    [v0.5.1 upgrade guide](../releases/v0.5.1-upgrade.md)

## Missing KEK

- Symptom: secret write/transit operations fail after migration
- Likely cause: initial KEK was not created
- Fix:
  - run `create-kek` once after migration
  - verify key creation logs

## Metrics Troubleshooting Matrix

| Symptom | Likely cause | Fix |
| --- | --- | --- |
| `GET /metrics` returns `404` | `METRICS_ENABLED=false` or server restarted with metrics disabled | Set `METRICS_ENABLED=true` and restart server |
| Prometheus scrape target is down | Wrong host/port or network path | Verify target URL and network reachability from Prometheus |
| Metrics present but missing expected prefix | Unexpected namespace value | Confirm `METRICS_NAMESPACE` and update queries/dashboards |
| Dashboards show empty values for paths | Query uses concrete URLs, not route patterns | Query by route pattern labels (for example `/v1/secrets/*path`) |
| Prometheus memory growth or slow queries | High-cardinality query patterns | Aggregate by stable labels and avoid per-request dimensions |

## Tokenization migration verification

- Symptom: tokenization endpoints return `404`/`500` after upgrading to `v0.4.x`
- Likely cause: tokenization migration (`000002_add_tokenization`) not applied or partially applied
- Fix:
  - run `./bin/app migrate` (or Docker `... allisson/secrets:v0.7.0 migrate`)
  - verify migration logs indicate `000002_add_tokenization` applied for your DB
  - confirm initial KEK exists (`create-kek` if missing)
  - re-run smoke flow for tokenization (`tokenize -> detokenize -> validate -> revoke`)

## Rotation completed but server still uses old key context

- Symptom: master key/KEK rotation completed, but runtime behavior suggests old values are still in use
- Likely cause: server process was not restarted after rotation
- Fix:
  - perform rolling restart of all API servers
  - verify `health` endpoint and key-dependent operations after restart
  - apply restart step whenever master keys or KEKs are rotated

## Token issuance fails with valid-looking credentials

- Symptom: `POST /v1/token` still fails
- Likely cause: wrong client secret (one-time output lost), inactive client, deleted client
- Fix:
  - recreate client and securely store the returned one-time secret
  - verify `is_active` is true

## Policy matcher FAQ

Q: Why does `/v1/transit/keys/*/rotate` not match `/v1/transit/keys/payment/extra/rotate`?

- A: Mid-path `*` matches exactly one segment; the extra segment changes route and policy shape.

Q: Why does `prod-*` not work in policy paths?

- A: Partial-segment wildcards are unsupported. Use exact paths, full `*`, trailing `/*`, or mid-path segment `*`.

Q: Why is wildcard `*` risky for normal service clients?

- A: `*` matches every path and can unintentionally grant broad admin-like access. Reserve it for controlled
  break-glass workflows.

## Quick diagnostics checklist

1. `curl http://localhost:8080/health` returns `{"status":"healthy"}`
2. DB is reachable from app runtime context
3. Migrations succeeded
4. Initial KEK exists
5. Token issuance works
6. Caller policy matches endpoint capability and path

## See also

- [Smoke test](smoke-test.md)
- [Docker getting started](docker.md)
- [Local development](local-development.md)
- [Operator runbook index](../operations/runbook-index.md)
- [Production operations](../operations/production.md)
- [Trusted proxy reference](../operations/trusted-proxy-reference.md)
