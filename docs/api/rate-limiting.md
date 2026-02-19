# 🚦 API Rate Limiting

> Last updated: 2026-02-19
> Applies to: API v1

Secrets enforces per-client rate limiting for authenticated API routes when
`RATE_LIMIT_ENABLED=true` (default).

## Scope

Rate limiting scope matrix:

| Route group/endpoint | Rate limited | Notes |
| --- | --- | --- |
| `/v1/clients/*` | Yes | Requires Bearer auth |
| `/v1/audit-logs` | Yes | Requires Bearer auth |
| `/v1/secrets/*` | Yes | Requires Bearer auth |
| `/v1/transit/*` | Yes | Requires Bearer auth |
| `/v1/tokenization/*` | Yes | Requires Bearer auth |
| `POST /v1/token` | No | Token issuance route |
| `GET /health` | No | Liveness checks |
| `GET /ready` | No | Readiness checks |
| `GET /metrics` | No | Prometheus scraping |

## Defaults

```dotenv
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_SEC=10.0
RATE_LIMIT_BURST=20
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

## Client retry guidance

- Respect `Retry-After` before retrying
- Use exponential backoff with jitter
- Avoid synchronized retries across many workers
- Reduce per-client burst and concurrency where possible

## Distinguishing `403` vs `429`

- `403 Forbidden`: policy/capability denies access
- `429 Too Many Requests`: request was authenticated/authorized but throttled

## See also

- [Environment variables](../configuration/environment-variables.md)
- [API error decision matrix](error-decision-matrix.md)
- [Response shapes](response-shapes.md)
- [Troubleshooting](../getting-started/troubleshooting.md)
