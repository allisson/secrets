# 🚑 Failure Playbooks

> Last updated: 2026-02-14

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
2. Confirm client policy path matching (`*`, exact, `/*` prefix)
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
- [Transit API](../api/transit.md)
- [Production operations](production.md)
