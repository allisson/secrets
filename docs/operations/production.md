# 🏭 Production Deployment Guide

> Last updated: 2026-02-19

This guide covers baseline production hardening and operations for Secrets.

**For comprehensive security hardening, see [Security Hardening Guide](security-hardening.md).**

## 📑 Table of Contents

- [1) TLS and Reverse Proxy](#1-tls-and-reverse-proxy)
- [2) Runtime Configuration and Secrets](#2-runtime-configuration-and-secrets)
- [3) Database Operations](#3-database-operations)
- [4) Key Rotation Schedule](#4-key-rotation-schedule)
- [5) Observability and Monitoring](#5-observability-and-monitoring)
- [6) Retention Defaults](#6-retention-defaults)
- [7) Incident Response Checklist](#7-incident-response-checklist)
- [8) Go-Live Checklist](#8-go-live-checklist)
- [9) Golden Path Rollout (Recommended)](#9-golden-path-rollout-recommended)

## 1) TLS and Reverse Proxy

- Run Secrets behind HTTPS/TLS termination (Nginx, Envoy, Traefik, ALB, or equivalent)
- Enforce HTTPS-only access
- Set request size limits and conservative upstream timeouts
- Restrict public exposure of admin paths by network policy

Minimal reverse proxy checklist:

1. TLS certificate management in place
2. HTTP -> HTTPS redirect enabled
3. Forwarded headers configured correctly
4. Access logs and request IDs preserved

## 2) Runtime Configuration and Secrets

- Inject env vars via secure runtime mechanism (orchestrator secrets, vault/KMS integrations)
- Do not bake `MASTER_KEYS` into images
- Use distinct clients/policies per workload
- Keep token expiration short enough for your threat model

## 3) Database Operations

- Enable DB backups and test restores regularly
- Use encrypted storage and restricted DB network access
- Monitor connection pool metrics and error rates
- Run migrations before rolling out new app versions
- Define and execute audit log retention cleanup on a fixed cadence
- Define and execute expired token cleanup on a fixed cadence when tokenization is enabled

Backup/restore checklist:

1. Daily backup configured
2. Retention policy defined
3. Restore drill tested in non-production
4. Recovery time objective documented

Audit log retention routine (recommended monthly):

```bash
# 1) Preview rows older than 90 days
./bin/app clean-audit-logs --days 90 --dry-run --format json

# 2) Execute deletion
./bin/app clean-audit-logs --days 90 --format text
```

Token retention routine (recommended monthly for tokenization workloads):

```bash
# 1) Preview expired tokens older than 30 days
./bin/app clean-expired-tokens --days 30 --dry-run --format json

# 2) Execute deletion
./bin/app clean-expired-tokens --days 30 --format text
```

## 4) Key Rotation Schedule

- Rotate KEKs on a fixed cadence (for example every 90 days)
- Rotate immediately after suspected compromise
- Rotate client credentials periodically and on team changes
- Review and prune unused clients/policies
- Restart API servers after master key or KEK rotation so processes load new values

Suggested monthly routine:

1. Review active clients and policies
2. Inspect audit logs for denied and unusual access patterns
3. Confirm backup and restore readiness
4. Validate runbooks and on-call contacts

Rolling restart runbook after key rotation:

Single-node:

1. Rotate master key and/or KEK
2. Stop API process
3. Start API process
4. Verify `GET /health` and run smoke test flow

Multi-node:

1. Rotate master key and/or KEK
2. Restart one instance at a time (rolling)
3. Wait for `GET /health` success before moving to next instance
4. After all instances restart, validate secrets and transit operations

## 5) Observability and Monitoring

- Alert on elevated `401`/`403` rates
- Alert on repeated denied authorization attempts from same client/IP
- Track API latency and error rates by endpoint
- Correlate request failures using `request_id`
- Scrape and alert on `secrets_http_requests_total`, `secrets_http_request_duration_seconds`, and `secrets_operations_total`

Secure `/metrics` in production:

1. Keep `/metrics` reachable only from internal monitoring networks
2. Restrict source IP ranges at load balancer or reverse proxy
3. If needed, add proxy-level auth in front of `/metrics`
4. Do not expose `/metrics` on public internet-facing routes

SLO examples (starting point):

- API availability: 99.9% monthly
- Health endpoint latency (`GET /health`): p95 < 100 ms
- Token issuance latency (`POST /v1/token`): p95 < 300 ms
- Secrets read/write latency (`GET/POST /v1/secrets/*`): p95 < 500 ms
- Server error budget: 5xx < 0.1% of total requests

## 6) Retention Defaults

| Dataset | Suggested retention | Cleanup command | Cadence |
| --- | --- | --- | --- |
| Audit logs | 90 days | `clean-audit-logs --days 90` | Monthly |
| Expired tokens | 30 days | `clean-expired-tokens --days 30` | Monthly |

Adjust retention to match your compliance and incident-response requirements.

## 7) Incident Response Checklist

1. Identify affected client/key/path scope
2. Revoke/deactivate compromised clients
3. Rotate KEK (and master key if needed)
4. Perform rolling restart of API servers to pick up rotated key material
5. Reissue credentials with least-privilege policies
6. Review audit logs for lateral movement or unusual access
7. Record timeline and remediation actions

## 8) Go-Live Checklist

- [ ] HTTPS/TLS enabled and verified
- [ ] DB backups and restore drill validated
- [ ] `MASTER_KEYS` stored securely outside source control
- [ ] Initial KEK created and documented
- [ ] Restart procedure documented after master key or KEK rotation
- [ ] Least-privilege policies applied for all clients
- [ ] Monitoring alerts configured
- [ ] Incident response owner and process documented

## 9) Golden Path Rollout (Recommended)

- Follow [Production rollout golden path](production-rollout.md) for step-by-step deployment,
  verification gates, and rollback triggers
- Use [Release compatibility matrix](../releases/compatibility-matrix.md) before planning upgrades
- Keep [v0.5.1 upgrade guide](../releases/v0.5.1-upgrade.md) attached to rollout change tickets

## See also

- [Security hardening guide](security-hardening.md)
- [Key management operations](key-management.md)
- [Production rollout golden path](production-rollout.md)
- [Operator runbook index](runbook-index.md)
- [Monitoring](monitoring.md)
- [Operator drills (quarterly)](operator-drills.md)
- [Policy smoke tests](policy-smoke-tests.md)
- [v0.5.1 release notes](../releases/v0.5.1.md)
- [v0.5.1 upgrade guide](../releases/v0.5.1-upgrade.md)
- [Release compatibility matrix](../releases/compatibility-matrix.md)
- [Environment variables](../configuration/environment-variables.md)
- [Security model](../concepts/security-model.md)
- [Troubleshooting](../getting-started/troubleshooting.md)
