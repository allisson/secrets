# 🧪 CLI Commands Reference

> Last updated: 2026-02-14

Use the `app` CLI for server runtime, key management, and client lifecycle operations.

## Binary and Docker forms

Local binary:

```bash
./bin/app <command> [flags]
```

Docker image (v0.2.0):

```bash
docker run --rm --env-file .env allisson/secrets:v0.2.0 <command> [flags]
```

## Core Runtime

### `server`

Starts the HTTP API server.

```bash
./bin/app server
```

### `migrate`

Runs database migrations.

```bash
./bin/app migrate
```

## Key Management

### `create-master-key`

Generates a new 32-byte master key and prints `MASTER_KEYS` / `ACTIVE_MASTER_KEY_ID` values.

Flags:

- `--id`, `-i`: master key ID

```bash
./bin/app create-master-key --id default
```

### `create-kek`

Creates an initial KEK from the active master key.

Flags:

- `--algorithm`, `--alg`: `aes-gcm` (default) or `chacha20-poly1305`

```bash
./bin/app create-kek --algorithm aes-gcm
```

### `rotate-kek`

Rotates KEK to a new version.

Flags:

- `--algorithm`, `--alg`: `aes-gcm` (default) or `chacha20-poly1305`

```bash
./bin/app rotate-kek --algorithm aes-gcm
```

After master key or KEK rotation, restart API server instances so they load updated key material.

## Client Management

### `create-client`

Creates an API client and returns `client_id` plus one-time `secret`.

Flags:

- `--name`, `-n` (required): client name
- `--active`, `-a` (default `true`): whether client can authenticate immediately
- `--policies`, `-p`: JSON array of policy documents (omit to use interactive mode)
- `--format`, `-f`: `text` (default) or `json`

Non-interactive example:

```bash
./bin/app create-client \
  --name bootstrap-admin \
  --active \
  --policies '[{"path":"*","capabilities":["read","write","delete","encrypt","decrypt","rotate"]}]' \
  --format json
```

Interactive example:

```bash
./bin/app create-client --name bootstrap-admin
```

### `update-client`

Updates client name, active state, and policies.

Flags:

- `--id`, `-i` (required): client UUID
- `--name`, `-n` (required): client name
- `--active`, `-a` (default `true`): whether client can authenticate
- `--policies`, `-p`: JSON array of policy documents (omit to use interactive mode)
- `--format`, `-f`: `text` (default) or `json`

```bash
./bin/app update-client \
  --id <client-uuid> \
  --name payments-api \
  --active=true \
  --policies '[{"path":"/v1/secrets/*","capabilities":["read","encrypt"]}]' \
  --format json
```

## Output and Safety Notes

- `create-client` secret is shown once and cannot be retrieved later
- Prefer `--format json` for automation
- Store client secrets in a secure secret manager
- Use least-privilege policies per workload and path

## Audit Log Maintenance

### `clean-audit-logs`

Deletes audit logs older than a specified retention period.

Flags:

- `--days`, `-d` (required): delete logs older than this many days
- `--dry-run`, `-n` (default `false`): preview count without deleting
- `--format`, `-f`: `text` (default) or `json`

Examples:

```bash
# Preview (no deletion)
./bin/app clean-audit-logs --days 90 --dry-run

# Execute deletion
./bin/app clean-audit-logs --days 90 --format text

# Docker form
docker run --rm --network secrets-net --env-file .env allisson/secrets:v0.2.0 \
  clean-audit-logs --days 90 --dry-run --format json
```

Example text output:

```text
Dry-run mode: Would delete 1234 audit log(s) older than 90 day(s)
```

Example JSON output:

```json
{
  "count": 1234,
  "days": 90,
  "dry_run": true
}
```

Requirements:

- Database must be reachable and migrated
- Use `--dry-run` before deletion in production environments

## See also

- [Docker getting started](../getting-started/docker.md)
- [Local development](../getting-started/local-development.md)
- [Authentication API](../api/authentication.md)
- [Policies cookbook](../api/policies.md)
