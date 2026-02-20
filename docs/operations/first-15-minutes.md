# ⏱️ First 15 Minutes Playbook

> Last updated: 2026-02-20

Use this for high-severity incidents where API availability or auth flows are degraded.

## Minute 0-3: Establish Service State

```bash
curl -i http://localhost:8080/health
curl -i http://localhost:8080/ready
```

Expected:

- `GET /health` -> `200`
- `GET /ready` -> `200`

## Minute 3-6: Validate Authentication Path

```bash
curl -i -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{"client_id":"<client-id>","client_secret":"<client-secret>"}'
```

Expected:

- Normal flow -> `201 Created`
- If throttled -> `429` with `Retry-After`

## Minute 6-10: Validate Crypto Data Path

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

## Minute 10-15: Decide Mitigation Path

1. `401`-heavy: credential/token issue -> [Failure playbooks](failure-playbooks.md)
2. `403`-heavy: policy mismatch -> [Policy smoke tests](policy-smoke-tests.md)
3. `429` on `/v1/token`: IP throttling/proxy path -> [Token throttling runbook](production.md#10-token-endpoint-throttling-runbook)
4. `5xx`/readiness failures: dependency/runtime path -> [Production rollout rollback triggers](production-rollout.md#rollback-trigger-conditions)

## Command status markers

> Command status: verified on 2026-02-20

## See also

- [Incident decision tree](incident-decision-tree.md)
- [Production rollout golden path](production-rollout.md)
- [Troubleshooting](../getting-started/troubleshooting.md)
