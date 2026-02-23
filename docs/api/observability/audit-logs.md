# 📜 Audit Logs API

> Last updated: 2026-02-23
> Applies to: API v1

Audit logs capture capability checks and access attempts for monitoring and compliance.

## Compatibility

- API surface: `/v1/audit-logs`
- Server expectation: Secrets server with audit logging middleware active
- OpenAPI baseline: `docs/openapi.yaml`

Authentication: required (Bearer token).
Authorization: `read` capability for `/v1/audit-logs`.

Capability reference:

- Canonical mapping source: [Capability matrix](../fundamentals.md#capability-matrix)

## Endpoint

- `GET /v1/audit-logs`

## Status Code Quick Reference

| Endpoint | Success | Common error statuses |
| --- | --- | --- |
| `GET /v1/audit-logs` | `200` | `401`, `403`, `422`, `429` |

Query parameters:

- `offset` (default `0`)
- `limit` (default `50`, max `100`)
- `created_at_from` (RFC3339)
- `created_at_to` (RFC3339)

## Example

```bash
curl "http://localhost:8080/v1/audit-logs?created_at_from=2026-02-13T00:00:00Z&limit=20" \
  -H "Authorization: Bearer <token>"
```

Example response (`200 OK`):

```json
{
  "audit_logs": [
    {
      "id": "0194f4a7-8fbe-7e3b-b7b2-72f3ac8f6ed0",
      "request_id": "0194f4a7-8fbc-73c1-a114-88c1d8682cb7",
      "client_id": "0194f4a6-7ec7-78e6-9fe7-5ca35fef48db",
      "capability": "decrypt",
      "path": "/v1/secrets/app/prod/database-password",
      "metadata": {
        "allowed": true,
        "ip": "192.168.1.10",
        "user_agent": "curl/8.7.1"
      },
      "signature": "YWJjZGVmZ2hpamtsbW5vcHFyc3R1dnd4eXoxMjM0NTY3ODkwYWJjZGVmZ2hp",
      "kek_id": "0194f4a6-7ec7-78e6-9fe7-5ca35fef48db",
      "is_signed": true,
      "created_at": "2026-02-14T18:35:12Z"
    }
  ]
}
```

**Note:** Audit logs created before v0.9.0 will have `is_signed=false`, `signature=null`, and `kek_id=null` (legacy unsigned logs).

## Returned Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | UUID | Audit log unique identifier (UUIDv7) |
| `request_id` | UUID | Request unique identifier |
| `client_id` | UUID | Client that performed the operation |
| `capability` | string | Capability required for operation (e.g., `read`, `write`, `decrypt`) |
| `path` | string | Resource path accessed |
| `metadata` | object | Operation metadata (allowed, ip, user_agent) |
| `metadata.allowed` | boolean | Whether access was allowed by policy |
| `metadata.ip` | string | Client IP address |
| `metadata.user_agent` | string | Client user agent |
| `signature` | string | Base64-encoded HMAC-SHA256 signature (32 bytes, v0.9.0+) |
| `kek_id` | UUID | KEK used for signing (null for legacy logs, v0.9.0+) |
| `is_signed` | boolean | True if cryptographically signed (v0.9.0+) |
| `created_at` | string | ISO 8601 timestamp (UTC) |

### Signature Fields (v0.9.0+)

**Cryptographic Integrity:**

- All audit logs created in v0.9.0+ are automatically signed with HMAC-SHA256 for tamper detection
- Signature derived from KEK using HKDF-SHA256 key derivation (separates encryption and signing usage)

**Field Details:**

- `signature`: HMAC-SHA256 signature for tamper detection (null for legacy logs created before v0.9.0)
- `kek_id`: References KEK used for signing (null for legacy logs)
- `is_signed`: `true` for signed logs, `false` for legacy unsigned logs

**Legacy vs Signed Logs:**

- Logs created before v0.9.0 have `is_signed=false` (legacy unsigned)
- Logs created in v0.9.0+ have `is_signed=true` with signature and KEK ID
- Use `verify-audit-logs` CLI command to verify cryptographic integrity

**Verification:**

```bash
# Verify audit log integrity for a date range
./bin/app verify-audit-logs --start-date "2026-02-20" --end-date "2026-02-20"

# JSON output for automation
./bin/app verify-audit-logs --start-date "2026-02-20" --end-date "2026-02-20" --format json
```

See [CLI commands](../../cli-commands.md#verify-audit-logs) for verification details.

## Practical Checks

- 🚨 Detect repeated denied actions (`metadata.allowed == false`)
- 🌐 Spot unusual source IP changes per client
- 🧭 Correlate a request with app logs via `request_id`

## Retention and Cleanup

- Audit log cleanup is an operator workflow via CLI, not an HTTP delete endpoint
- Use `clean-audit-logs` to delete old records by retention days
- Start with `--dry-run` to preview affected rows before deletion

Example:

```bash
./bin/app clean-audit-logs --days 90 --dry-run --format json
```

## Common Errors

- `401 Unauthorized`: missing/invalid bearer token
- `403 Forbidden`: caller lacks `read` capability for `/v1/audit-logs`
- `422 Unprocessable Entity`: invalid query values (offset/limit/timestamps)
- `429 Too Many Requests`: per-client rate limit exceeded

## Error Payload Examples

Representative error payloads (exact messages may vary):

`401 Unauthorized`

```json
{
  "error": "unauthorized",
  "message": "missing or invalid bearer token"
}
```

`403 Forbidden`

```json
{
  "error": "forbidden",
  "message": "insufficient capability for path"
}
```

`422 Unprocessable Entity`

```json
{
  "error": "validation_error",
  "message": "invalid query parameter values"
}
```

## ⚡ Quick Flow (Under 2 Minutes)

```bash
# 1) Query recent audit logs
curl -i "http://localhost:8080/v1/audit-logs?limit=10&offset=0" \
  -H "Authorization: Bearer <token>"

# 2) Query a time window
curl -i "http://localhost:8080/v1/audit-logs?created_at_from=2026-02-14T00:00:00Z&limit=10" \
  -H "Authorization: Bearer <token>"
```

Expected result: both requests return `200 OK` with `audit_logs` array.

## Quick Analysis Examples

Denied requests:

```bash
curl -s "http://localhost:8080/v1/audit-logs?limit=100" \
  -H "Authorization: Bearer <token>" | jq '.audit_logs[] | select(.metadata.allowed == false)'
```

Operations by capability:

```bash
curl -s "http://localhost:8080/v1/audit-logs?limit=100" \
  -H "Authorization: Bearer <token>" | jq '[.audit_logs[].capability] | group_by(.) | map({capability: .[0], count: length})'
```

## Related Examples

- `docs/examples/curl.md`
- `docs/examples/python.md`
- `docs/examples/javascript.md`
- `docs/examples/go.md`
- `docs/api/observability/response-shapes.md`

## See also

- [Authentication API](../auth/authentication.md)
- [API error decision matrix](../fundamentals.md#error-decision-matrix)
- [API rate limiting](../fundamentals.md#rate-limiting)
- [Clients API](../auth/clients.md)
- [Policies cookbook](../auth/policies.md)
- [Route shape vs policy shape](../auth/policies.md#route-shape-vs-policy-shape)
- [Policy review checklist before deploy](../auth/policies.md#policy-review-checklist-before-deploy)
- [Capability matrix](../fundamentals.md#capability-matrix)
- [Response shapes](response-shapes.md)
- [API compatibility policy](../fundamentals.md#compatibility-and-versioning-policy)
- [Glossary](../../concepts/architecture.md#glossary)
