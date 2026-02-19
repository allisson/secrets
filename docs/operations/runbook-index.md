# 🧭 Operator Runbook Index

> Last updated: 2026-02-19

Use this page as the single entry point for rollout, validation, and incident runbooks.

## Release and Rollout

- [v0.6.0 release notes](../releases/v0.6.0.md)
- [v0.6.0 upgrade guide](../releases/v0.6.0-upgrade.md)
- [Release compatibility matrix](../releases/compatibility-matrix.md)
- [Production rollout golden path](production-rollout.md)
- [Production deployment guide](production.md)
- [KMS setup guide](kms-setup.md)
- [KMS migration checklist](kms-migration-checklist.md)

## Authorization Policy Validation

- [Policies cookbook](../api/policies.md)
- [Path matching behavior](../api/policies.md#path-matching-behavior)
- [Route shape vs policy shape](../api/policies.md#route-shape-vs-policy-shape)
- [Policy review checklist before deploy](../api/policies.md#policy-review-checklist-before-deploy)
- [Policy smoke tests](policy-smoke-tests.md)

## API and Access Verification

- [Capability matrix](../api/capability-matrix.md)
- [API error decision matrix](../api/error-decision-matrix.md)
- [Authentication API](../api/authentication.md)
- [Audit logs API](../api/audit-logs.md)

## Incident and Recovery

- [Failure playbooks](failure-playbooks.md)
- [Operator drills (quarterly)](operator-drills.md)
- [Troubleshooting](../getting-started/troubleshooting.md)
- [Key management operations](key-management.md)

## Observability and Health

- [Monitoring](monitoring.md)
- [Smoke test guide](../getting-started/smoke-test.md)

## Suggested Operator Flow

1. Read release notes for behavior changes and upgrade notes
2. Apply policy review checklist and rollout changes
3. Run smoke tests and policy smoke tests before traffic cutover
4. Verify denied/allowed patterns in audit logs after rollout
5. Use failure playbooks and troubleshooting for incidents
