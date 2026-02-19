# 🧰 Troubleshooting

> Last updated: 2026-02-19

Use this guide for common setup and runtime errors.

## 🧭 Decision Tree

Use this quick route before diving into detailed sections:

1. `curl http://localhost:8080/health` fails -> go to `Database connection failure` and `Migration failure`
2. Token endpoint (`POST /v1/token`) returns `401`/`403` -> go to `401 Unauthorized` or `Token issuance fails with valid-looking credentials`
3. API requests return `403` with valid token -> go to `403 Forbidden` (policy/capability mismatch)
4. API requests return `422` -> go to `422 Unprocessable Entity` (payload/query format)
5. After rotating keys behavior is stale -> go to `Rotation completed but server still uses old key context`
6. Startup fails with key config errors -> go to `Missing or Invalid Master Keys`
7. Monitoring data is missing -> go to `Metrics Troubleshooting Matrix`
8. Tokenization endpoints fail after upgrade -> go to `Tokenization migration verification`

## 📑 Table of Contents

- [401 Unauthorized](#401-unauthorized)
- [403 Forbidden](#403-forbidden)
- [409 Conflict](#409-conflict)
- [422 Unprocessable Entity](#422-unprocessable-entity)
- [Database connection failure](#database-connection-failure)
- [Migration failure](#migration-failure)
- [Missing or Invalid Master Keys](#missing-or-invalid-master-keys)
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
  - run `./bin/app migrate` (or Docker `... allisson/secrets:v0.4.1 migrate`)
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
