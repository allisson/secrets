# 🚀 Production Rollout Golden Path

> **Document version**: v0.13.0  
> Last updated: 2026-02-25

Use this runbook for a standard production rollout with verification and rollback checkpoints.

## Scope

- Deploy target: Secrets (latest)

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

> Command status: verified on 2026-02-20

```bash
# 1) Pull target release
docker pull allisson/secrets

# 2) Run migrations
docker run --rm --network secrets-net --env-file .env allisson/secrets migrate

# 3) Bootstrap KEK only for first-time environment setup
docker run --rm --network secrets-net --env-file .env allisson/secrets create-kek --algorithm aes-gcm

# 4) Start API
docker run --rm --name secrets-api --network secrets-net --env-file .env -p 8080:8080 \
  allisson/secrets server

```

## Verification Gates

Gate A (before traffic):

- `GET /health` returns `200` - see [Health Check Endpoints](../observability/health-checks.md)

- `GET /ready` returns `200` - see [Health Check Endpoints](../observability/health-checks.md)

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

## Rollback Testing Procedure

**Purpose**: Validate that you can safely rollback to the previous version in production without data loss or service disruption.

**When to test**:

- Before major version upgrades (e.g., v0.11.0 → v0.13.0)

- After significant schema changes or breaking changes

- As part of quarterly disaster recovery drills

- Before high-traffic events (sales, launches)

**Time required**: 15-30 minutes per environment

### Pre-Test Checklist

Before beginning rollback testing:

1. **Document current state**:

   ```bash
   # Capture current version
   docker exec secrets-api /app/secrets --version > version-before.txt
   
   # Capture database schema version
   docker exec secrets-db psql -U secrets -c "SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;" > schema-version.txt
   
   # Take database backup
   docker exec secrets-db pg_dump -U secrets secrets > backup-$(date +%Y%m%d-%H%M%S).sql
   ```

2. **Verify prerequisites**:
   - [ ] Database backup completed successfully

   - [ ] Previous version image/binary available (`docker images | grep secrets`)

   - [ ] `.env` file backed up (contains config for both versions)

   - [ ] Monitoring/alerting temporarily disabled or acknowledged

   - [ ] Traffic load is at baseline (not during peak hours)

3. **Communication**:
   - [ ] Notify team of rollback test window

   - [ ] Set status page to "maintenance" (if applicable)

   - [ ] Prepare incident channel for real-time updates

### Test Procedure

#### Step 1: Capture Baseline Metrics

```bash
# Test current version (e.g., v0.13.0)
curl -s http://localhost:8080/health | jq .
curl -s http://localhost:8080/ready | jq .

# Test secrets functionality
export CLIENT_ID="your-client-id"
export CLIENT_SECRET="your-client-secret"

# Get token
TOKEN=$(curl -s -X POST http://localhost:8080/v1/token \
  -u "${CLIENT_ID}:${CLIENT_SECRET}" | jq -r .access_token)

# Write test secret
curl -s -X POST http://localhost:8080/v1/secrets \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"data": {"test": "rollback-test-v0.13.0"}}' | jq . > test-secret-new.json

# Record secret ID
export SECRET_ID=$(cat test-secret-new.json | jq -r .id)

```

#### Step 2: Perform Rollback

**Docker**:

```bash
# Stop current version
docker stop secrets-api

# Start previous version (e.g., v0.9.0 - use actual previous version)
docker run -d --name secrets-api \
  --network secrets-net \
  --env-file .env \
  -p 8080:8080 \
  allisson/secrets:v<PREVIOUS_VERSION> server

```

**Docker Compose**:

```bash
# Update docker-compose.yml to use previous version
sed -i.bak 's|allisson/secrets:v0.14.1|allisson/secrets:v<PREVIOUS_VERSION>|' docker-compose.yml

# Restart service
docker-compose up -d secrets-api

```

#### Step 3: Verify Rollback Success

```bash
# 1. Verify version rolled back
docker exec secrets-api /app/secrets --version
# Expected: Version: v0.11.0

# 2. Health checks
curl -s http://localhost:8080/health | jq .
# Expected: {"status": "ok"}

curl -s http://localhost:8080/ready | jq .
# Expected: {"status": "ready", "database": "ok"}

# 3. Verify existing data readable (secret created in Step 1)
TOKEN=$(curl -s -X POST http://localhost:8080/v1/token \
  -u "${CLIENT_ID}:${CLIENT_SECRET}" | jq -r .access_token)

curl -s -X GET "http://localhost:8080/v1/secrets/${SECRET_ID}" \
  -H "Authorization: Bearer ${TOKEN}" | jq .
# Expected: Should return the secret created in Step 1

# 4. Test write functionality
curl -s -X POST http://localhost:8080/v1/secrets \
  -H "Authorization: Bearer ${TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"data": {"test": "rollback-test-v0.11.0"}}' | jq .
# Expected: 201 Created

# 5. Check logs for errors
docker logs secrets-api --tail 50
# Expected: No errors, warnings acceptable

```

#### Step 4: Test Forward Rollout (Optional)

After confirming rollback works, test rolling forward again:

```bash
# Roll forward to new version
docker stop secrets-api
docker run -d --name secrets-api \
  --network secrets-net \
  --env-file .env \
  -p 8080:8080 \
  allisson/secrets:v0.14.1 server

# Verify health and functionality (repeat Step 3 checks)

```

#### Step 5: Document Results

Record test results in your runbook:

```markdown
## Rollback Test Results - [Date]

- **Versions tested**: v0.13.0 → v0.11.0 → v0.13.0

- **Environment**: staging/production

- **Rollback time**: [X minutes]

- **Data loss**: None / [Describe if any]

- **Issues encountered**: [List any problems]

- **Rollback success**: ✅ Yes / ❌ No (explain)

### Verification Checklist

- [x] Health checks passed

- [x] Existing secrets readable

- [x] New secrets writable

- [x] Transit encryption functional

- [x] Authentication working

- [x] No errors in logs

### Lessons Learned
[Document any issues, workarounds, or improvements needed]

```

### Common Rollback Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Container fails to start (v0.10.0 → v0.9.0) | Volume permissions (v0.10.0 runs as UID 65532) | Remove volume or `chown 65532:65532` on host directory |
| Database migrations incompatible | Forward-only migrations applied | Restore database from backup before rollback |
| Secrets unreadable after rollback | KEK rotation or KMS key change | Verify `MASTER_KEY_*` env vars match original config |
| 500 errors on `/v1/secrets` | Database connection failure | Check `DB_CONNECTION_STRING`, network connectivity |
| Authentication failures | Client secret hash format changed | Recreate clients or use backup `.env` |

### Version-Specific Notes

**v0.10.0 → v0.9.0 Rollback**:

- ✅ **Safe**: No database migrations in v0.10.0, rollback is data-safe

- ⚠️ **Volume permissions**: v0.10.0 runs as non-root (UID 65532), v0.9.0 runs as root

  - If using bind mounts, files created by v0.10.0 may be unreadable by v0.9.0

  - Solution: Use named volumes or `chown` host directory back to root

- ⚠️ **Healthcheck format**: v0.9.0 uses `/health`, v0.10.0 uses `/health` + `/ready`

  - Update orchestration probes if rolling back

**General Rollback Rules**:

- **Database migrations**: Keep applied unless documented rollback procedure exists

- **KMS keys**: Never change `MASTER_KEY_*` config during rollback

- **Environment variables**: Use same `.env` for both versions (additive changes only)

- **Volumes**: Test with both bind mounts and named volumes

### Rollback Automation

For production environments, consider automating rollback testing:

```bash
#!/bin/bash
# rollback-test.sh - Automated rollback verification

set -e

CURRENT_VERSION="v0.13.0"
PREVIOUS_VERSION="v0.11.0"
BASE_URL="http://localhost:8080"

echo "=== Rollback Test: ${CURRENT_VERSION} → ${PREVIOUS_VERSION} ==="

# Step 1: Test current version
echo "Testing current version..."
docker run -d --name secrets-test --network secrets-net --env-file .env -p 8080:8080 \
  allisson/secrets:${CURRENT_VERSION} server
sleep 5
curl -f ${BASE_URL}/health || exit 1

# Step 2: Rollback
echo "Rolling back to ${PREVIOUS_VERSION}..."
docker stop secrets-test && docker rm secrets-test
docker run -d --name secrets-test --network secrets-net --env-file .env -p 8080:8080 \
  allisson/secrets:${PREVIOUS_VERSION} server
sleep 5

# Step 3: Verify
echo "Verifying rollback..."
curl -f ${BASE_URL}/health || exit 1
docker logs secrets-test --tail 20

# Cleanup
docker stop secrets-test && docker rm secrets-test
echo "✅ Rollback test passed"

```

### Post-Test Actions

After completing rollback testing:

1. **Restore to target version**: If you rolled back during testing, roll forward again
2. **Update documentation**: Record any issues or workarounds discovered
3. **Re-enable monitoring**: Remove maintenance mode, re-enable alerting
4. **Notify team**: Share test results and any action items
5. **Schedule next test**: Quarterly or before next major release

## Post-Rollout Checklist

- Confirm token expiration behavior matches configured policy

- Confirm CORS behavior matches expected browser/server mode

- Confirm rate limiting thresholds are appropriate for production traffic

- Schedule cleanup routines (`clean-audit-logs`, `clean-expired-tokens` if tokenization enabled)

## See also

- [Production deployment guide](../deployment/docker-hardened.md)

- [Release notes](../../releases/RELEASES.md)

- [KMS migration checklist](../kms/setup.md#migration-checklist)

- [Release compatibility matrix](../../releases/compatibility-matrix.md)

- [Smoke test guide](../../getting-started/smoke-test.md)
