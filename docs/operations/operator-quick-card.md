# ⚡ Operator Quick Card

> Last updated: 2026-02-20

Use this page during rollout and incidents when you need a fast, minimal checklist.

## Rollout Preflight (5-minute check)

1. Confirm target version and image tag match release plan
2. Confirm DB connectivity and migration window
3. Confirm key mode settings (`KMS_PROVIDER` + `KMS_KEY_URI` or legacy mode)
4. Confirm token and route rate-limit settings are intentional
5. Confirm rollback owner and communication channel

Primary references:

- [Production rollout golden path](production-rollout.md)
- [Release compatibility matrix](../releases/compatibility-matrix.md)
- [v0.7.0 upgrade guide](../releases/v0.7.0-upgrade.md)

## Baseline Verification (before traffic cutover)

1. `GET /health` returns `200`
2. `GET /ready` returns `200`
3. `POST /v1/token` returns `201`
4. Secrets write/read passes
5. Transit encrypt/decrypt passes

Reference:

- [Smoke test guide](../getting-started/smoke-test.md)

## Fast Status Triage (`401` / `403` / `429`)

1. `401`: re-check credentials/token issuance path
2. `403`: verify policy path and capability mapping
3. `429`: check `Retry-After`, then decide per-client vs token-IP tuning path

References:

- [API error decision matrix](../api/error-decision-matrix.md)
- [API rate limiting](../api/rate-limiting.md)
- [Monitoring](monitoring.md)

## Token Endpoint `429` Quick Path

1. Confirm `429` concentrated on `POST /v1/token`
2. Verify shared NAT/proxy egress is not collapsing many clients to one IP
3. Validate trusted proxy and forwarded header behavior
4. Apply temporary `RATE_LIMIT_TOKEN_*` tuning only if traffic is legitimate
5. Revert temporary tuning after stability

References:

- [Production token throttling runbook](production.md#10-token-endpoint-throttling-runbook)
- [Trusted proxy reference](trusted-proxy-reference.md)
- [Troubleshooting](../getting-started/troubleshooting.md)

## Rollback Triggers

- Sustained elevated `5xx`
- Widespread token/auth failures
- Unexpected data-integrity behavior
- Failed verification gates after rollout

Reference:

- [Rollback procedure](production-rollout.md#rollback-procedure-binaryimage)

## Incident Notes Minimum

Capture these before closing:

- timeline (detection -> mitigation -> recovery)
- affected routes/clients
- config changes applied (`RATE_LIMIT_*`, `RATE_LIMIT_TOKEN_*`, policy updates)
- final mitigation and follow-up owner

## See also

- [Operator runbook index](runbook-index.md)
- [Production deployment guide](production.md)
- [Failure playbooks](failure-playbooks.md)
