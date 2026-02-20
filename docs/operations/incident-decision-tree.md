# 🌲 Incident Decision Tree

> Last updated: 2026-02-20

Use this page to route incidents quickly to the right runbook.

## Start

1. Is `GET /health` failing?
   - Yes -> infrastructure/runtime path: [First 15 Minutes Playbook](first-15-minutes.md)
   - No -> continue
2. Is `GET /ready` failing?
   - Yes -> dependencies/migrations/key-load path: [Troubleshooting](../getting-started/troubleshooting.md)
   - No -> continue
3. Identify dominant status code and route group:
   - `401` -> [Failure playbooks: 401](failure-playbooks.md#401-spike-unauthorized)
   - `403` -> [Failure playbooks: 403](failure-playbooks.md#403-spike-policycapability-mismatch)
   - `429` on `/v1/token` -> [Token throttling runbook](production.md#10-token-endpoint-throttling-runbook)
   - `429` on authenticated routes -> [API rate limiting](../api/rate-limiting.md)
   - `422` -> [API error decision matrix](../api/error-decision-matrix.md)
   - `5xx` -> [First 15 Minutes Playbook](first-15-minutes.md)

## Fast Branches

### `401 Unauthorized`

- Re-issue token via `POST /v1/token`
- Confirm caller sends `Authorization: Bearer <token>`
- Check client active status and secret rotation history

### `403 Forbidden`

- Verify endpoint path shape and required capability
- Verify policy matching semantics (`*`, trailing `/*`, mid-path `*`)
- Re-issue token after policy fix

### `429 Too Many Requests`

- Read `Retry-After` header
- Separate `/v1/token` from authenticated-route throttling
- Validate proxy/source-IP behavior if `/v1/token` is impacted

### `5xx`

- Check database connectivity and pool saturation
- Check migration and key-load startup logs
- Use rollback triggers in production rollout runbook

## Search Aliases

- `retry-after`
- `rate limit exceeded`
- `token endpoint throttling`
- `unauthorized spike`
- `forbidden policy mismatch`

## See also

- [First 15 Minutes Playbook](first-15-minutes.md)
- [Failure playbooks](failure-playbooks.md)
- [Troubleshooting](../getting-started/troubleshooting.md)
- [Operator quick card](operator-quick-card.md)
