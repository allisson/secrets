# 📜 Audit Logs API

> Last updated: 2026-02-14
> Applies to: API v1

Audit logs capture capability checks and access attempts for monitoring and compliance.

## Compatibility

- API surface: `/v1/audit-logs`
- Server expectation: Secrets server with audit logging middleware active
- OpenAPI baseline: `docs/openapi.yaml`

Authentication: required (Bearer token).
Authorization: `read` capability for `/v1/audit-logs`.

## Endpoint

- `GET /v1/audit-logs`

## Status Code Quick Reference

| Endpoint | Success | Common error statuses |
| --- | --- | --- |
| `GET /v1/audit-logs` | `200` | `401`, `403`, `422` |

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
      "created_at": "2026-02-14T18:35:12Z"
    }
  ]
}
```

## Returned Fields

- `id`
- `request_id`
- `client_id`
- `capability`
- `path`
- `metadata.allowed`
- `metadata.ip`
- `metadata.user_agent`
- `created_at`

## Practical Checks

- 🚨 Detect repeated denied actions (`metadata.allowed == false`)
- 🌐 Spot unusual source IP changes per client
- 🧭 Correlate a request with app logs via `request_id`

## Common Errors

- `401 Unauthorized`: missing/invalid bearer token
- `403 Forbidden`: caller lacks `read` capability for `/v1/audit-logs`
- `422 Unprocessable Entity`: invalid query values (offset/limit/timestamps)

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
- `docs/api/response-shapes.md`

## See also

- [Authentication API](authentication.md)
- [Clients API](clients.md)
- [Policies cookbook](policies.md)
- [Response shapes](response-shapes.md)
- [API compatibility policy](versioning-policy.md)
- [Glossary](../concepts/glossary.md)
