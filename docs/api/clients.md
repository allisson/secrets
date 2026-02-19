# 👤 Clients API

> Last updated: 2026-02-19
> Applies to: API v1

Client APIs manage machine identities and policy documents.

## Compatibility

- API surface: `/v1/clients*`
- Server expectation: Secrets server with auth and authorization middleware enabled
- OpenAPI baseline: `docs/openapi.yaml`

## Endpoints

- `POST /v1/clients`
- `GET /v1/clients`
- `GET /v1/clients/:id`
- `PUT /v1/clients/:id`
- `DELETE /v1/clients/:id`

All endpoints require Bearer authentication.

Capability mapping:

- `POST /v1/clients` -> `write`
- `GET /v1/clients` -> `read`
- `GET /v1/clients/:id` -> `read`
- `PUT /v1/clients/:id` -> `write`
- `DELETE /v1/clients/:id` -> `delete`

## Status Code Quick Reference

| Endpoint | Success | Common error statuses |
| --- | --- | --- |
| `POST /v1/clients` | `201` | `401`, `403`, `409`, `422`, `429` |
| `GET /v1/clients` | `200` | `401`, `403`, `422`, `429` |
| `GET /v1/clients/:id` | `200` | `401`, `403`, `404`, `422`, `429` |
| `PUT /v1/clients/:id` | `200` | `401`, `403`, `404`, `409`, `422`, `429` |
| `DELETE /v1/clients/:id` | `204` | `401`, `403`, `404`, `422`, `429` |

## Create Client

```bash
curl -X POST http://localhost:8080/v1/clients \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "payments-api",
    "is_active": true,
    "policies": [
      {"path":"/v1/secrets/*","capabilities":["decrypt"]},
      {"path":"/v1/transit/keys/payment/encrypt","capabilities":["encrypt"]}
    ]
  }'
```

Response includes `id` and one-time `secret`.

Example success status: `201 Created`.

## List Clients

```bash
curl "http://localhost:8080/v1/clients?offset=0&limit=20" \
  -H "Authorization: Bearer <token>"
```

Example success status: `200 OK`.

## Common Errors

- `401 Unauthorized`: missing/invalid bearer token
- `403 Forbidden`: caller lacks required capability for the path
- `404 Not Found`: client ID not found
- `409 Conflict`: unique constraint conflicts
- `422 Unprocessable Entity`: invalid request/query payload
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
  "message": "invalid client request"
}
```

## ⚡ Quick Flow (Under 2 Minutes)

```bash
# 1) Create a client
curl -s -X POST http://localhost:8080/v1/clients \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "quickflow-client",
    "is_active": true,
    "policies": [{"path":"/v1/secrets/*","capabilities":["decrypt"]}]
  }'

# 2) List clients
curl -i "http://localhost:8080/v1/clients?offset=0&limit=10" \
  -H "Authorization: Bearer <admin-token>"
```

Expected result: create returns `201 Created` with one-time `secret`; list returns `200 OK`.

## Related

- 📘 [Policy cookbook](policies.md)
- 🧭 [Wildcard matcher semantics](policies.md#path-matching-behavior)
- 🧪 [Curl examples](../examples/curl.md)
- 🐍 [Python examples](../examples/python.md)
- 🟨 [JavaScript examples](../examples/javascript.md)
- 🐹 [Go examples](../examples/go.md)
- 🧱 [Response shapes](response-shapes.md)

## Use Cases

- 🧩 Create one client per service
- 🔒 Grant only required capabilities per path
- 🔄 Rotate credentials by creating new clients and deactivating old ones

## See also

- [Authentication API](authentication.md)
- [API error decision matrix](error-decision-matrix.md)
- [API rate limiting](rate-limiting.md)
- [Policies cookbook](policies.md)
- [Capability matrix](capability-matrix.md)
- [Audit logs API](audit-logs.md)
- [Response shapes](response-shapes.md)
- [API compatibility policy](versioning-policy.md)
- [Glossary](../concepts/glossary.md)
