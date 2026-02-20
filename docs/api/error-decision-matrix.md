# 🚨 API Error Decision Matrix

> Last updated: 2026-02-20
> Applies to: API v1

Use this matrix to triage API failures quickly and choose the next action.

## Decision Matrix

| Status | Meaning | Common causes | First action |
| --- | --- | --- | --- |
| `401 Unauthorized` | Authentication failed | Missing/invalid Bearer token, invalid client credentials, expired token | Re-issue token and verify `Authorization: Bearer <token>` |
| `403 Forbidden` | Authenticated but not allowed | Policy/capability mismatch for request path | Check policy path + required capability mapping |
| `404 Not Found` | Route/resource missing | Wrong endpoint shape, unknown resource ID/key/path | Verify endpoint path shape first, then resource existence |
| `409 Conflict` | Resource state conflict | Duplicate create (for example existing transit key name) | Switch to rotate/update flow or use unique resource name |
| `422 Unprocessable Entity` | Validation failed | Invalid JSON/body/query, bad base64, malformed ciphertext contract | Validate payload and endpoint-specific contract |
| `429 Too Many Requests` | Request throttled | Per-client or per-IP rate limit exceeded | Respect `Retry-After` and retry with backoff + jitter |

## Fast Triage Order

1. Check status code class (`401/403/404/409/422/429`)
2. Validate route shape (to avoid misreading `404` as policy issue)
3. Validate token/authn (`401`) before policy/authz (`403`)
4. Validate payload contract (`422`) using endpoint docs
5. For `429`, apply retry policy and reassess client concurrency

## Fast discriminator (`401` vs `403` vs `429`)

- `401 Unauthorized`: authentication failed before policy check; verify token or client credentials first
- `403 Forbidden`: authentication succeeded, but policy/capability denied requested path
- `429 Too Many Requests`: request hit per-client or per-IP throttling; inspect `Retry-After`

First place to look:

- `401`: token issuance/authentication logs and credential validity
- `403`: policy document, capability mapping, and path matcher behavior
- `429`: rate-limit settings (`RATE_LIMIT_*`, `RATE_LIMIT_TOKEN_*`) and traffic burst patterns

## Capability mismatch quick map (`403`)

- `GET /v1/secrets/*path` requires `decrypt`
- `POST /v1/secrets/*path` requires `encrypt`
- `POST /v1/transit/keys/:name/rotate` requires `rotate`
- `POST /v1/tokenization/detokenize` requires `decrypt`
- `GET /v1/audit-logs` requires `read`

## See also

- [Capability matrix](capability-matrix.md)
- [Policies cookbook](policies.md)
- [API rate limiting](rate-limiting.md)
- [Troubleshooting](../getting-started/troubleshooting.md)
