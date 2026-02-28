# 🧭 Day 0 Walkthrough

> Last updated: 2026-02-28

Choose your onboarding path based on your role:

## 📑 Quick Navigation

- [👷 Operator Path](#operator-path) - For deployment and operations
- [👨‍💻 Developer Path](#developer-path) - For API integration

---

## 👷 Operator Path {#operator-path}

Use this linear path for first-time operations onboarding.

### Step 1: Bring up a local baseline

- Follow: [Run with Docker](docker.md)
- Verify: `GET /health`, `GET /ready`

### Step 2: Validate core flows

- Run: [Smoke test script](smoke-test.md)
- Confirm token issuance, secrets, and transit checks pass

### Step 3: Learn rollout and rollback flow

- Read: [Production rollout golden path](../operations/deployment/production-rollout.md)
- Focus: verification gates and rollback triggers

### Step 4: Learn incident response path

- Use: [Incident response guide](../operations/observability/incident-response.md)
- Drill: [Operator drills](../operations/runbooks/README.md#operator-drills-quarterly)

### Step 5: Harden production posture

- Read: [Production deployment guide](../operations/deployment/docker-hardened.md)
- Read: [Security hardening guide](../operations/deployment/docker-hardened.md)
- Check: [Known limitations](../operations/deployment/docker-hardened.md)

### Expected outcomes

- You can validate service health and auth quickly
- You can identify `401/403/429/5xx` primary runbook path
- You can execute a basic rollback trigger decision under pressure

### See also

- [Operator quick card](../operations/runbooks/README.md#operator-quick-card)
- [Operator runbook index](../operations/runbooks/README.md)

---

## 👨‍💻 Developer Path {#developer-path}

Use this path for first-time contributors integrating with Secrets APIs.

### Step 1: Run locally

- Follow: [Run locally](local-development.md)
- Build and start API, then verify health

### Step 2: Understand auth + policy behavior

- Read: [Authentication API](../api/auth/authentication.md)
- Read: [Policies cookbook](../api/auth/policies.md)
- Read: [Capability matrix](../api/fundamentals.md#capability-matrix)

### Step 3: Validate error and retry behavior

- Read: [API error decision matrix](../api/fundamentals.md#error-decision-matrix)
- Read: [API rate limiting](../api/fundamentals.md#rate-limiting)

### Step 4: Use examples by runtime

- Start with: [Code examples](../examples/README.md)
- Then use: [Curl](../examples/curl.md), [Python](../examples/python.md), [JavaScript](../examples/javascript.md), [Go](../examples/go.md)

### Step 5: Follow docs contribution quality bar

- Read: [Documentation contributing guide](../contributing.md)
- Use: [Docs release checklist](../contributing.md#docs-release-checklist)

### Expected outcomes

- You can obtain tokens and call protected endpoints reliably
- You can distinguish authn/authz/throttling failures in client integrations
- You can submit feature PRs with aligned API + docs changes

### See also

- [Development and testing](../contributing.md#development-and-testing)
- [Docs architecture map](../contributing.md#docs-architecture-map)
