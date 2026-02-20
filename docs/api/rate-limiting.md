# 🚦 API Rate Limiting

> Last updated: 2026-02-20
> Applies to: API v1

Secrets enforces two rate-limiting scopes:

- Per-client limits for authenticated API routes (`RATE_LIMIT_*`)
- Per-IP limits for unauthenticated token issuance (`RATE_LIMIT_TOKEN_*`)

## Scope

Rate limiting scope matrix:

| Route group/endpoint | Rate limited | Notes |
| --- | --- | --- |
| `/v1/clients/*` | Yes | Requires Bearer auth |
| `/v1/audit-logs` | Yes | Requires Bearer auth |
| `/v1/secrets/*` | Yes | Requires Bearer auth |
| `/v1/transit/*` | Yes | Requires Bearer auth |
| `/v1/tokenization/*` | Yes | Requires Bearer auth |
| `POST /v1/token` | Yes | Unauthenticated endpoint, rate-limited per client IP |
| `GET /health` | No | Liveness checks |
| `GET /ready` | No | Readiness checks |
| `GET /metrics` | No | Prometheus scraping |

## Defaults

```dotenv
# Authenticated endpoints (per client)
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_SEC=10.0
RATE_LIMIT_BURST=20

# Token endpoint (per IP)
RATE_LIMIT_TOKEN_ENABLED=true
RATE_LIMIT_TOKEN_REQUESTS_PER_SEC=5.0
RATE_LIMIT_TOKEN_BURST=10
```

## Response behavior

When a request exceeds the allowed rate, the API returns:

- Status: `429 Too Many Requests`
- Header: `Retry-After: <seconds>`
- Body:

```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests. Please retry after the specified delay."
}
```

Token endpoint (`POST /v1/token`) uses the same status/header contract and returns an endpoint-specific
message indicating too many token requests from the caller IP.

## Client retry guidance

- Respect `Retry-After` before retrying
- Use exponential backoff with jitter
- Avoid synchronized retries across many workers
- Reduce per-client burst and concurrency where possible
- For token issuance, review shared NAT/proxy behavior and tune `RATE_LIMIT_TOKEN_*` if needed

## Distinguishing `403` vs `429`

- `403 Forbidden`: policy/capability denies access
- `429 Too Many Requests`: request was throttled by per-client or per-IP rate limits

## See also

- [Environment variables](../configuration/environment-variables.md)
- [API error decision matrix](error-decision-matrix.md)
- [Response shapes](response-shapes.md)
- [Troubleshooting](../getting-started/troubleshooting.md)
