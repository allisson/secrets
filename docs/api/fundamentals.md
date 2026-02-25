# 🧩 API Fundamentals

> Last updated: 2026-02-25
> Applies to: API v1

This page consolidates foundational API concepts for quick reference: error triage, capability mapping, rate limiting, and versioning policy.

## Table of Contents

- [Error Decision Matrix](#error-decision-matrix)
- [Capability Matrix](#capability-matrix)
- [Rate Limiting](#rate-limiting)
- [Compatibility and Versioning Policy](#compatibility-and-versioning-policy)

---

## Error Decision Matrix

Use this matrix to triage API failures quickly and choose the next action.

### Decision Matrix

| Status | Meaning | Common causes | First action |
| --- | --- | --- | --- |
| `401 Unauthorized` | Authentication failed | Missing/invalid Bearer token, invalid client credentials, expired token | Re-issue token and verify `Authorization: Bearer <token>` |
| `403 Forbidden` | Authenticated but not allowed | Policy/capability mismatch for request path | Check policy path + required capability mapping |
| `404 Not Found` | Route/resource missing | Wrong endpoint shape, unknown resource ID/key/path | Verify endpoint path shape first, then resource existence |
| `409 Conflict` | Resource state conflict | Duplicate create (for example existing transit key name) | Switch to rotate/update flow or use unique resource name |
| `422 Unprocessable Entity` | Validation failed | Invalid JSON/body/query, bad base64, malformed ciphertext contract | Validate payload and endpoint-specific contract |
| `429 Too Many Requests` | Request throttled | Per-client or per-IP rate limit exceeded | Respect `Retry-After` and retry with backoff + jitter |

### Fast Triage Order

1. Check status code class (`401/403/404/409/422/429`)
2. Validate route shape (to avoid misreading `404` as policy issue)
3. Validate token/authn (`401`) before policy/authz (`403`)
4. Validate payload contract (`422`) using endpoint docs
5. For `429`, apply retry policy and reassess client concurrency

### Fast discriminator (`401` vs `403` vs `429`)

- `401 Unauthorized`: authentication failed before policy check; verify token or client credentials first
- `403 Forbidden`: authentication succeeded, but policy/capability denied requested path
- `429 Too Many Requests`: request hit per-client or per-IP throttling; inspect `Retry-After`

First place to look:

- `401`: token issuance/authentication logs and credential validity
- `403`: policy document, capability mapping, and path matcher behavior
- `429`: rate-limit settings (`RATE_LIMIT_*`, `RATE_LIMIT_TOKEN_*`) and traffic burst patterns

### Capability mismatch quick map (`403`)

- `GET /v1/secrets/*path` requires `decrypt`
- `POST /v1/secrets/*path` requires `encrypt`
- `POST /v1/transit/keys/:name/rotate` requires `rotate`
- `POST /v1/tokenization/detokenize` requires `decrypt`
- `GET /v1/audit-logs` requires `read`

---

## Capability Matrix

This section is the canonical capability-to-endpoint reference used by API docs and policy templates.

### Capability Definitions

- `read`: list or inspect metadata/state without decrypting payload values
- `write`: create or update non-cryptographic resources and key definitions
- `delete`: delete resources or revoke token lifecycle entries
- `encrypt`: create encrypted outputs (secrets writes, transit encrypt, tokenization tokenize)
- `decrypt`: resolve encrypted/tokenized values back to plaintext
- `rotate`: create new key versions

### Endpoint Matrix

| Endpoint | Required capability |
| --- | --- |
| `POST /v1/clients` | `write` |
| `GET /v1/clients` | `read` |
| `GET /v1/clients/:id` | `read` |
| `PUT /v1/clients/:id` | `write` |
| `DELETE /v1/clients/:id` | `delete` |
| `POST /v1/clients/:id/unlock` | `write` |
| `GET /v1/audit-logs` | `read` |
| `POST /v1/secrets/*path` | `encrypt` |
| `GET /v1/secrets/*path` | `decrypt` |
| `DELETE /v1/secrets/*path` | `delete` |
| `POST /v1/transit/keys` | `write` |
| `POST /v1/transit/keys/:name/rotate` | `rotate` |
| `DELETE /v1/transit/keys/:id` | `delete` |
| `POST /v1/transit/keys/:name/encrypt` | `encrypt` |
| `POST /v1/transit/keys/:name/decrypt` | `decrypt` |
| `POST /v1/tokenization/keys` | `write` |
| `POST /v1/tokenization/keys/:name/rotate` | `rotate` |
| `DELETE /v1/tokenization/keys/:id` | `delete` |
| `POST /v1/tokenization/keys/:name/tokenize` | `encrypt` |
| `POST /v1/tokenization/detokenize` | `decrypt` |
| `POST /v1/tokenization/validate` | `read` |
| `POST /v1/tokenization/revoke` | `delete` |

### Policy Authoring Notes

Policy matcher quick reference:

| Pattern type | Example | Matching behavior |
| --- | --- | --- |
| Exact | `/v1/audit-logs` | Only that exact path |
| Full wildcard | `*` | Any request path |
| Trailing wildcard | `/v1/secrets/*` | Prefix + nested paths |
| Mid-path wildcard | `/v1/transit/keys/*/rotate` | `*` matches one segment |

For complete matcher semantics and unsupported forms, see [Policies cookbook](auth/policies.md#path-matching-behavior).

See [ADR 0003: Capability-Based Authorization Model](../adr/0003-capability-based-authorization-model.md) for the architectural rationale behind this design.

- Use path scope as narrowly as possible (service + environment prefixes).
- Avoid wildcard `*` except temporary break-glass workflows.
- Keep encrypt and decrypt separated across clients when operationally possible.
- For tokenization lifecycle endpoints, token value is passed in JSON body; policy path is endpoint path.

---

## Rate Limiting

Secrets enforces two rate-limiting scopes:

- Per-client limits for authenticated API routes (`RATE_LIMIT_*`)
- Per-IP limits for unauthenticated token issuance (`RATE_LIMIT_TOKEN_*`)

See [ADR 0006: Dual-Scope Rate Limiting Strategy](../adr/0006-dual-scope-rate-limiting-strategy.md) for the architectural rationale behind this design.

### Scope

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

### Defaults

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

### Response behavior

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

### Client retry guidance

- Respect `Retry-After` before retrying
- Use exponential backoff with jitter
- Avoid synchronized retries across many workers
- Reduce per-client burst and concurrency where possible
- For token issuance, review shared NAT/proxy behavior and tune `RATE_LIMIT_TOKEN_*` if needed

### Distinguishing `403` vs `429`

- `403 Forbidden`: policy/capability denies access
- `429 Too Many Requests`: request was throttled by per-client or per-IP rate limits

---

## Compatibility and Versioning Policy

This section defines compatibility expectations for HTTP API changes.

See [ADR 0007: Path-Based API Versioning](../adr/0007-path-based-api-versioning.md) for the architectural rationale behind this design.

### Compatibility Contract

- Current public baseline is API v1 (`/v1/*`)
- Existing endpoint paths and JSON field names are treated as stable unless explicitly deprecated
- OpenAPI source of truth: `docs/openapi.yaml`

### OpenAPI Coverage

- `docs/openapi.yaml` is a baseline subset focused on high-traffic/common integration flows
- `docs/openapi.yaml` includes tokenization endpoint coverage in the current release
- `docs/openapi.yaml` includes `429 Too Many Requests` response modeling for protected routes
- Endpoint pages in `docs/api/*.md` define full public behavior for covered operations
- Endpoints may exist in runtime before they are expanded in OpenAPI detail

### App Version vs API Version

- Application release is pre-1.0 software and may evolve quickly
- API v1 path contract (`/v1/*`) remains the compatibility baseline for consumers
- Breaking API behavior changes require explicit documentation and migration notes

### Breaking Changes

Treat these as breaking:

- changing endpoint paths or required path parameters
- removing response fields or changing field meaning/type
- changing required request fields or accepted formats
- changing status code semantics for successful behavior

Required process for breaking changes:

1. Update `docs/openapi.yaml`
2. Update affected API docs and examples
3. Add migration notes in `docs/operations/troubleshooting/index.md` or relevant runbook
4. Add explicit entry in `releases/RELEASES.md`

### Non-Breaking Changes

Usually non-breaking:

- adding optional request/response fields
- adding new endpoints under `/v1/*`
- clarifying documentation text and examples
- adding additional error examples without changing behavior

### Telemetry Change Examples

Breaking telemetry examples:

- renaming a published metric name (for example `secrets_http_requests_total`)
- renaming/removing metric labels used by dashboards or alerts

Non-breaking telemetry examples:

- adding a new metric family
- adding new label values for existing labels
- adding new dashboard examples without changing metric contracts

### Deprecation Guidance

- Mark deprecated behavior clearly in endpoint docs
- Provide replacement behavior and example migration path
- Keep deprecated behavior available long enough for operational rollout

---

## See also

- [Authentication API](auth/authentication.md)
- [Clients API](auth/clients.md)
- [Policies cookbook](auth/policies.md)
- [Secrets API](data/secrets.md)
- [Transit API](data/transit.md)
- [Tokenization API](data/tokenization.md)
- [Audit Logs API](observability/audit-logs.md)
- [Response shapes](observability/response-shapes.md)
- [Environment variables](../configuration.md)
- [Troubleshooting](../operations/troubleshooting/index.md)
- [Contributing guide](../contributing.md)
