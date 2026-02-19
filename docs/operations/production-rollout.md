# 🚀 Production Rollout Golden Path

> Last updated: 2026-02-19

Use this runbook for a standard production rollout with verification and rollback checkpoints.

## Scope

- Deploy target: Secrets `v0.6.0`
- Database schema changes: run migrations before traffic cutover
- Crypto bootstrap: ensure initial KEK exists for write/encrypt flows

## Golden Path

1. Deploy new image/binary to staging/prod environment
2. Run migrations once per environment
3. Verify KEK presence (create only if first bootstrap)
4. Start/roll API instances with health checks
5. Execute smoke checks and policy checks
6. Shift traffic gradually and monitor 4xx/5xx/latency

## Copy/Paste Rollout Commands

```bash
# 1) Pull target release
docker pull allisson/secrets:v0.6.0

# 2) Run migrations
docker run --rm --network secrets-net --env-file .env allisson/secrets:v0.6.0 migrate

# 3) Bootstrap KEK only for first-time environment setup
docker run --rm --network secrets-net --env-file .env allisson/secrets:v0.6.0 create-kek --algorithm aes-gcm

# 4) Start API
docker run --rm --name secrets-api --network secrets-net --env-file .env -p 8080:8080 \
  allisson/secrets:v0.6.0 server
```

## Verification Gates

Gate A (before traffic):

- `GET /health` returns `200`
- `GET /ready` returns `200`
- `POST /v1/token` returns `201`

Gate B (functional):

- Secrets flow write/read passes
- Transit encrypt/decrypt passes
- Tokenization flow (if enabled) passes

Gate C (policy and observability):

- Expected denied actions produce `403`
- Load behavior returns controlled `429` with `Retry-After`
- Metrics and logs ingest normally

## Rollback Trigger Conditions

- Sustained elevated `5xx`
- Widespread auth/token issuance failures
- Migration side effects not recoverable via config changes
- Data integrity concerns

## Rollback Procedure (Binary/Image)

1. Freeze rollout and stop new traffic shift
2. Roll API instances back to previous stable image
3. Keep additive migrations applied unless a validated DB rollback plan exists
4. Re-run health + smoke checks on rolled-back version
5. Capture incident notes and remediation actions

## Post-Rollout Checklist

- Confirm token expiration behavior matches configured policy
- Confirm CORS behavior matches expected browser/server mode
- Confirm rate limiting thresholds are appropriate for production traffic
- Schedule cleanup routines (`clean-audit-logs`, `clean-expired-tokens` if tokenization enabled)

## See also

- [Production deployment guide](production.md)
- [v0.6.0 release notes](../releases/v0.6.0.md)
- [v0.6.0 upgrade guide](../releases/v0.6.0-upgrade.md)
- [KMS migration checklist](kms-migration-checklist.md)
- [Release compatibility matrix](../releases/compatibility-matrix.md)
- [Smoke test guide](../getting-started/smoke-test.md)
