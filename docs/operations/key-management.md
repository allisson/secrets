# 🔑 Key Management Operations

> Last updated: 2026-02-14

This guide covers master keys and KEK lifecycle operations.

## Master Keys

Generate:

```bash
./bin/app create-master-key --id prod-2026-01
```

Docker image equivalent:

```bash
docker run --rm allisson/secrets:latest create-master-key --id prod-2026-01
```

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
3. Rotate KEK
4. Verify secret read/write and transit encrypt/decrypt
5. Review audit logs for anomalies

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

- 🏭 Production deployment: `docs/operations/production.md`

## See also

- [Production operations](production.md)
- [Security model](../concepts/security-model.md)
- [Transit API](../api/transit.md)
- [Environment variables](../configuration/environment-variables.md)
