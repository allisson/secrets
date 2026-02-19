# 🧯 Operator Drills (Quarterly)

> Last updated: 2026-02-19

Use this page for quarterly game-day exercises that validate operational readiness.

## Drill Catalog

| Drill | Scenario | Primary runbooks | Evidence to collect |
| --- | --- | --- | --- |
| Credential compromise | Client secret leaked | `production.md`, `key-management.md`, `failure-playbooks.md` | revocation timeline, new client IDs, audit evidence |
| Key rotation under load | KEK/master-key rotation while traffic is active | `key-management.md`, `production-rollout.md` | rotation timestamps, restart logs, smoke checks |
| Traffic surge / throttling | Burst traffic causes `429` pressure | `monitoring.md`, `../api/rate-limiting.md` | `429` ratio, retry behavior, threshold decision |
| Database outage | DB unreachable / failover | `failure-playbooks.md`, `production.md` | outage timeline, failover duration, restore checks |

## Quarterly Execution Template

1. Pick one drill owner and one incident commander
2. Define blast radius and rollback boundary
3. Execute drill in staging (or prod shadow) with fixed timebox
4. Capture metrics, logs, and runbook deviations
5. Produce remediation actions with owners and due dates

## Pass Criteria

- Critical runbooks are executable without undocumented tribal knowledge
- On-call can identify root cause and containment path within target SLA
- Recovery path is validated by health checks and smoke tests
- Postmortem includes at least one docs/process improvement item

## Evidence Checklist

- Timeline with UTC timestamps
- Request IDs for key failure and recovery events
- Alert timeline (fired, acknowledged, resolved)
- Commands executed and operator decisions
- Follow-up tickets and target completion dates

## See also

- [Production rollout golden path](production-rollout.md)
- [Production deployment guide](production.md)
- [Failure playbooks](failure-playbooks.md)
- [Monitoring](monitoring.md)
- [Troubleshooting](../getting-started/troubleshooting.md)
