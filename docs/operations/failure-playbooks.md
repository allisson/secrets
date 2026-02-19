# 🚑 Failure Playbooks

> Last updated: 2026-02-19

Use this page for fast incident triage on common API failures.

## 401 Spike (Unauthorized)

Symptoms:

- sudden increase in `401` across multiple endpoints

Triage steps:

1. Verify token issuance with `POST /v1/token`
2. Confirm callers send `Authorization: Bearer <token>`
3. Check token expiry and client active state
4. Inspect audit logs for broad denied patterns

## 403 Spike (Policy/Capability Mismatch)

Symptoms:

- valid tokens but access denied with `403`

Triage steps:

1. Identify failing endpoint path and required capability
2. Confirm client policy path matching (`*`, exact, trailing `/*`, and mid-path `*` segment rules)
3. Validate capability mapping for endpoint (`read`, `write`, `delete`, `encrypt`, `decrypt`, `rotate`)
4. Re-issue token after policy update

## 409 on Transit Key Create

Symptoms:

- `POST /v1/transit/keys` returns `409 Conflict`

Triage steps:

1. Treat conflict as "key already initialized"
2. Call `POST /v1/transit/keys/:name/rotate` to create a new active version
3. Confirm encrypt/decrypt still work after rotation
4. Update automation to avoid repeated create for existing names

## 404/422 on Tokenization Detokenize

Symptoms:

- `POST /v1/tokenization/detokenize` returns `404 Not Found` or `422 Unprocessable Entity`

Triage steps:

1. Confirm token was produced by `POST /v1/tokenization/keys/:name/tokenize`
2. Confirm request shape uses JSON body `{"token":"..."}` (not URL path token)
3. Check if token is expired (`ttl`) or revoked
4. Validate caller has `decrypt` capability on `/v1/tokenization/detokenize`
5. If expired tokens accumulate, run cleanup routine (`clean-expired-tokens`)

## 409 on Tokenization Key Create

Symptoms:

- `POST /v1/tokenization/keys` returns `409 Conflict`

Triage steps:

1. Treat conflict as "key already initialized"
2. Call `POST /v1/tokenization/keys/:name/rotate` for a new active version
3. Confirm tokenize/detokenize paths remain healthy after rotation
4. Update automation to avoid repeated create for existing names

## Quick Commands

```bash
# Health
curl -s http://localhost:8080/health

# Token check
curl -i -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{"client_id":"<client-id>","client_secret":"<client-secret>"}'

# Audit logs snapshot
curl -s "http://localhost:8080/v1/audit-logs?limit=50&offset=0" \
  -H "Authorization: Bearer <token>"
```

## See also

- [Troubleshooting](../getting-started/troubleshooting.md)
- [Policies cookbook](../api/policies.md)
- [Policy smoke tests](policy-smoke-tests.md)
- [Transit API](../api/transit.md)
- [Tokenization API](../api/tokenization.md)
- [Production operations](production.md)
