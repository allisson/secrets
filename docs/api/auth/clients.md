# 👤 Clients API

> Last updated: 2026-02-28
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
- `POST /v1/clients/:id/unlock`

All endpoints require Bearer authentication.

Capability mapping:

- `POST /v1/clients` -> `write`
- `GET /v1/clients` -> `read`
- `GET /v1/clients/:id` -> `read`
- `PUT /v1/clients/:id` -> `write`
- `DELETE /v1/clients/:id` -> `delete`
- `POST /v1/clients/:id/unlock` -> `write`

## Status Code Quick Reference

| Endpoint | Success | Common error statuses |
| --- | --- | --- |
| `POST /v1/clients` | `201` | `401`, `403`, `409`, `422`, `429` |
| `GET /v1/clients` | `200` | `401`, `403`, `422`, `429` |
| `GET /v1/clients/:id` | `200` | `401`, `403`, `404`, `422`, `429` |
| `PUT /v1/clients/:id` | `200` | `401`, `403`, `404`, `409`, `422`, `429` |
| `DELETE /v1/clients/:id` | `204` | `401`, `403`, `404`, `422`, `429` |
| `POST /v1/clients/:id/unlock` | `200` | `401`, `403`, `404`, `422`, `429` |

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

Example response (`201 Created`):

```json
{
  "id": "0194f4a6-7ec7-78e6-9fe7-5ca35fef48db",
  "secret": "s3cr3t-v4lu3-th4t-sh0uld-b3-s4v3d"
}
```

Example success status: `201 Created`.

## List Clients

```bash
curl "http://localhost:8080/v1/clients?offset=0&limit=20" \
  -H "Authorization: Bearer <token>"
```

Example response (`200 OK`):

```json
{
  "data": [
    {
      "id": "0194f4a6-7ec7-78e6-9fe7-5ca35fef48db",
      "name": "payments-api",
      "is_active": true,
      "policies": [
        {
          "path": "/v1/secrets/*",
          "capabilities": ["decrypt"]
        }
      ],
      "created_at": "2026-02-27T18:35:12Z"
    }
  ]
}
```

Example success status: `200 OK`.

## Get Client

```bash
curl http://localhost:8080/v1/clients/0194f4a6-7ec7-78e6-9fe7-5ca35fef48db \
  -H "Authorization: Bearer <token>"
```

Example response (`200 OK`):

```json
{
  "id": "0194f4a6-7ec7-78e6-9fe7-5ca35fef48db",
  "name": "payments-api",
  "is_active": true,
  "policies": [
    {
      "path": "/v1/secrets/*",
      "capabilities": ["decrypt"]
    }
  ],
  "created_at": "2026-02-27T18:35:12Z"
}
```

## Unlock Client

Clears the lockout state for a client that was locked due to too many failed authentication attempts. See [Account Lockout](authentication.md#account-lockout) for lockout behavior.

```bash
curl -X POST http://localhost:8080/v1/clients/<id>/unlock \
  -H "Authorization: Bearer <admin-token>"
```

Returns `200 OK` with the updated client object.

Example response (`200 OK`):

```json
{
  "id": "0194f4a6-7ec7-78e6-9fe7-5ca35fef48db",
  "name": "payments-api",
  "is_active": true,
  "policies": [
    {
      "path": "/v1/secrets/*",
      "capabilities": ["decrypt"]
    }
  ],
  "created_at": "2026-02-27T18:35:12Z"
}
```

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
  "message": "Authentication is required"
}
```

`403 Forbidden`

```json
{
  "error": "forbidden",
  "message": "You don't have permission to access this resource"
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
- 🧪 [Curl examples](../../examples/curl.md)
- 🐍 [Python examples](../../examples/python.md)
- 🟨 [JavaScript examples](../../examples/javascript.md)
- 🐹 [Go examples](../../examples/go.md)
- 🧱 [Response shapes](../observability/response-shapes.md)

## Use Cases

- 🧩 Create one client per service
- 🔒 Grant only required capabilities per path
- 🔄 Rotate credentials by creating new clients and deactivating old ones

## See also

- [Authentication API](authentication.md)
- [API error decision matrix](../fundamentals.md#error-decision-matrix)
- [API rate limiting](../fundamentals.md#rate-limiting)
- [Policies cookbook](policies.md)
- [Capability matrix](../fundamentals.md#capability-matrix)
- [Audit logs API](../observability/audit-logs.md)
- [Response shapes](../observability/response-shapes.md)
- [API compatibility policy](../fundamentals.md#compatibility-and-versioning-policy)
- [Glossary](../../concepts/architecture.md#glossary)
