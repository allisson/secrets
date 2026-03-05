# 🧪 CLI Commands Reference

Use the `app` CLI for server runtime, key management, and client lifecycle operations.

## Binary and Docker forms

Local binary:

```bash
./bin/app <command> [flags]
```

Docker image:

```bash
docker run --rm --env-file .env allisson/secrets <command> [flags]
```

## Core Runtime

### `server`

Starts the HTTP API server.

Local:

```bash
./bin/app server
```

Docker:

```bash
docker run --rm --network secrets-net --env-file .env -p 8080:8080 allisson/secrets server
```

### `migrate`

Runs database migrations.

Local:

```bash
./bin/app migrate
```

Docker:

```bash
docker run --rm --network secrets-net --env-file .env allisson/secrets migrate
```

### `migrate-down`

Rolls back database migrations. This command should **only be used for emergency rollbacks**, not for regular operations.

Flags:

- `--steps`, `-n` (default `1`): number of migrations to rollback

Local:

```bash
# Rollback the last migration
./bin/app migrate-down

# Rollback the last 3 migrations
./bin/app migrate-down --steps 3
```

Docker:

```bash
docker run --rm --network secrets-net --env-file .env allisson/secrets migrate-down --steps 1
```

**Important warnings:**

- Migration rollbacks are **potentially destructive** operations that may result in data loss
- Always backup your database before running rollback operations
- Only use this command for emergency rollbacks (e.g., after a failed migration)
- For production systems, test rollback procedures in a staging environment first
- Consider forward-only migrations instead of rollbacks when possible

Requirements:

- Database must be reachable
- Down migration SQL files must exist for the migrations being rolled back

## Key Management

### `create-master-key`

Generates a new 32-byte master key and prints `MASTER_KEYS` / `ACTIVE_MASTER_KEY_ID` values.
KMS mode is **required** in v0.19.0+ (breaking change).

Required Flags:

- `--kms-provider`: KMS provider (`localsecrets`, `gcpkms`, `awskms`, `azurekeyvault`, `hashivault`)
- `--kms-key-uri`: KMS key URI

Flags:

- `--id`, `-i`: master key ID (optional, defaults to `master-key-YYYY-MM-DD`)

For local development, use `localsecrets` provider:

```bash
# Generate a KMS key first
openssl rand -base64 32

# Create master key
./bin/app create-master-key --id default \
  --kms-provider=localsecrets \
  --kms-key-uri="base64key://<base64-32-byte-key>"
```

Docker:

```bash
docker run --rm allisson/secrets create-master-key \
  --id default \
  --kms-provider=localsecrets \
  --kms-key-uri="base64key://<base64-32-byte-key>"
```

### `rotate-master-key`

Generates a new master key and combines it with existing `MASTER_KEYS`. The command outputs the updated environment variables that must be manually applied.

**Requirements:**

- Existing `MASTER_KEYS` and `ACTIVE_MASTER_KEY_ID` must be set in the environment (or `.env` file).
- KMS mode is **required** in v0.19.0+ (breaking change).

**Required Flags:**

- `--kms-provider`: KMS provider (`localsecrets`, `gcpkms`, `awskms`, `azurekeyvault`, `hashivault`)
- `--kms-key-uri`: KMS key URI

**Flags:**

- `--id`, `-i`: new master key ID (optional, defaults to `master-key-YYYY-MM-DD`)

Local:

```bash
# Ensure existing keys are loaded (from .env or exported)
./bin/app rotate-master-key --id master-key-2026-02-27 \
  --kms-provider=localsecrets \
  --kms-key-uri="base64key://<base64-32-byte-key>"
```

Docker:

```bash
docker run --rm --env-file .env allisson/secrets rotate-master-key \
  --id master-key-2026-02-27 \
  --kms-provider=localsecrets \
  --kms-key-uri="base64key://<base64-32-byte-key>"
```

### `create-kek`

Creates an initial KEK from the active master key.

Flags:

- `--algorithm`, `--alg`: `aes-gcm` (default) or `chacha20-poly1305`

Local:

```bash
./bin/app create-kek --algorithm aes-gcm
```

Docker:

```bash
docker run --rm --network secrets-net --env-file .env allisson/secrets create-kek --algorithm aes-gcm
```

### `rotate-kek`

Rotates KEK to a new version.

Flags:

- `--algorithm`, `--alg`: `aes-gcm` (default) or `chacha20-poly1305`

Local:

```bash
./bin/app rotate-kek --algorithm aes-gcm
```

Docker:

```bash
docker run --rm --network secrets-net --env-file .env allisson/secrets rotate-kek --algorithm aes-gcm
```

### `rewrap-deks`

Rewraps all Data Encryption Keys (DEKs) that are not currently encrypted with the specified KEK ID.

Flags:

- `--kek-id`, `-k` (required): target KEK ID to encrypt DEKs with
- `--batch-size`, `-b` (default `100`): number of DEKs to process per batch

Local:

```bash
./bin/app rewrap-deks --kek-id "target-kek-id" --batch-size 100
```

Docker:

```bash
docker run --rm --network secrets-net --env-file .env allisson/secrets rewrap-deks --kek-id "target-kek-id"
```

After master key or KEK rotation, restart API server instances so they load updated key material.

Master key rotation quick sequence:

```bash
./bin/app rotate-master-key --id master-key-2026-02-27 \
  --kms-provider=localsecrets \
  --kms-key-uri="base64key://<base64-32-byte-key>"
# update env vars from output
# rolling restart API instances
./bin/app rotate-kek --algorithm aes-gcm
# remove old master key from MASTER_KEYS after verification
```

## Transit

### `purge-transit-keys`

Permanently deletes soft-deleted transit keys older than a specified number of days. This operation is **irreversible** and any data encrypted with these keys will become permanently inaccessible.

Flags:

- `--days`, `-d` (default `30`): delete transit keys soft-deleted more than this many days ago
- `--dry-run`, `-n` (default `false`): preview count without deleting
- `--format`, `-f`: `text` (default) or `json`

Examples:

```bash
# Preview (no deletion) - see what would be deleted
./bin/app purge-transit-keys --days 30 --dry-run

# Execute deletion with default 30 days threshold
./bin/app purge-transit-keys

# Execute deletion with custom threshold
./bin/app purge-transit-keys --days 90 --format text

# Docker form
docker run --rm --network secrets-net --env-file .env allisson/secrets \
  purge-transit-keys --days 30 --dry-run --format json
```

Example text output:

```text
Successfully deleted 42 transit key(s) older than 30 day(s)
```

Example JSON output:

```json
{
  "count": 42,
  "days": 30,
  "dry_run": false
}
```

Important notes:

- Only affects **soft-deleted** transit keys (where `deleted_at` is set)
- Active transit keys are never touched
- This operation is **irreversible** - any data encrypted with these keys will become permanently inaccessible
- Always use `--dry-run` first to preview the impact
- Recommended to run periodically as part of maintenance

Requirements:

- Database must be reachable and migrated
- Use `--dry-run` before deletion in production environments

## Tokenization

### `create-tokenization-key`

Creates a tokenization key with version `1`.

Flags:

- `--name`, `-n` (required): unique tokenization key name
- `--format`, `--fmt`: `uuid` (default), `numeric`, `luhn-preserving`, or `alphanumeric`
- `--deterministic`, `--det` (default `false`): generate deterministic tokens for identical plaintext
- `--algorithm`, `--alg`: `aes-gcm` (default) or `chacha20-poly1305`

Examples:

```bash
./bin/app create-tokenization-key \
  --name payment-cards \
  --format luhn-preserving \
  --deterministic \
  --algorithm aes-gcm

docker run --rm --network secrets-net --env-file .env allisson/secrets \
  create-tokenization-key --name payment-cards --format luhn-preserving --deterministic --algorithm aes-gcm
```

### `rotate-tokenization-key`

Creates a new version for an existing tokenization key.

Flags:

- `--name`, `-n` (required): tokenization key name to rotate
- `--format`, `--fmt`: `uuid` (default), `numeric`, `luhn-preserving`, or `alphanumeric`
- `--deterministic`, `--det` (default `false`)
- `--algorithm`, `--alg`: `aes-gcm` (default) or `chacha20-poly1305`

Examples:

```bash
./bin/app rotate-tokenization-key \
  --name payment-cards \
  --format luhn-preserving \
  --deterministic \
  --algorithm chacha20-poly1305

docker run --rm --network secrets-net --env-file .env allisson/secrets \
  rotate-tokenization-key --name payment-cards --format luhn-preserving --deterministic --algorithm chacha20-poly1305
```

### `clean-expired-tokens`

Deletes expired tokens older than a retention window.

Flags:

- `--days`, `-d` (required): delete tokens older than this many days
- `--dry-run`, `-n` (default `false`): preview count without deleting
- `--format`, `-f`: `text` (default) or `json`

Examples:

```bash
# Preview (no deletion)
./bin/app clean-expired-tokens --days 30 --dry-run --format json

# Execute deletion
./bin/app clean-expired-tokens --days 30 --format text

# Docker form
docker run --rm --network secrets-net --env-file .env allisson/secrets \
  clean-expired-tokens --days 30 --dry-run --format json
```

### `purge-tokenization-keys`

Permanently deletes soft-deleted tokenization keys and all of their associated tokens older than a specified number of days. This operation is **irreversible** and any tokens generated with these keys will be permanently deleted.

Flags:

- `--days`, `-d` (default `30`): delete tokenization keys soft-deleted more than this many days ago
- `--dry-run`, `-n` (default `false`): preview count of keys without deleting
- `--format`, `-f`: `text` (default) or `json`

Examples:

```bash
# Preview (no deletion) - see what would be deleted
./bin/app purge-tokenization-keys --days 30 --dry-run

# Execute deletion with default 30 days threshold
./bin/app purge-tokenization-keys

# Execute deletion with custom threshold
./bin/app purge-tokenization-keys --days 90 --format text

# Docker form
docker run --rm --network secrets-net --env-file .env allisson/secrets \
  purge-tokenization-keys --days 30 --dry-run --format json
```

Example text output:

```text
Successfully deleted 42 tokenization key(s) older than 30 day(s)
```

Example JSON output:

```json
{
  "count": 42,
  "days": 30,
  "dry_run": false
}
```

Important notes:

- Only affects **soft-deleted** tokenization keys (where `deleted_at` is set)
- Active tokenization keys are never touched
- This operation is **irreversible** - any tokens generated with these keys will be permanently deleted
- Always use `--dry-run` first to preview the impact
- Recommended to run periodically as part of maintenance

Requirements:

- Database must be reachable and migrated
- Use `--dry-run` before deletion in production environments

## Secrets

### `purge-secrets`

Permanently deletes soft-deleted secrets older than a specified number of days. This operation is **irreversible** and should be used with caution.

Flags:

- `--days`, `-d` (default `30`): delete secrets soft-deleted more than this many days ago
- `--dry-run`, `-n` (default `false`): preview count without deleting
- `--format`, `-f`: `text` (default) or `json`

Examples:

```bash
# Preview (no deletion) - see what would be deleted
./bin/app purge-secrets --days 30 --dry-run

# Execute deletion with default 30 days threshold
./bin/app purge-secrets

# Execute deletion with custom threshold
./bin/app purge-secrets --days 90 --format text

# Docker form
docker run --rm --network secrets-net --env-file .env allisson/secrets \
  purge-secrets --days 30 --dry-run --format json
```

Example text output:

```text
Successfully deleted 42 secret(s) older than 30 day(s)
```

Example JSON output:

```json
{
  "count": 42,
  "days": 30,
  "dry_run": false
}
```

Important notes:

- Only affects **soft-deleted** secrets (where `deleted_at` is set)
- Active secrets are never touched
- This operation is **irreversible** - deleted data cannot be recovered
- Always use `--dry-run` first to preview the impact
- Recommended to run periodically as part of maintenance (e.g., monthly cron job)

Requirements:

- Database must be reachable and migrated
- Use `--dry-run` before deletion in production environments

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
  --policies '[{"path":"/v1/secrets/*","capabilities":["decrypt","encrypt"]}]' \
  --format json
```

### `purge-auth-tokens`

Permanently deletes expired and revoked authentication tokens older than a specified number of days.

Flags:

- `--days`, `-d` (default `30`): delete tokens created more than this many days ago that are already expired or revoked
- `--dry-run`, `-n` (default `false`): preview count without deleting (Note: currently only shows notice)
- `--format`, `-f`: `text` (default) or `json`

Examples:

```bash
# Execute deletion with default 30 days threshold
./bin/app purge-auth-tokens

# Execute deletion with custom threshold and JSON output
./bin/app purge-auth-tokens --days 7 --format json

# Docker form
docker run --rm --network secrets-net --env-file .env allisson/secrets \
  purge-auth-tokens --days 30
```

## Audit Logs

### `verify-audit-logs`

Verifies cryptographic integrity of audit logs within a time range. Validates HMAC-SHA256 signatures against KEK-derived signing keys for tamper detection.

**Requirements:**

- Database migrated to version 000003 (signature columns)
- KEK chain loaded (for verifying signed logs)

**Required Flags:**

- `--start-date`, `-s`: start date (format: `YYYY-MM-DD` or `YYYY-MM-DD HH:MM:SS`)
- `--end-date`, `-e`: end date (format: `YYYY-MM-DD` or `YYYY-MM-DD HH:MM:SS`)

**Flags:**

- `--format`, `-f`: output format (`text` or `json`, default: `text`)

Local:

```bash
# Verify today's audit logs (text output)
# Note: end-date must be strictly AFTER start-date
TODAY=$(date +%Y-%m-%d)
TOMORROW=$(date -v+1d +%Y-%m-%d) # macOS/BSD
# TOMORROW=$(date -d "tomorrow" +%Y-%m-%d) # Linux/GNU
./bin/app verify-audit-logs --start-date "$TODAY" --end-date "$TOMORROW"

# Verify with datetime precision for exact day coverage
./bin/app verify-audit-logs \
  --start-date "2026-02-27 00:00:00" \
  --end-date "2026-02-27 23:59:59"

# Verify date range (JSON output for automation)
./bin/app verify-audit-logs \
  --start-date "2026-02-01" \
  --end-date "2026-02-27" \
  --format json
```

Docker:

```bash
docker run --rm --env-file .env allisson/secrets \
  verify-audit-logs \
  --start-date "2026-02-27" \
  --end-date "2026-02-28" \
  --format text
```

Output (text format):

```text
Audit Log Integrity Verification
=================================

Time Range: 2026-02-27 00:00:00 to 2026-02-27 23:59:59

Total Checked:  150
Signed:         120
Unsigned:       30 (legacy)
Valid:          120
Invalid:        0

Status: PASSED ✓
```

Output (JSON format):

```json
{
  "total_checked": 150,
  "signed_count": 120,
  "unsigned_count": 30,
  "valid_count": 120,
  "invalid_count": 0,
  "invalid_logs": [],
  "passed": true
}
```

Output (failed verification):

```text
Audit Log Integrity Verification
=================================

Time Range: 2026-02-27 00:00:00 to 2026-02-27 23:59:59

Total Checked:  100
Signed:         100
Unsigned:       0 (legacy)
Valid:          95
Invalid:        5

WARNING: 5 log(s) failed integrity check!

Invalid Log IDs:
  - 0194f4a6-7ec7-78e6-9fe7-5ca35fef48db
  - 0194f4a6-7ec7-78e6-9fe7-5ca35fef48dc
  ...

Status: FAILED ❌
```

Exit Codes:

- `0`: All signatures valid (or no logs found)
- `1`: Invalid signatures detected (integrity check failed)

Use Cases:

- Periodic compliance audits
- Incident investigation and tamper detection
- Post-KEK-rotation verification
- Continuous monitoring integration (CI/CD, cron jobs)

Notes:

- Legacy unsigned logs (`is_signed=false`) are counted but not verified
- Invalid signatures indicate potential tampering or KEK mismatch
- Verification requires KEK IDs referenced in audit logs to exist in database

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
docker run --rm --network secrets-net --env-file .env allisson/secrets \
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

## Output and Safety Notes

- `create-client` secret is shown once and cannot be retrieved later
- Prefer `--format json` for automation
- Store client secrets in a secure secret manager
- Use least-privilege policies per workload and path

## See also

- [Docker getting started](getting-started/docker.md)
- [Local development](getting-started/local-development.md)
- [Authentication API](auth/authentication.md)
- [Policies cookbook](auth/policies.md)
