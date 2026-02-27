# 🧪 CLI Commands Reference

> Last updated: 2026-02-26

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

## Key Management

### `create-master-key`

Generates a new 32-byte master key and prints `MASTER_KEYS` / `ACTIVE_MASTER_KEY_ID` values.
KMS mode is **required** in v0.19.0+ (breaking change).

Required Flags:

- `--id`, `-i`: master key ID
- `--kms-provider`: KMS provider (`localsecrets`, `gcpkms`, `awskms`, `azurekeyvault`, `hashivault`)
- `--kms-key-uri`: KMS key URI

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

Generates a new master key, combines it with existing `MASTER_KEYS`, and sets the new key as active.

Flags:

- `--id`, `-i`: new master key ID

Local:

```bash
./bin/app rotate-master-key --id master-key-2026-08
```

Docker:

```bash
docker run --rm --env-file .env allisson/secrets rotate-master-key --id master-key-2026-08
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
./bin/app rotate-master-key --id master-key-2026-08
# update env vars from output
# rolling restart API instances
./bin/app rotate-kek --algorithm aes-gcm
# remove old master key from MASTER_KEYS after verification
```

### `verify-audit-logs`

Verifies cryptographic integrity of audit logs within a time range. Validates HMAC-SHA256 signatures against KEK-derived signing keys for tamper detection.

**Requirements:**

- Database migrated to version 000003 (signature columns)
- KEK chain loaded (for verifying signed logs)

Flags:

- `--start-date`, `-s`: start date (format: `YYYY-MM-DD` or `YYYY-MM-DD HH:MM:SS`)
- `--end-date`, `-e`: end date (format: `YYYY-MM-DD` or `YYYY-MM-DD HH:MM:SS`)
- `--format`, `-f`: output format (`text` or `json`, default: `text`)

Local:

```bash
# Verify today's audit logs (text output)
TODAY=$(date +%Y-%m-%d)
./bin/app verify-audit-logs --start-date "$TODAY" --end-date "$TODAY"

# Verify date range (JSON output for automation)
./bin/app verify-audit-logs \
  --start-date "2026-02-01" \
  --end-date "2026-02-20" \
  --format json

# Verify with datetime precision
./bin/app verify-audit-logs \
  --start-date "2026-02-20 00:00:00" \
  --end-date "2026-02-20 23:59:59"
```

Docker:

```bash
docker run --rm --env-file .env allisson/secrets \
  verify-audit-logs \
  --start-date "2026-02-20" \
  --end-date "2026-02-20" \
  --format text
```

Output (text format):

```text
Audit Log Integrity Verification
=================================

Time Range: 2026-02-20 00:00:00 to 2026-02-20 23:59:59

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

Time Range: 2026-02-15 00:00:00 to 2026-02-15 23:59:59

Total Checked:  100
Signed:         100
Unsigned:       0
Valid:          95
Invalid:        5

Status: FAILED ✗

Details:
- 5 audit logs with invalid signatures detected
- KEK chain may be missing historical KEKs
- Use --format json to identify specific failed log IDs
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

## See also

- [Docker getting started](getting-started/docker.md)
- [Local development](getting-started/local-development.md)
- [Authentication API](api/auth/authentication.md)
- [Policies cookbook](api/auth/policies.md)
