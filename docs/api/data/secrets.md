# 📦 Secrets API

> Last updated: 2026-02-19
> Applies to: API v1

Secrets are versioned by path and encrypted with envelope encryption.

## Compatibility

- API surface: `/v1/secrets/*path`
- Server expectation: Secrets server with initialized KEK and active master key configuration
- OpenAPI baseline: `docs/openapi.yaml`

All endpoints require Bearer authentication.

## Endpoints

- `POST /v1/secrets/*path` (create or update)
- `GET /v1/secrets/*path` (read latest)
- `DELETE /v1/secrets/*path` (soft delete latest)

## Status Code Quick Reference

| Endpoint | Success | Common error statuses |
| --- | --- | --- |
| `POST /v1/secrets/*path` | `201` | `401`, `403`, `422`, `429` |
| `GET /v1/secrets/*path` | `200` | `401`, `403`, `404`, `429` |
| `DELETE /v1/secrets/*path` | `204` | `401`, `403`, `404`, `429` |

## Create or Update Secret

```bash
curl -X POST http://localhost:8080/v1/secrets/app/prod/database-password \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"value":"bXktc3VwZXItc2VjcmV0LXBhc3N3b3Jk"}'
```

`value` must be base64-encoded plaintext.

⚠️ Security Warning: base64 is encoding, not encryption. Always use HTTPS/TLS.

Example response (`201 Created`):

```json
{
  "id": "0194f4a5-73fe-7a7d-a3a0-6fbe9b5ef8f3",
  "path": "/app/prod/database-password",
  "version": 3,
  "created_at": "2026-02-14T18:22:00Z"
}
```

## Read Secret

```bash
curl http://localhost:8080/v1/secrets/app/prod/database-password \
  -H "Authorization: Bearer <token>"
```

Example response (`200 OK`):

```json
{
  "id": "0194f4a5-73fe-7a7d-a3a0-6fbe9b5ef8f3",
  "path": "/app/prod/database-password",
  "version": 3,
  "value": "bXktc3VwZXItc2VjcmV0LXBhc3N3b3Jk",
  "created_at": "2026-02-14T18:22:00Z"
}
```

## Delete Secret

```bash
curl -X DELETE http://localhost:8080/v1/secrets/app/prod/database-password \
  -H "Authorization: Bearer <token>"
```

Delete returns `204 No Content`.

## Common Errors

- `401 Unauthorized`: missing/invalid bearer token
- `403 Forbidden`: caller lacks required capability for the path
- `404 Not Found`: secret path not found (or soft-deleted in current context)
- `422 Unprocessable Entity`: invalid request body
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
  "message": "value must be base64-encoded"
}
```

## ⚡ Quick Flow (Under 2 Minutes)

```bash
# 1) Write a secret ("hello-secret" -> base64)
curl -i -X POST http://localhost:8080/v1/secrets/app/prod/quickflow \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"value":"aGVsbG8tc2VjcmV0"}'

# 2) Read it back
curl -i http://localhost:8080/v1/secrets/app/prod/quickflow \
  -H "Authorization: Bearer <token>"
```

Expected result: write returns `201 Created`; read returns `200 OK` with base64 `value`.

## Use Cases

- 🌍 Environment-specific credentials (`/app/dev/*`, `/app/prod/*`)
- 🔐 API keys and tokens for backend services
- 🧱 Zero-trust style central secret retrieval at runtime

## Capability Mapping

- `POST /v1/secrets/*path` -> `encrypt`
- `GET /v1/secrets/*path` -> `decrypt`
- `DELETE /v1/secrets/*path` -> `delete`

Wildcard matcher semantics reference:

- [Policies cookbook / Path matching behavior](../auth/policies.md#path-matching-behavior)

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
- [Policies cookbook](../auth/policies.md)
- [Capability matrix](../fundamentals.md#capability-matrix)
- [Response shapes](../observability/response-shapes.md)
- [API compatibility policy](../fundamentals.md#compatibility-and-versioning-policy)
- [Curl examples](../../examples/curl.md)
- [Glossary](../../concepts/architecture.md#glossary)
