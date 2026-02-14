# 🧰 Troubleshooting

> Last updated: 2026-02-14

Use this guide for common setup and runtime errors.

## 🧭 Decision Tree

Use this quick route before diving into detailed sections:

1. `curl http://localhost:8080/health` fails -> go to `Database connection failure` and `Migration failure`
2. Token endpoint (`POST /v1/token`) returns `401`/`403` -> go to `401 Unauthorized` or `Token issuance fails with valid-looking credentials`
3. API requests return `403` with valid token -> go to `403 Forbidden` (policy/capability mismatch)
4. API requests return `422` -> go to `422 Unprocessable Entity` (payload/query format)
5. After rotating keys behavior is stale -> go to `Rotation completed but server still uses old key context`
6. Startup fails with key config errors -> go to `Missing or Invalid Master Keys`

## 📑 Table of Contents

- [401 Unauthorized](#401-unauthorized)
- [403 Forbidden](#403-forbidden)
- [422 Unprocessable Entity](#422-unprocessable-entity)
- [Database connection failure](#database-connection-failure)
- [Migration failure](#migration-failure)
- [Missing or Invalid Master Keys](#missing-or-invalid-master-keys)
- [Missing KEK](#missing-kek)
- [Rotation completed but server still uses old key context](#rotation-completed-but-server-still-uses-old-key-context)
- [Token issuance fails with valid-looking credentials](#token-issuance-fails-with-valid-looking-credentials)
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
  - verify path pattern (`*`, exact path, or prefix with `/*`)
  - update client policy and retry

## 422 Unprocessable Entity

- Symptom: request rejected with `422`
- Likely cause: malformed JSON, invalid query params, missing required fields
- Fix:
  - validate JSON body and required fields
  - for secrets/transit endpoints, send base64 values where required
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
- [Production operations](../operations/production.md)
