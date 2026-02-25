# 🎭 Documentation Paths by Persona

> Last updated: 2026-02-25

Choose your learning path based on your role and goals. Each persona path provides a curated journey through the documentation.

## 📑 Quick Navigation

- [👷 Operator Path](#operator-path) - Deployment and operations
- [👨‍💻 Developer Path](#developer-path) - API integration and development
- [🛡️ Security Engineer Path](#security-engineer-path) - Hardening and auditability

---

## Operator Path

Use this path when your goal is reliable deployment and fast incident response.

### Primary path

1. [Day 0 Operator Walkthrough](../getting-started/day-0-walkthrough.md#operator-path)
2. [Production rollout golden path](../operations/deployment/production-rollout.md)
3. [Operator quick card](../operations/runbooks/README.md#operator-quick-card)
4. [Incident response guide](../operations/observability/incident-response.md)

### Deep links

- [Monitoring](../operations/observability/monitoring.md)
- [Known limitations](../operations/deployment/docker-hardened.md)

---

## Developer Path

Use this path when your goal is API integration and feature delivery with docs parity.

### Primary path

1. [Day 0 Developer Walkthrough](../getting-started/day-0-walkthrough.md#developer-path)
2. [Authentication API](../api/auth/authentication.md)
3. [Error decision matrix](../api/fundamentals.md#error-decision-matrix)
4. [Rate limiting](../api/fundamentals.md#rate-limiting)
5. [Code examples](../examples/README.md)

### Deep links

- [Capability matrix](../api/fundamentals.md#capability-matrix)
- [Policies cookbook](../api/auth/policies.md)
- [Docs release checklist](../contributing.md#docs-release-checklist)

---

## Security Engineer Path

Use this path when your goal is threat reduction, hardening, and auditability.

### Primary path

1. [Security hardening guide](../operations/deployment/docker-hardened.md)
2. [Trusted proxy reference](../operations/deployment/docker-hardened.md)
3. [Known limitations](../operations/deployment/docker-hardened.md)
4. [Production deployment guide](../operations/deployment/docker-hardened.md)
5. [Monitoring](../operations/observability/monitoring.md)

### Deep links

- [Security model](../concepts/security-model.md)
- [KMS setup guide](../operations/kms/setup.md)
- [KMS migration checklist](../operations/kms/setup.md#migration-checklist)
