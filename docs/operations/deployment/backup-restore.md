# 💾 Backup and Restore Guide

> **Document version**: v0.x
> Last updated: 2026-02-25
> **Audience**: Platform engineers, SREs, DBAs
>
> **⚠️ UNTESTED PROCEDURES**: The procedures in this guide are reference examples and have not been tested in production. Always test in a non-production environment first and adapt to your infrastructure.

This guide covers backup and restore procedures for Secrets, including database backups, master key backups, and recovery validation.

## Table of Contents

- [Overview](#overview)

- [What to Back Up](#what-to-back-up)

- [Database Backup Procedures](#database-backup-procedures)

- [Master Key Backup](#master-key-backup)

- [Restore Procedures](#restore-procedures)

- [Automation Examples](#automation-examples)

- [Troubleshooting](#troubleshooting)

- [See Also](#see-also)

## Overview

Secrets stores two critical types of data:

1. **Database**: Encrypted secrets, transit keys, clients, audit logs
2. **Master Key**: Used to decrypt Key Encryption Keys (KEKs)

**CRITICAL**: Without both the database AND the master key, you cannot decrypt stored secrets. Backups of one without the other are useless.

### Backup Strategy

| Component | Backup Method | Frequency | Retention |
|-----------|---------------|-----------|-----------|
| Database | `pg_dump` / `mysqldump` | Hourly | 30 days |
| Master Key | KMS snapshot / encrypted file | On rotation | Forever |
| Application Config | Git repository | On change | Forever |

### Recovery Time Objective (RTO)

- **Database restore**: 5-30 minutes (depends on database size)

- **Master key restore**: < 5 minutes (KMS) or immediate (plaintext backup)

- **Full service recovery**: 15-60 minutes

### Recovery Point Objective (RPO)

- **Database**: Last successful backup (hourly = max 1 hour data loss)

- **Master key**: No data loss (immutable after creation)

## What to Back Up

### 1. Database (REQUIRED)

**PostgreSQL tables**:

```sql
-- Core tables

clients
key_encryption_keys
secrets
transit_keys
transit_key_versions
audit_logs
schema_migrations

-- All tables in public schema

SELECT table_name FROM information_schema.tables 
WHERE table_schema='public';

```

**MySQL tables**:

```sql
-- Same table list

SHOW TABLES;

```

### 2. Master Key (REQUIRED)

**KMS-based deployments**:

- KMS key ID/ARN (e.g., `aws:kms:us-east-1:123456789012:key/abc-def-123`)

- KMS key policy and IAM permissions

- No file backup needed (KMS handles durability)

**Plaintext-based deployments**:

- Base64-encoded 32-byte master key

- Store in encrypted vault (1Password, HashiCorp Vault, etc.)

### 3. Configuration (RECOMMENDED)

**Environment variables**:

```bash
# Critical config
DB_DRIVER=postgres
DB_CONNECTION_STRING=postgres://...
MASTER_KEY_PROVIDER=aws-kms
MASTER_KEY_KMS_KEY_ID=arn:aws:kms:...

# Rate limiting, CORS, etc.
RATE_LIMIT_ENABLED=true
AUTH_TOKEN_EXPIRATION_SECONDS=14400

```

**Store in**:

- Git repository (without secrets)

- Infrastructure-as-Code (Terraform, CloudFormation)

- Configuration management (Ansible, SaltStack)

### 4. Audit Logs (OPTIONAL)

If compliance requires long-term audit log retention:

- Export audit logs to S3/GCS/Azure Blob

- Use append-only storage with versioning

- Verify signatures before export

## Database Backup Procedures

### PostgreSQL Backup

**Full database dump**:

```bash
# Backup to file
pg_dump \
  --host=localhost \
  --port=5432 \
  --username=secrets \
  --dbname=secrets \
  --format=custom \
  --compress=9 \
  --file=secrets-backup-$(date +%Y%m%d-%H%M%S).dump

# Backup with verbose output
pg_dump \
  --host=localhost \
  --port=5432 \
  --username=secrets \
  --dbname=secrets \
  --format=custom \
  --compress=9 \
  --verbose \
  --file=secrets-backup-$(date +%Y%m%d-%H%M%S).dump

```

**Encrypted backup**:

```bash
# Dump and encrypt with GPG
pg_dump \
  --host=localhost \
  --port=5432 \
  --username=secrets \
  --dbname=secrets \
  --format=custom \
  | gpg --encrypt --recipient ops@example.com \
  > secrets-backup-$(date +%Y%m%d-%H%M%S).dump.gpg

```

**Upload to S3**:

```bash
# Backup and upload
BACKUP_FILE="secrets-backup-$(date +%Y%m%d-%H%M%S).dump"
pg_dump --host=localhost --username=secrets --dbname=secrets \
  --format=custom --compress=9 --file=$BACKUP_FILE

aws s3 cp $BACKUP_FILE s3://my-backups/secrets/ \
  --storage-class GLACIER \
  --server-side-encryption AES256

# Verify upload
aws s3 ls s3://my-backups/secrets/$BACKUP_FILE

```

### MySQL Backup

**Full database dump**:

```bash
# Backup to file
mysqldump \
  --host=localhost \
  --port=3306 \
  --user=secrets \
  --password \
  --databases secrets \
  --single-transaction \
  --quick \
  --compress \
  --result-file=secrets-backup-$(date +%Y%m%d-%H%M%S).sql

# Compressed backup
mysqldump \
  --host=localhost \
  --user=secrets \
  --password \
  --databases secrets \
  --single-transaction \
  | gzip > secrets-backup-$(date +%Y%m%d-%H%M%S).sql.gz

```

**Encrypted backup**:

```bash
# Dump and encrypt
mysqldump \
  --host=localhost \
  --user=secrets \
  --password \
  --databases secrets \
  --single-transaction \
  | gpg --encrypt --recipient ops@example.com \
  > secrets-backup-$(date +%Y%m%d-%H%M%S).sql.gpg

```

## Master Key Backup

### KMS-Based Master Key

**AWS KMS**:

```bash
# Get key metadata
aws kms describe-key --key-id alias/secrets-master-key

# Export key policy (for disaster recovery)
aws kms get-key-policy \
  --key-id alias/secrets-master-key \
  --policy-name default \
  > kms-key-policy-backup.json

# List key aliases
aws kms list-aliases | grep secrets

```

**IMPORTANT**: AWS KMS keys cannot be exported. Backup the key ID/ARN and ensure:

- KMS key policy allows your AWS account to use the key

- IAM roles/policies are backed up

- Multi-region key replication is configured (optional)

**GCP Cloud KMS**:

```bash
# Get key metadata
gcloud kms keys describe secrets-master-key \
  --location=us-east1 \
  --keyring=secrets

# Export key location (for disaster recovery)
echo "projects/my-project/locations/us-east1/keyRings/secrets/cryptoKeys/secrets-master-key" \
  > kms-key-id-backup.txt

```

### Plaintext Master Key

**Backup procedure**:

```bash
# Export master key from environment
echo $MASTER_KEY_PLAINTEXT > master-key-backup.txt

# Encrypt with GPG
gpg --encrypt --recipient ops@example.com master-key-backup.txt

# Store encrypted backup in vault
# NEVER commit plaintext master key to git

```

**Storage options**:

- 1Password / LastPass / Bitwarden

- HashiCorp Vault

- AWS Secrets Manager / GCP Secret Manager

- Encrypted USB drive in physical safe

## Restore Procedures

### Database Restore

**PostgreSQL restore**:

```bash
# Restore from dump file
pg_restore \
  --host=localhost \
  --port=5432 \
  --username=secrets \
  --dbname=secrets \
  --clean \
  --if-exists \
  --verbose \
  secrets-backup-20260221-120000.dump

# Restore from S3
aws s3 cp s3://my-backups/secrets/secrets-backup-20260221-120000.dump .
pg_restore --host=localhost --username=secrets --dbname=secrets \
     --clean --if-exists secrets-backup-20260221-120000.dump

```

1. **Restore master key** (KMS example):

   ```bash
   export MASTER_KEY_PROVIDER=aws-kms
   export MASTER_KEY_KMS_KEY_ID=arn:aws:kms:us-east-1:123456789012:key/abc-def
   ```

2. **Start application**:

   ```bash
   ./bin/app server
   ```

3. **Verify health**:

   ```bash
   curl http://localhost:8080/health
   curl http://localhost:8080/ready
   ```

4. **Test secret decryption**:

   ```bash
   # Get auth token
   TOKEN=$(curl -X POST http://localhost:8080/v1/token \
     -H "Content-Type: application/json" \
     -d '{"client_id":"xxx","client_secret":"yyy"}' | jq -r .token)

   # Retrieve a known secret
   curl -X GET http://localhost:8080/v1/secrets/my-test-secret \
     -H "Authorization: Bearer $TOKEN"
   ```

5. **Restore master key** (KMS example):

   ```bash
   export MASTER_KEY_PROVIDER=aws-kms
   export MASTER_KEY_KMS_KEY_ID=arn:aws:kms:us-east-1:123456789012:key/abc-def
   ```

6. **Start application**:

   ```bash
   ./bin/app server
   ```

7. **Verify health**:

   ```bash
   curl http://localhost:8080/health
   curl http://localhost:8080/ready
   ```

8. **Test secret decryption**:

   ```bash
   # Get auth token
   TOKEN=$(curl -X POST http://localhost:8080/v1/token \
     -H "Content-Type: application/json" \
     -d '{"client_id":"xxx","client_secret":"yyy"}' | jq -r .token)

   # Retrieve a known secret
   curl -X GET http://localhost:8080/v1/secrets/my-test-secret \
     -H "Authorization: Bearer $TOKEN"
   ```

## Backup Validation

### Test Restore in Non-Production

**Monthly validation**:

```bash
# 1. Create test database
createdb secrets-restore-test

# 2. Restore backup to test database
pg_restore --host=localhost --username=secrets \
  --dbname=secrets-restore-test \
  secrets-backup-latest.dump

# 3. Start app against test database
DB_CONNECTION_STRING=postgres://localhost/secrets-restore-test \
  ./bin/app server

# 4. Verify data integrity
curl http://localhost:8080/health
curl http://localhost:8080/ready

# 5. Drop test database
dropdb secrets-restore-test

```

### Verify Backup Integrity

**PostgreSQL**:

```bash
# Verify dump file is valid
pg_restore --list secrets-backup-20260221-120000.dump | head -20

# Count tables in backup
pg_restore --list secrets-backup-20260221-120000.dump | grep TABLE | wc -l

```

**MySQL**:

```bash
# Verify SQL file is valid
head -50 secrets-backup-20260221-120000.sql

# Count tables in backup
grep -c "CREATE TABLE" secrets-backup-20260221-120000.sql

```

### Verify Master Key Access

**KMS-based**:

```bash
# Test encryption with KMS key
echo "test data" | \
  aws kms encrypt --key-id alias/secrets-master-key \
  --plaintext fileb:///dev/stdin \
  --query CiphertextBlob --output text

```

**Plaintext-based**:

```bash
# Verify base64 decode works
echo $MASTER_KEY_PLAINTEXT | base64 -d | wc -c
# Should output: 32

```

## Automation Examples

### Cron-Based Backup (PostgreSQL)

```bash
# /etc/cron.d/secrets-backup
# Run hourly backup at minute 0
0 * * * * postgres /opt/scripts/backup-secrets.sh

# /opt/scripts/backup-secrets.sh
#!/bin/bash
set -euo pipefail

BACKUP_DIR=/var/backups/secrets
BACKUP_FILE="secrets-backup-$(date +\%Y\%m\%d-\%H\%M\%S).dump"

# Create backup
pg_dump --host=localhost --username=secrets --dbname=secrets \
  --format=custom --compress=9 --file=$BACKUP_DIR/$BACKUP_FILE

# Upload to S3
aws s3 cp $BACKUP_DIR/$BACKUP_FILE s3://my-backups/secrets/ \
  --storage-class STANDARD_IA

# Delete local backups older than 7 days
find $BACKUP_DIR -name "secrets-backup-*.dump" -mtime +7 -delete

# Delete S3 backups older than 30 days
# (Use S3 lifecycle policy instead)

```

## Troubleshooting

### Backup fails with "Permission denied"

**Cause**: Database user lacks permissions

**Solution**:

```sql
-- PostgreSQL: Grant permissions

GRANT SELECT ON ALL TABLES IN SCHEMA public TO secrets;

-- MySQL: Grant permissions

GRANT SELECT ON secrets.* TO 'secrets'@'%';

```

### Restore fails with "database already exists"

**Cause**: Target database already exists

**Solution**:

```bash
# PostgreSQL: Use --clean flag
pg_restore --clean --if-exists secrets-backup.dump

# MySQL: Drop database first
mysql -e "DROP DATABASE IF EXISTS secrets;"
mysql -e "CREATE DATABASE secrets;"
mysql secrets < secrets-backup.sql

```

### Restored data is encrypted garbage

**Cause**: Master key mismatch (wrong key used for restore)

**Solution**:

```bash
# Verify master key matches original
# 1. Check KMS key ID
echo $MASTER_KEY_KMS_KEY_ID

# 2. Or check plaintext key hash
echo $MASTER_KEY_PLAINTEXT | sha256sum

# 3. Restore with correct master key

```

### Backup file is too large

**Cause**: Audit logs table is huge

**Solution**:

```bash
# PostgreSQL: Exclude audit logs from backup
pg_dump --exclude-table=audit_logs \
  --format=custom --compress=9 \
  --file=secrets-backup-no-audit.dump

# Backup audit logs separately
pg_dump --table=audit_logs \
  --format=custom --compress=9 \
  --file=audit-logs-backup.dump

```

### S3 upload fails with "Access Denied"

**Cause**: AWS credentials missing or invalid

**Solution**:

```bash
# Verify AWS credentials
aws sts get-caller-identity

# Test S3 access
aws s3 ls s3://my-backups/secrets/

# Check IAM policy allows s3:PutObject

```

## See Also

- [Production Deployment Guide](docker-hardened.md) - Pre-production checklist includes backup validation

- [Disaster Recovery Runbook](../runbooks/disaster-recovery.md) - Full DR procedures

- [Database Scaling Guide](database-scaling.md) - Backup considerations for large databases

- [Security Hardening Guide](docker-hardened.md) - Backup encryption best practices
