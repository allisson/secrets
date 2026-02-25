# 🔑 Key Management Operations

> Last updated: 2026-02-20

This guide covers master keys and KEK lifecycle operations.

## Master Keys

Generate:

```bash
./bin/app create-master-key --id prod-2026-01

# KMS mode (recommended for production)
./bin/app create-master-key --id prod-2026-01 \
  --kms-provider=localsecrets \
  --kms-key-uri="base64key://<base64-32-byte-key>"
```

Docker image equivalent:

```bash
docker run --rm allisson/secrets create-master-key --id prod-2026-01
```

Rotate master key:

```bash
./bin/app rotate-master-key --id prod-2026-08
```

`rotate-master-key` reads current `MASTER_KEYS`, appends a new key, and sets
`ACTIVE_MASTER_KEY_ID` to the new key. It does not remove old keys automatically.

Set output in environment:

```dotenv
MASTER_KEYS=prod-2026-01:<base64-key>
ACTIVE_MASTER_KEY_ID=prod-2026-01
```

## Create Initial KEK

```bash
./bin/app create-kek --algorithm aes-gcm
```

Requirements:

- DB reachable and migrated
- `MASTER_KEYS` and `ACTIVE_MASTER_KEY_ID` configured

## Rotate KEK

```bash
./bin/app rotate-kek --algorithm aes-gcm
```

Why rotate:

- regular hygiene (for example every 90 days)
- incident response
- algorithm transitions
- after master key changes

Rotation behavior:

- creates a new active KEK version
- keeps older KEKs for decrypting historical DEKs
- does not break reads for existing data

## Restart Required After Master Key or KEK Rotation

After rotating a master key or KEK, restart all Secrets API servers.

Reason:

- master key chain is loaded once at process startup
- active KEK context is loaded once at process startup
- running processes do not hot-reload new key values

Operational step:

1. Rotate master key and/or KEK
2. Perform rolling restart of all API server instances
3. Verify health and read/write/transit operations

## Suggested Runbook

1. Validate backups and operational readiness
2. Rotate master key (if planned)
3. Restart API instances to load the updated master key chain
4. Rotate KEK
5. Re-wrap existing DEKs with the new KEK using `rewrap-deks`
6. Verify secret read/write and transit encrypt/decrypt
7. Remove old master key from `MASTER_KEYS` after KEK rotation completes
8. Review audit logs for anomalies

## Copy/Paste Rotation Runbook

Use this sequence for master key rotation with minimal operator drift:

```bash
# 1) Generate next master key entry
./bin/app rotate-master-key --id prod-2026-08

# 2) Update MASTER_KEYS / ACTIVE_MASTER_KEY_ID from command output
# 3) Restart API instances (rolling)

# 4) Rotate KEK to re-wrap with active master key
./bin/app rotate-kek --algorithm aes-gcm

# 5) Determine new KEK ID and rewrap DEKs
# Get the new active KEK ID from the database or logs, then:
./bin/app rewrap-deks --kek-id "<new-kek-id>" --batch-size 100

# 6) Validate key-dependent paths
curl -sS http://localhost:8080/health
curl -sS http://localhost:8080/ready

# 7) Remove old master key from MASTER_KEYS
# 8) Restart API instances again
```

## Transit Create/Rotate Automation

When automating transit key lifecycle:

- call create once per key name (`POST /v1/transit/keys`)
- if create returns `409 Conflict`, treat it as "key already initialized"
- call rotate (`POST /v1/transit/keys/:name/rotate`) to create a new active version

Example decision flow:

```text
create key -> 201 Created: continue
create key -> 409 Conflict: rotate key
```

## Related

- 🏭 Production deployment: `docs/operations/deployment/production.md`

## See also

- [Production operations](../deployment/production.md)
- [Security model](../../concepts/security-model.md)
- [Transit API](../../api/data/transit.md)
- [Environment variables](../../configuration.md)
- [KMS setup guide](setup.md)
