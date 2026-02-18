# 🔐 Authentication API

> Last updated: 2026-02-18
> Applies to: API v1

All protected endpoints require `Authorization: Bearer <token>`.

## Compatibility

- API surface: `/v1/*`
- Server expectation: Secrets server with token issuance enabled at `POST /v1/token`
- OpenAPI baseline: `docs/openapi.yaml`

## Issue Token

`POST /v1/token`

Request:

```json
{
  "client_id": "018d7e95-1a23-7890-bcde-f1234567890a",
  "client_secret": "sec_1234567890abcdef"
}
```

Response (`201 Created`):

```json
{
  "token": "tok_abcdef1234567890...",
  "expires_at": "2026-02-13T20:13:45Z"
}
```

Curl:

```bash
curl -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{"client_id":"<client-id>","client_secret":"<client-secret>"}'
```

## Use Token

```bash
curl -H "Authorization: Bearer <token>" http://localhost:8080/v1/clients
```

## ⚡ Quick Flow (Under 2 Minutes)

```bash
# 1) Request token
curl -s -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{"client_id":"<client-id>","client_secret":"<client-secret>"}'

# 2) Use returned token in Authorization header
curl -i http://localhost:8080/v1/clients \
  -H "Authorization: Bearer <token>"
```

Expected result: token request returns `201 Created`, authenticated clients request returns `200 OK` (or `403 Forbidden` if token is valid but policy denies list access).

## Common Errors

- `401 Unauthorized`: invalid credentials
- `403 Forbidden`: inactive client
- `422 Unprocessable Entity`: malformed request

## Error Payload Examples

Representative error payloads (exact messages may vary):

`401 Unauthorized`

```json
{
  "error": "unauthorized",
  "message": "invalid client credentials"
}
```

`403 Forbidden`

```json
{
  "error": "forbidden",
  "message": "client is inactive"
}
```

`422 Unprocessable Entity`

```json
{
  "error": "validation_error",
  "message": "invalid request body"
}
```

## Related Examples

- `docs/examples/curl.md`
- `docs/examples/python.md`
- `docs/examples/javascript.md`
- `docs/examples/go.md`
- `docs/api/response-shapes.md`

## Notes

- `Bearer` prefix is case-insensitive (`bearer`, `Bearer`, `BEARER`)
- Tokens are time-limited and should be renewed before expiration

## See also

- [Clients API](clients.md)
- [Policies cookbook](policies.md)
- [Capability matrix](capability-matrix.md)
- [Audit logs API](audit-logs.md)
- [Response shapes](response-shapes.md)
