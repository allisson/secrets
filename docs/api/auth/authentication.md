# 🔐 Authentication API

> Last updated: 2026-02-28
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
  "expires_at": "2026-02-27T20:13:45Z"
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
- `423 Locked`: client account locked due to too many failed authentication attempts
- `429 Too Many Requests`: token issuance throttled by IP-based token endpoint limits

Rate limiting note:

- `POST /v1/token` is rate-limited per client IP when `RATE_LIMIT_TOKEN_ENABLED=true`
- Protected endpoints called with issued tokens are rate-limited per authenticated client

## Account Lockout

`POST /v1/token` enforces account lockout to prevent brute-force attacks.

**Behavior:**

1. Each failed authentication attempt (wrong secret) increments the client's `failed_attempts` counter
2. When `failed_attempts` reaches `LOCKOUT_MAX_ATTEMPTS` (default `10`), the client is locked for `LOCKOUT_DURATION_MINUTES` (default `30` minutes)
3. While locked, `POST /v1/token` returns `423 Locked` regardless of the provided secret
4. After the lock window expires, the next request is evaluated normally
5. A successful authentication resets `failed_attempts` to `0` and clears `locked_until`

**Response when locked (`423 Locked`):**

```json
{
  "error": "client_locked",
  "message": "Account is locked due to too many failed authentication attempts"
}
```

**Configuration:**

| Variable | Default | Description |
| --- | --- | --- |
| `LOCKOUT_MAX_ATTEMPTS` | `10` | Failed attempts before lockout |
| `LOCKOUT_DURATION_MINUTES` | `30` | Lock duration in minutes |

**Manual unlock** (for operators):

```text
POST /v1/clients/{id}/unlock  (requires WriteCapability on /v1/clients/{id})
```

See [Configuration reference](../../configuration.md#account-lockout) for details.

## Token `429` Handling Quick Check

Inspect headers and status:

```bash
curl -i -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{"client_id":"<client-id>","client_secret":"<client-secret>"}'
```

Expected when throttled:

- HTTP status `429 Too Many Requests`
- `Retry-After` response header (seconds)

Minimal retry-after extraction example:

```bash
RETRY_AFTER="$(curl -s -D - -o /dev/null -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{"client_id":"<client-id>","client_secret":"<client-secret>"}' \
  | awk -F': ' 'tolower($1)=="retry-after" {print $2}' | tr -d '\r')"

echo "Retry after: ${RETRY_AFTER}s"
```

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
  "message": "invalid request body"
}
```

## Related Examples

- `docs/examples/curl.md`
- `docs/examples/python.md`
- `docs/examples/javascript.md`
- `docs/examples/go.md`
- `docs/api/observability/response-shapes.md`

## Notes

- `Bearer` prefix is case-insensitive (`bearer`, `Bearer`, `BEARER`)
- Tokens are time-limited and should be renewed before expiration
- Client secrets are hashed using Argon2id (see [ADR 0010: Argon2id for Client Secret Hashing](../../adr/0010-argon2id-for-client-secret-hashing.md))
- For wildcard path matcher semantics used by authorization, see
  [Policies cookbook / Path matching behavior](policies.md#path-matching-behavior)

## See also

- [Clients API](clients.md)
- [API error decision matrix](../fundamentals.md#error-decision-matrix)
- [API rate limiting](../fundamentals.md#rate-limiting)
- [Policies cookbook](policies.md)
- [Capability matrix](../fundamentals.md#capability-matrix)
- [Audit logs API](../observability/audit-logs.md)
- [Response shapes](../observability/response-shapes.md)
