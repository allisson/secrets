# 🚨 Incident Response Guide

> Last updated: 2026-02-25

This guide provides fast incident triage workflows, decision paths, and failure playbooks for common API issues.

## 📑 Quick Navigation

- [Quick Start: First 15 Minutes](#quick-start-first-15-minutes) - High-severity incident triage
- [Incident Decision Tree](#incident-decision-tree) - Fast status code branching
- [Failure Playbooks](#failure-playbooks) - Detailed remediation for specific errors

**Fast Branches**: [401](#401-unauthorized) | [403](#403-forbidden) | [429](#429-too-many-requests) | [5xx](#5xx)

---

## Quick Start: First 15 Minutes

Use this for high-severity incidents where API availability or auth flows are degraded.

### Minute 0-3: Establish Service State

```bash
curl -i http://localhost:8080/health
curl -i http://localhost:8080/ready
```

Expected:

- `GET /health` -> `200`
- `GET /ready` -> `200`

### Minute 3-6: Validate Authentication Path

```bash
curl -i -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{"client_id":"<client-id>","client_secret":"<client-secret>"}'
```

Expected:

- Normal flow -> `201 Created`
- If throttled -> `429` with `Retry-After`

### Minute 6-10: Validate Crypto Data Path

```bash
TOKEN="<token>"

curl -i -X POST http://localhost:8080/v1/secrets/incident/check \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"value":"aW5jaWRlbnQtY2hlY2s="}'

curl -i -X GET http://localhost:8080/v1/secrets/incident/check \
  -H "Authorization: Bearer ${TOKEN}"
```

Expected:

- write/read path succeeds

### Minute 10-15: Decide Mitigation Path

1. `401`-heavy: credential/token issue → [401 Spike Playbook](#401-spike-unauthorized)
2. `403`-heavy: policy mismatch → [403 Spike Playbook](#403-spike-policycapability-mismatch) and [Policy smoke tests](../runbooks/policy-smoke-tests.md)
3. `429` on `/v1/token`: IP throttling/proxy path → [Token throttling runbook](../deployment/docker-hardened.md)
4. `5xx`/readiness failures: dependency/runtime path → [Production rollout rollback triggers](../deployment/production-rollout.md#rollback-trigger-conditions)

---

## Incident Decision Tree

Use this to route incidents quickly to the right runbook.

### Decision Flow

1. Is `GET /health` failing?
   - Yes → infrastructure/runtime path: Follow [First 15 Minutes](#quick-start-first-15-minutes) above
   - No → continue
2. Is `GET /ready` failing?
   - Yes → dependencies/migrations/key-load path: [Troubleshooting](../../operations/troubleshooting/index.md)
   - No → continue
3. Identify dominant status code and route group:
   - `401` → [401 Spike Playbook](#401-spike-unauthorized)
   - `403` → [403 Spike Playbook](#403-spike-policycapability-mismatch)
   - `429` on `/v1/token` → [Token throttling runbook](../deployment/docker-hardened.md)
   - `429` on authenticated routes → [API rate limiting](../../api/fundamentals.md#rate-limiting)
   - `422` → [API error decision matrix](../../api/fundamentals.md#error-decision-matrix)
   - `5xx` → [First 15 Minutes](#quick-start-first-15-minutes)

### Fast Branches

#### `401 Unauthorized`

- Re-issue token via `POST /v1/token`
- Confirm caller sends `Authorization: Bearer <token>`
- Check client active status and secret rotation history

#### `403 Forbidden`

- Verify endpoint path shape and required capability
- Verify policy matching semantics (`*`, trailing `/*`, mid-path `*`)
- Re-issue token after policy fix

#### `429 Too Many Requests`

- Read `Retry-After` header
- Separate `/v1/token` from authenticated-route throttling
- Validate proxy/source-IP behavior if `/v1/token` is impacted

#### `5xx`

- Check database connectivity and pool saturation
- Check migration and key-load startup logs
- Use rollback triggers in production rollout runbook

### Search Aliases

- `retry-after`
- `rate limit exceeded`
- `token endpoint throttling`
- `unauthorized spike`
- `forbidden policy mismatch`

---

## Failure Playbooks

Use these for fast incident triage on common API failures.

### 401 Spike (Unauthorized)

**Symptoms:**

- Sudden increase in `401` across multiple endpoints

**Triage steps:**

1. Verify token issuance with `POST /v1/token`
2. Confirm callers send `Authorization: Bearer <token>`
3. Check token expiry and client active state
4. Inspect audit logs for broad denied patterns

### 403 Spike (Policy/Capability Mismatch)

**Symptoms:**

- Valid tokens but access denied with `403`

**Triage steps:**

1. Identify failing endpoint path and required capability
2. Confirm client policy path matching (`*`, exact, trailing `/*`, and mid-path `*` segment rules)
3. Validate capability mapping for endpoint (`read`, `write`, `delete`, `encrypt`, `decrypt`, `rotate`)
4. Re-issue token after policy update

### 409 on Transit Key Create

**Symptoms:**

- `POST /v1/transit/keys` returns `409 Conflict`

**Triage steps:**

1. Treat conflict as "key already initialized"
2. Call `POST /v1/transit/keys/:name/rotate` to create a new active version
3. Confirm encrypt/decrypt still work after rotation
4. Update automation to avoid repeated create for existing names

### 404/422 on Tokenization Detokenize

**Symptoms:**

- `POST /v1/tokenization/detokenize` returns `404 Not Found` or `422 Unprocessable Entity`

**Triage steps:**

1. Confirm token was produced by `POST /v1/tokenization/keys/:name/tokenize`
2. Confirm request shape uses JSON body `{"token":"..."}` (not URL path token)
3. Check if token is expired (`ttl`) or revoked
4. Validate caller has `decrypt` capability on `/v1/tokenization/detokenize`
5. If expired tokens accumulate, run cleanup routine (`clean-expired-tokens`)

### 409 on Tokenization Key Create

**Symptoms:**

- `POST /v1/tokenization/keys` returns `409 Conflict`

**Triage steps:**

1. Treat conflict as "key already initialized"
2. Call `POST /v1/tokenization/keys/:name/rotate` for a new active version
3. Confirm tokenize/detokenize paths remain healthy after rotation
4. Update automation to avoid repeated create for existing names

---

## Quick Commands

```bash
# Health
curl -s http://localhost:8080/health

# Token check
curl -i -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{"client_id":"<client-id>","client_secret":"<client-secret>"}'

# Audit logs snapshot
curl -s "http://localhost:8080/v1/audit-logs?limit=50&offset=0" \
  -H "Authorization: Bearer <token>"
```

---

## Command Status Markers

> Command status: verified on 2026-02-20

---

## See also

- [Production rollout golden path](../deployment/production-rollout.md)
- [Troubleshooting](../../operations/troubleshooting/index.md)
- [Operator quick card](../runbooks/README.md#operator-quick-card)
- [Policies cookbook](../../api/auth/policies.md)
- [Policy smoke tests](../runbooks/policy-smoke-tests.md)
- [Transit API](../../api/data/transit.md)
- [Tokenization API](../../api/data/tokenization.md)
- [API rate limiting](../../api/fundamentals.md#rate-limiting)
- [API error decision matrix](../../api/fundamentals.md#error-decision-matrix)
- [Production operations](../deployment/docker-hardened.md)
