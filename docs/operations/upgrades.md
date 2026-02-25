# 🆙 Upgrades & Rollbacks

> Last updated: 2026-02-25

This guide serves as the universal runbook for standard Secrets upgrades and rollbacks. Because Secrets is a stateless binary/container, upgrades are typically straightforward.

## Standard Upgrade Flow

For minor or patch releases that do not require explicitly documented migration steps (e.g. `0.x.0` to `0.x.1`), use the standard procedure:

### 1. Preparation

- **Release Notes**: Check the [Changelog](../releases/RELEASES.md) for breaking changes.
- **Backup**: Always backup your database before modifying the deployment.

```bash
pg_dump $DB_CONNECTION_STRING > secrets_backup_$(date +%s).sql
```

### 2. Apply Migrations

If the release contains schema changes, they are safely embedded in the binary. Migrate the database *before* starting the new application instances.

```bash
docker run --rm --env-file .env allisson/secrets:<new_version> migrate
```

### 3. Deploy New Application

Perform a rolling restart of your API instances or Docker Compose stack.

```bash
# Docker Compose example
docker compose pull secrets-api
docker compose up -d secrets-api
```

### 4. Verification

Check the health and readiness probes to confirm successful startup:

```bash
curl -sS http://localhost:8080/health
curl -sS http://localhost:8080/ready
```

## Rollback Instructions

If an upgrade introduces breaking unexpected behavior, restoring service stability is the priority.

### Rolling Back the App Version

Downgrading the binary/container is fast since Secrets is stateless:

```bash
# Example
docker run -d --name secrets-api --env-file .env allisson/secrets:<old_version> server
```

### Reverting Database Migrations

If the bad release included database migrations, you must apply the down migrations *before* or *immediately after* downgrading the app to prevent schema mismatches:

1. Identify the applied migrations from the `schema_migrations` table.
2. Run the down migrations located in `migrations/postgresql/` or `migrations/mysql/`.

```bash
# PostgreSQL example
psql $DB_CONNECTION_STRING < migrations/postgresql/000004_add_account_lockout.down.sql
```

> [!WARNING]
> Down migrations are often destructive (e.g. dropping columns added in the new release). Always refer back to your pre-upgrade database backup if there is any data loss during rollback.

## Version-Specific Guides

Some major updates require specific procedural steps beyond the standard upgrade guide. Read these guides carefully before proceeding with those specific upgrades:

- **v0.11.0**: [Upgrade Guide](../releases/v0.11.0-upgrade.md) (Introduces Account Lockout)
- **v0.10.0**: [Base Image Migration](deployment/docker-hardened.md) (Introduces Distroless and UID 65532)
- **v0.9.0**: [Upgrade Guide](../releases/v0.9.0-upgrade.md) (Introduces Cryptographic Audit Log Signatures)
