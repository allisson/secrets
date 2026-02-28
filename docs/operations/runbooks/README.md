# 🧭 Operator Runbook Index

> Last updated: 2026-02-28

Use this page as the single entry point for rollout, validation, and incident runbooks.

## Release and Rollout

- [Release notes](../../releases/RELEASES.md)
- [Production rollout golden path](../deployment/production-rollout.md)
- [Production deployment guide](../deployment/docker-hardened.md)
- [KMS setup guide](../kms/setup.md)

## Authorization Policy Validation

- [Policies cookbook](../../api/auth/policies.md)
- [Path matching behavior](../../api/auth/policies.md#path-matching-behavior)
- [Route shape vs policy shape](../../api/auth/policies.md#route-shape-vs-policy-shape)
- [Policy review checklist before deploy](../../api/auth/policies.md#policy-review-checklist-before-deploy)
- [Policy smoke tests](../runbooks/policy-smoke-tests.md)

## API and Access Verification

- [Capability matrix](../../api/fundamentals.md#capability-matrix)
- [API error decision matrix](../../api/fundamentals.md#error-decision-matrix)
- [Authentication API](../../api/auth/authentication.md)
- [Audit logs API](../../api/observability/audit-logs.md)

## Incident and Recovery

- [Disaster Recovery Runbook](disaster-recovery.md) - Complete service restoration procedures
- [Incident response guide](../observability/incident-response.md)
- [Troubleshooting](../../operations/troubleshooting/index.md)
- [Key management operations](../kms/key-management.md)
- [Backup and Restore Guide](../deployment/backup-restore.md)

## Observability and Health

- [Monitoring](../observability/monitoring.md)
- [Smoke test guide](../../getting-started/smoke-test.md)

## Suggested Operator Flow

1. Read release notes for behavior changes
2. Apply policy review checklist and rollout changes
3. Run smoke tests and policy smoke tests before traffic cutover
4. Verify denied/allowed patterns in audit logs after rollout
5. Use failure playbooks and troubleshooting for incidents

## Operator Quick Card

Use this section during rollout and incidents when you need a fast, minimal checklist.

### Rollout Preflight (5-minute check)

1. Confirm target version and image tag match release plan
2. Confirm DB connectivity and migration window
3. Confirm key mode settings (`KMS_PROVIDER` + `KMS_KEY_URI`)
4. Confirm token and route rate-limit settings are intentional
5. Confirm rollback owner and communication channel

Primary references:

- [Production rollout golden path](../deployment/production-rollout.md)
- [Release notes](../../releases/RELEASES.md)

### Baseline Verification (before traffic cutover)

1. `GET /health` returns `200`
2. `GET /ready` returns `200`
3. `POST /v1/token` returns `201`
4. Secrets write/read passes
5. Transit encrypt/decrypt passes

Reference:

- [Smoke test guide](../../getting-started/smoke-test.md)

### Fast Status Triage (`401` / `403` / `429`)

1. `401`: re-check credentials/token issuance path
2. `403`: verify policy path and capability mapping
3. `429`: check `Retry-After`, then decide per-client vs token-IP tuning path

References:

- [API error decision matrix](../../api/fundamentals.md#error-decision-matrix)
- [API rate limiting](../../api/fundamentals.md#rate-limiting)
- [Monitoring](../observability/monitoring.md)

### Token Endpoint `429` Quick Path

1. Confirm `429` concentrated on `POST /v1/token`
2. Verify shared NAT/proxy egress is not collapsing many clients to one IP
3. Validate trusted proxy and forwarded header behavior
4. Apply temporary `RATE_LIMIT_TOKEN_*` tuning only if traffic is legitimate
5. Revert temporary tuning after stability

References:

- [Production token throttling runbook](../deployment/docker-hardened.md)
- [Trusted proxy reference](../deployment/docker-hardened.md)

### Rollback Triggers

- Sustained elevated `5xx`
- Widespread token/auth failures
- Unexpected data-integrity behavior
- Failed verification gates after rollout

Reference:

- [Rollback procedure](../deployment/production-rollout.md#rollback-procedure-binaryimage)

### Incident Notes Minimum

Capture these before closing:

- timeline (detection -> mitigation -> recovery)
- affected routes/clients
- config changes applied (`RATE_LIMIT_*`, `RATE_LIMIT_TOKEN_*`, policy updates)
- final mitigation and follow-up owner

## Operator Drills (Quarterly)

Use this section for quarterly game-day exercises that validate operational readiness.

### Drill Catalog

| Drill | Scenario | Primary runbooks | Evidence to collect |
| --- | --- | --- | --- |
| Credential compromise | Client secret leaked | `production.md`, `key-management.md`, `incident-response.md` | revocation timeline, new client IDs, audit evidence |
| Key rotation under load | KEK/master-key rotation while traffic is active | `key-management.md`, `production-rollout.md` | rotation timestamps, restart logs, smoke checks |
| Traffic surge / throttling | Burst traffic causes `429` pressure | `monitoring.md`, `api/fundamentals.md#rate-limiting` | `429` ratio, retry behavior, threshold decision |
| Database outage | DB unreachable / failover | `disaster-recovery.md`, `backup-restore.md`, `incident-response.md` | outage timeline, failover duration, restore checks |

### Quarterly Execution Template

1. Pick one drill owner and one incident commander
2. Define blast radius and rollback boundary
3. Execute drill in staging (or prod shadow) with fixed timebox
4. Capture metrics, logs, and runbook deviations
5. Produce remediation actions with owners and due dates

### Pass Criteria

- Critical runbooks are executable without undocumented tribal knowledge
- On-call can identify root cause and containment path within target SLA
- Recovery path is validated by health checks and smoke tests
- Postmortem includes at least one docs/process improvement item

### Evidence Checklist

- Timeline with UTC timestamps
- Request IDs for key failure and recovery events
- Alert timeline (fired, acknowledged, resolved)
- Commands executed and operator decisions
- Follow-up tickets and target completion dates
