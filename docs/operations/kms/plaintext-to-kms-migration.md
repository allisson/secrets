# 🔑 Plaintext to KMS Migration Guide

> **Document version**: v0.13.0  
> Last updated: 2026-02-25  
> **Audience**: Security engineers, SRE teams, platform engineers
>
> **⚠️ UNTESTED PROCEDURES**: The procedures in this guide are reference examples and have not been tested in production. Always test in a non-production environment first and adapt to your infrastructure.

This guide walks you through migrating from plaintext master keys to cloud KMS providers (AWS KMS, GCP Cloud KMS, Azure Key Vault) for enhanced security and compliance.

## Table of Contents

- [Overview](#overview)

- [Migration Planning](#migration-planning)

- [Pre-Migration Checklist](#pre-migration-checklist)

- [Migration Procedures](#migration-procedures)

- [Validation](#validation)

- [Rollback Plan](#rollback-plan)

- [Post-Migration](#post-migration)

- [Troubleshooting](#troubleshooting)

- [See Also](#see-also)

## Overview

### Why Migrate to KMS

**Security benefits**:

- **Hardware security**: Keys stored in FIPS 140-2 Level 3 (AWS/GCP) or Level 2 (Azure) HSMs

- **Access control**: IAM policies restrict key usage to authorized services

- **Audit logging**: All key operations logged to CloudTrail/Cloud Audit Logs/Azure Monitor

- **Key rotation**: Automatic key rotation without re-encrypting data

- **Compliance**: Meets SOC 2, PCI-DSS, HIPAA, ISO 27001 requirements

**Operational benefits**:

- **No key management**: Cloud provider handles key durability and availability

- **Disaster recovery**: Keys automatically replicated across availability zones

- **Access revocation**: Disable key access instantly without redeploying

- **Multi-region**: Use same key across regions (with multi-region keys)

### Migration Impact

| Aspect | Impact | Downtime Required? |
|--------|--------|-------------------|
| **KEK rotation** | New KEK created and encrypted with KMS key; old KEKs remain for backward compatibility | No |
| **Secret data** | No changes (secrets encrypted with KEKs, not master key directly) | No |
| **Application restart** | Required to load new KMS configuration | Yes (rolling restart) |
| **Configuration changes** | Add `KMS_PROVIDER` and `KMS_KEY_URI` env vars, update `MASTER_KEYS` | Yes |
| **Backup compatibility** | Old backups require old master keys in `MASTER_KEYS` to restore | N/A |

**Downtime estimate**: 5-10 minutes (rolling restart)

### Supported KMS Providers

- **AWS KMS**: `aws-kms` (recommended for AWS deployments)

- **GCP Cloud KMS**: `gcp-kms` (recommended for GCP deployments)

- **Azure Key Vault**: `azure-keyvault` (recommended for Azure deployments)

## Migration Planning

### Prerequisites

1. **Cloud KMS access**:

   - AWS: IAM role/user with `kms:Decrypt`, `kms:Encrypt`, `kms:GenerateDataKey` permissions

   - GCP: Service account with `cloudkms.cryptoKeyVersions.useToEncrypt` and `cloudkms.cryptoKeyVersions.useToDecrypt` roles

   - Azure: Managed identity or service principal with `Key Vault Crypto User` role

2. **Plaintext master key backup**:

   ```bash
   # Backup current master key (encrypted)
   echo $MASTER_KEY_PLAINTEXT | gpg --encrypt --recipient ops@example.com \
     > master-key-plaintext-backup-$(date +%Y%m%d).txt.gpg
   ```

3. **Database backup**:

   ```bash
   # Full backup before migration
   pg_dump --host=localhost --username=secrets --dbname=secrets \
     --format=custom --compress=9 \
     --file=secrets-pre-kms-migration-$(date +%Y%m%d).dump
   ```

4. **Maintenance window** (optional):

   - Schedule migration during low-traffic period

   - Or use rolling restart (no downtime)

### Migration Timeline

| Phase | Duration | Description |
|-------|----------|-------------|
| **Planning** | 1-2 hours | Create KMS keys, configure IAM, test access |
| **Backup** | 15-30 minutes | Backup database and existing master key configuration |
| **KMS Setup** | 30-60 minutes | Create KMS keys, configure policies, test encryption |
| **Migration** | 5-10 minutes | Generate new master key config, update env vars, restart app |
| **KEK Rotation** | < 1 minute | Create new KEK with `rotate-kek` command |
| **Validation** | 15-30 minutes | Test secret operations, verify KEK rotation |
| **Total** | 2-4 hours | End-to-end migration |

## Pre-Migration Checklist

- [ ] **KMS key created** (see [KMS Setup Guide](setup.md))

- [ ] **IAM permissions configured** (application can encrypt/decrypt with KMS key)

- [ ] **Plaintext master key backed up** (encrypted with GPG)

- [ ] **Database backed up** (full backup before migration)

- [ ] **Test environment migration completed** (validate procedure)

- [ ] **Rollback plan documented** (see [Rollback Plan](#rollback-plan))

- [ ] **Team trained** (SRE team aware of migration steps)

- [ ] **Monitoring enabled** (alerts for KMS errors)

## Migration Procedures

### Step 1: Create KMS Key

**AWS KMS**:

```bash
# Create KMS key
aws kms create-key \
  --description "Secrets master key (production)" \
  --key-usage ENCRYPT_DECRYPT \
  --origin AWS_KMS \
  --multi-region false

# Create alias
aws kms create-alias \
  --alias-name alias/secrets-master-key \
  --target-key-id <key-id-from-previous-command>

# Get key ARN
aws kms describe-key --key-id alias/secrets-master-key \
  --query 'KeyMetadata.Arn' --output text
# Output: arn:aws:kms:us-east-1:123456789012:key/abc-def-123

```

**GCP Cloud KMS**:

```bash
# Create keyring
gcloud kms keyrings create secrets \
  --location=us-east1

# Create key
gcloud kms keys create master-key \
  --location=us-east1 \
  --keyring=secrets \
  --purpose=encryption

# Get key ID
gcloud kms keys describe master-key \
  --location=us-east1 --keyring=secrets \
  --format='value(name)'
# Output: projects/my-project/locations/us-east1/keyRings/secrets/cryptoKeys/master-key

```

**Azure Key Vault**:

```bash
# Create key vault
az keyvault create \
  --name secrets-kv-prod \
  --resource-group secrets-rg \
  --location eastus

# Create key
az keyvault key create \
  --vault-name secrets-kv-prod \
  --name master-key \
  --protection software

# Get key ID
az keyvault key show \
  --vault-name secrets-kv-prod \
  --name master-key \
  --query 'key.kid' --output tsv
# Output: https://secrets-kv-prod.vault.azure.net/keys/master-key/abc123

```

### Step 2: Configure IAM Permissions

**AWS KMS**:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "kms:Decrypt",
        "kms:Encrypt",
        "kms:GenerateDataKey",
        "kms:DescribeKey"
      ],
      "Resource": "arn:aws:kms:us-east-1:123456789012:key/abc-def-123"
    }
  ]
}

```

Attach policy to IAM role/user used by Secrets application.

**GCP Cloud KMS**:

```bash
# Grant service account access to key
gcloud kms keys add-iam-policy-binding master-key \
  --location=us-east1 --keyring=secrets \
  --member='serviceAccount:secrets@my-project.iam.gserviceaccount.com' \
  --role='roles/cloudkms.cryptoKeyEncrypterDecrypter'

```

**Azure Key Vault**:

```bash
# Assign managed identity to Key Vault
az role assignment create \
  --assignee <managed-identity-principal-id> \
  --role "Key Vault Crypto User" \
  --scope /subscriptions/<sub-id>/resourceGroups/secrets-rg/providers/Microsoft.KeyVault/vaults/secrets-kv-prod

```

### Step 3: Test KMS Access

**AWS KMS**:

```bash
# Test encryption
echo "test data" | aws kms encrypt \
  --key-id alias/secrets-master-key \
  --plaintext fileb:///dev/stdin \
  --query CiphertextBlob --output text

# Test decryption
aws kms decrypt \
  --ciphertext-blob fileb://<(echo <ciphertext-from-previous-command> | base64 -d) \
  --query Plaintext --output text | base64 -d
# Should output: test data

```

**GCP Cloud KMS**:

```bash
# Test encryption
echo "test data" | gcloud kms encrypt \
  --location=us-east1 --keyring=secrets --key=master-key \
  --plaintext-file=- --ciphertext-file=/tmp/ciphertext

# Test decryption
gcloud kms decrypt \
  --location=us-east1 --keyring=secrets --key=master-key \
  --ciphertext-file=/tmp/ciphertext --plaintext-file=-
# Should output: test data

```

### Step 4: Generate New Master Key Configuration

This command generates a new master key encrypted with KMS and outputs the configuration needed to update your environment variables. **It does NOT modify your database or rotate KEKs** - those steps come later.

**Set KMS environment variables** (required for command to run in KMS mode):

```bash
export KMS_PROVIDER=aws-kms
export KMS_KEY_URI=arn:aws:kms:us-east-1:123456789012:key/abc-def-123

# Also set existing MASTER_KEYS (required by the command)
export MASTER_KEYS=<your-current-master-keys>
export ACTIVE_MASTER_KEY_ID=<your-current-active-key-id>
```

**Run master key rotation**:

```bash
./bin/app rotate-master-key
```

**Expected output**:

```text
# KMS Mode: Encrypting new master key with KMS
# KMS Provider: aws-kms

# Master Key Rotation (KMS Mode)
# Update these environment variables in your .env file or secrets manager

KMS_PROVIDER="aws-kms"
KMS_KEY_URI="arn:aws:kms:us-east-1:123456789012:key/abc-def-123"
MASTER_KEYS="<old-master-keys>,master-key-2026-02-21:<kms-encrypted-key-base64>"
ACTIVE_MASTER_KEY_ID="master-key-2026-02-21"

# Rotation Workflow:
# 1. Update the above environment variables
# 2. Restart the application
# 3. Rotate KEKs: app rotate-kek --algorithm aes-gcm
# 4. After all KEKs rotated, remove old master key: MASTER_KEYS="master-key-2026-02-21:<key>"
```

**IMPORTANT**:

- Copy the `MASTER_KEYS` and `ACTIVE_MASTER_KEY_ID` values - you'll need them in the next step
- The new master key is encrypted with KMS and appended to your existing `MASTER_KEYS`
- Both old and new master keys will be available during the transition (for backward compatibility)

**Duration**: < 5 seconds (cryptographic operation only)

### Step 5: Update Application Configuration

Update your application's environment configuration with the values from Step 4.

**Docker / Docker Compose** (`.env` file):

```bash
# Update .env file with new values from Step 4
# Replace the MASTER_KEYS and ACTIVE_MASTER_KEY_ID lines with the output from Step 4
nano .env

# Add or update these lines:
KMS_PROVIDER=aws-kms
KMS_KEY_URI=arn:aws:kms:us-east-1:123456789012:key/abc-def-123
MASTER_KEYS=<value-from-step-4>
ACTIVE_MASTER_KEY_ID=<value-from-step-4>

# Remove old plaintext-only configuration if present:
# MASTER_KEY_PROVIDER=plaintext
# MASTER_KEY_PLAINTEXT=xxx
```

**Kubernetes** (update ConfigMap/Secret):

```bash
kubectl edit configmap secrets-config -n production
# Add or update:
#   KMS_PROVIDER: "aws-kms"
#   KMS_KEY_URI: "arn:aws:kms:us-east-1:123456789012:key/abc-def-123"
#   MASTER_KEYS: "<value-from-step-4>"
#   ACTIVE_MASTER_KEY_ID: "<value-from-step-4>"
```

**Systemd** (`/etc/secrets/config.env`):

```bash
# Update /etc/secrets/config.env
sudo nano /etc/secrets/config.env

# Add new lines:
KMS_PROVIDER=aws-kms
KMS_KEY_URI=arn:aws:kms:us-east-1:123456789012:key/abc-def-123
MASTER_KEYS=<value-from-step-4>
ACTIVE_MASTER_KEY_ID=<value-from-step-4>

# Remove old lines if present:
# MASTER_KEY_PROVIDER=plaintext
# MASTER_KEY_PLAINTEXT=xxx
```

### Step 6: Restart Application

Restart the application to load the new KMS master key chain.

**Docker Compose**:

```bash
docker-compose restart secrets
```

**Kubernetes** (rolling restart):

```bash
kubectl rollout restart deployment/secrets -n production
kubectl rollout status deployment/secrets -n production
```

**Systemd**:

```bash
sudo systemctl restart secrets
```

**Verify application health**:

```bash
# Health checks
curl http://localhost:8080/health
# Expected: {"status":"healthy"}

curl http://localhost:8080/ready
# Expected: {"status":"ready"}

# Check logs for KMS initialization
docker-compose logs secrets | grep -i "master key"
# Should see: "master key chain initialized" with active key ID

# Or for Kubernetes
kubectl logs -n production deployment/secrets | grep -i "master key"

# Or for systemd
journalctl -u secrets -n 50 | grep -i "master key"
```

### Step 7: Rotate KEK to Use New Master Key

Create a new Key Encryption Key (KEK) that will be encrypted with the new KMS master key. This new KEK will be used to encrypt all new secrets going forward.

**IMPORTANT**: Old KEKs (encrypted with the old plaintext master key) remain in the database for backward compatibility. They are still used to decrypt existing secrets.

**Run KEK rotation**:

```bash
./bin/app rotate-kek --algorithm aes-gcm
```

**Expected output**:

```json
{"level":"INFO","msg":"rotating KEK","algorithm":"aes-gcm"}
{"level":"INFO","msg":"master key chain loaded","active_master_key_id":"master-key-2026-02-21"}
{"level":"INFO","msg":"KEK rotated successfully","algorithm":"aes-gcm","master_key_id":"master-key-2026-02-21"}
```

**What this does**:

- Creates a new KEK with `version = <current_version> + 1`
- Encrypts the new KEK using the active KMS master key (`master-key-2026-02-21`)
- Marks the new KEK as active (used for all new secret encryption operations)
- Old KEKs remain accessible for decrypting existing secrets

**Duration**: < 5 seconds (single database transaction)

### Step 8: Verify KEK Rotation

Confirm the new KEK was created and is using the KMS master key.

**Check KEK versions in database**:

```sql
-- Verify new KEK was created
SELECT id, version, algorithm, created_at 
FROM key_encryption_keys 
ORDER BY version DESC 
LIMIT 5;

-- Expected: New KEK with highest version number and recent created_at timestamp
-- Example output:
--   id                                    | version | algorithm | created_at
--  --------------------------------------+---------+-----------+----------------------------
--   550e8400-e29b-41d4-a716-446655440002 |       2 | aes-gcm   | 2026-02-21 14:30:15.123456
--   550e8400-e29b-41d4-a716-446655440001 |       1 | aes-gcm   | 2026-01-15 10:00:00.000000
```

**Understanding KEK versions**:

- **Old KEKs** (lower version numbers): Encrypted with old plaintext master key, used to decrypt existing secrets
- **New KEK** (highest version): Encrypted with new KMS master key, used to encrypt all new secrets
- Both co-exist for backward compatibility

## Validation

### Verify KEK Versions

Check that a new KEK was created with the latest version:

```sql
-- List all KEKs ordered by version (latest first)
SELECT id, version, algorithm, created_at, updated_at 
FROM key_encryption_keys 
ORDER BY version DESC;

-- Expected: Multiple KEKs with different versions
-- The highest version KEK should have a recent created_at timestamp (from Step 7)

-- Count total KEKs
SELECT COUNT(*) FROM key_encryption_keys;
```

### Test Secret Operations

**Create new secret** (will use new KMS-encrypted KEK):

```bash
# Get auth token
TOKEN=$(curl -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{"client_id":"xxx","client_secret":"yyy"}' | jq -r .token)

# Create new secret (encrypted with latest KEK version = KMS master key)
curl -X POST http://localhost:8080/v1/secrets \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"test-kms-migration","value":"test data"}'

# Retrieve new secret
curl -X GET http://localhost:8080/v1/secrets/test-kms-migration \
  -H "Authorization: Bearer $TOKEN" | jq .value
# Expected: "test data"
```

**Retrieve old secret** (encrypted with old KEK, still works):

```bash
# Old secrets use old KEK version (encrypted with old plaintext master key)
# Application can still decrypt because MASTER_KEYS contains both old and new keys
curl -X GET http://localhost:8080/v1/secrets/old-secret \
  -H "Authorization: Bearer $TOKEN" | jq .value
# Expected: Should decrypt successfully
```

### Verify KMS Usage in Logs

**AWS CloudTrail**:

```bash
# Check CloudTrail for Decrypt operations
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=ResourceName,AttributeValue=abc-def-123 \
  --max-results 10

```

**GCP Cloud Audit Logs**:

```bash
gcloud logging read \
  'protoPayload.serviceName="cloudkms.googleapis.com"' \
  --limit=10 --format=json

```

## Rollback Plan

### When to Rollback

Rollback if:

- KEK rotation fails or new KEK cannot be created

- Application fails to start with KMS configuration

- KMS access denied errors

- Unacceptable performance degradation

### Rollback Procedure

**Step 1: Restore previous configuration**:

```bash
# Docker Compose: Update .env file to remove KMS configuration
nano .env

# Restore original MASTER_KEYS and ACTIVE_MASTER_KEY_ID (from before Step 4)
MASTER_KEYS=<original-master-keys>
ACTIVE_MASTER_KEY_ID=<original-active-key-id>

# Remove KMS configuration
# KMS_PROVIDER=aws-kms
# KMS_KEY_URI=arn:aws:kms:...

# Or for Kubernetes
kubectl edit configmap secrets-config -n production
# Restore original MASTER_KEYS and ACTIVE_MASTER_KEY_ID
# Remove KMS_PROVIDER and KMS_KEY_URI

# Or for systemd
sudo nano /etc/secrets/config.env
# Restore original values
```

**Step 2: Restart application**:

```bash
# Docker Compose
docker-compose restart secrets

# Kubernetes
kubectl rollout restart deployment/secrets -n production

# Systemd
sudo systemctl restart secrets
```

**Step 3: Verify rollback**:

```bash
# Health checks
curl http://localhost:8080/health
# Expected: {"status":"healthy"}

curl http://localhost:8080/ready
# Expected: {"status":"ready"}

# Test secret retrieval
curl -X GET http://localhost:8080/v1/secrets/old-secret \
  -H "Authorization: Bearer $TOKEN"
# Expected: Should decrypt successfully
```

**Step 4: Rotate KEK back to plaintext master key** (optional, only if Step 7 was completed):

⚠️ **IMPORTANT**: This step is OPTIONAL. If you completed Step 7 (rotated KEK to KMS), you'll have KEKs encrypted with both old plaintext and new KMS keys.

To fully revert to plaintext-only (and create a new KEK encrypted with plaintext master key):

```bash
# Ensure KMS environment variables are NOT set
unset KMS_PROVIDER
unset KMS_KEY_URI

# Verify MASTER_KEYS is set to original plaintext keys
echo $MASTER_KEYS
echo $ACTIVE_MASTER_KEY_ID

# Rotate KEK to create new KEK encrypted with plaintext master key
./bin/app rotate-kek --algorithm aes-gcm
```

This creates a new KEK encrypted with the plaintext master key. Old secrets encrypted with the KMS-based KEK can still be decrypted (because `MASTER_KEYS` includes both the plaintext and KMS-encrypted master keys).

**NOTE**: If KEK rotation (Step 7) was never completed, rollback does NOT require this step. Simply reverting the configuration (Steps 1-3) is sufficient.

## Post-Migration

### Immediate Actions

1. **⚠️ DO NOT delete old master key from `MASTER_KEYS` yet**:

   The old master key is still needed for:
   - Decrypting old KEKs (which decrypt existing secrets)
   - Restoring old database backups
   - Rollback capability

   **Wait at least 30 days** before considering removal (see "Within 1 Month" section)

2. **Verify backups work with KMS**:

   ```bash
   # Test restore in non-production environment
   pg_restore --host=test-db --dbname=secrets secrets-backup.dump
   
   # Start app with KMS config and verify secrets decrypt
   # Ensure MASTER_KEYS contains both old and new keys
   docker run --rm \
     -e KMS_PROVIDER=aws-kms \
     -e KMS_KEY_URI=arn:aws:kms:... \
     -e MASTER_KEYS="<old-and-new-keys>" \
     -e ACTIVE_MASTER_KEY_ID="<new-key-id>" \
     allisson/secrets:latest server
   ```

3. **Update runbooks**:

   - Disaster recovery procedures now require KMS access
   - Backup restore requires `MASTER_KEYS` with both old and new keys
   - Master key rotation uses `rotate-master-key` + `rotate-kek` workflow

4. **Document migration details**:

   Store these values in a secure location (password manager, secrets vault):
   - Old master key ID and configuration
   - New master key ID (from Step 4)
   - KMS key ARN/ID
   - Migration date
   - KEK version before and after migration

### Within 1 Week

1. **Security review**:

   - Verify IAM policies follow least privilege
   - Enable KMS key rotation (AWS: automatic annual rotation)
   - Review CloudTrail/Cloud Audit Logs for unexpected KMS usage

2. **Monitoring**:

   - Add alerts for KMS access denied errors
   - Monitor KMS request latency
   - Track KMS costs (AWS: $1/month per key + $0.03 per 10,000 requests)

3. **Documentation**:

   - Update architecture diagrams with KMS
   - Document KMS key ID in secure location
   - Update DR runbook with KMS recovery procedures

### Within 1 Month

1. **Compliance audit**:

   - Verify KMS setup meets compliance requirements
   - Generate audit report from CloudTrail/Cloud Audit Logs
   - Review key access policies with security team

2. **Performance review**:

   - Compare pre/post-migration latency
   - Review KMS throttling (AWS: 5,500 req/sec per key)
   - Optimize caching if needed

3. **Consider removing old master key** (optional, after 30+ days):

   After verifying all systems are stable:

   ```sql
   -- Check if any secrets are still using old KEK versions
   SELECT kek.version, COUNT(s.id) as secret_count
   FROM secrets s
   JOIN key_encryption_keys kek ON s.kek_id = kek.id
   GROUP BY kek.version
   ORDER BY kek.version DESC;
   
   -- If all secrets use the latest KEK version, you can consider removing old master key
   ```

   **⚠️ WARNING**: Only remove old master key if:
   - All secrets have been re-encrypted with new KEK (version = latest)
   - All database backups older than 30 days can be discarded
   - You have tested backup restore with only the new master key

   ```bash
   # Update MASTER_KEYS to only include new key
   nano .env
   # Change: MASTER_KEYS="old-key:xxx,new-key:yyy"
   # To:     MASTER_KEYS="new-key:yyy"
   
   # Restart application
   docker-compose restart secrets
   ```

## Troubleshooting

### KEK rotation fails with "master key not found"

**Error**:

```text
ERROR: failed to rotate KEK: master key not found: master-key-2026-02-21
```

**Cause**: Application restarted with new configuration, but `MASTER_KEYS` or `ACTIVE_MASTER_KEY_ID` not set correctly

**Solution**:

```bash
# Verify environment variables are set
docker-compose exec secrets env | grep MASTER_KEYS
docker-compose exec secrets env | grep ACTIVE_MASTER_KEY_ID

# Should match output from Step 4
# If missing, update .env file and restart

# For Docker Compose
docker-compose restart secrets

# For Kubernetes
kubectl get configmap secrets-config -n production -o yaml | grep MASTER_KEYS

# Retry KEK rotation after restart
./bin/app rotate-kek --algorithm aes-gcm
```

### Application fails to start with "KMS access denied"

**Error**:

```text
FATAL: failed to initialize KMS client: AccessDeniedException
```

**Cause**: IAM role/service account lacks permissions

**Solution**:

```bash
# AWS: Verify KMS key exists and permissions are correct
aws kms describe-key --key-id arn:aws:kms:us-east-1:123456789012:key/abc-def-123

# Check IAM policy attached to role
aws iam get-role-policy --role-name secrets-app-role --policy-name kms-access

# Attach policy to IAM role if missing
aws iam attach-role-policy \
  --role-name secrets-app-role \
  --policy-arn arn:aws:iam::123456789012:policy/kms-access

# GCP: Grant service account permissions
gcloud kms keys add-iam-policy-binding master-key \
  --location=us-east1 --keyring=secrets \
  --member='serviceAccount:secrets@my-project.iam.gserviceaccount.com' \
  --role='roles/cloudkms.cryptoKeyEncrypterDecrypter'

# Restart application after fixing permissions
docker-compose restart secrets
```

### Old secrets fail to decrypt after migration

**Symptoms**: Secrets created before migration return decryption errors

**Cause**: `MASTER_KEYS` doesn't include old master key, or old KEKs were accidentally deleted

**Solution**:

```bash
# Verify MASTER_KEYS contains both old and new master keys
docker-compose exec secrets env | grep MASTER_KEYS
# Should see: old-key-id:xxx,new-key-id:yyy

# Check KEK versions in database
```

```sql
SELECT id, version, created_at FROM key_encryption_keys ORDER BY version DESC;
```

```bash
# If MASTER_KEYS is missing old key, restore it
nano .env
# Update MASTER_KEYS to include both old and new keys (from Step 4 output)

# Restart application
docker-compose restart secrets
```

### KMS latency too high

**Symptoms**: API responses slow after migration

**Cause**: KMS decrypt calls add latency (~10-50ms per call)

**Solution**:

- Enable KEK caching (Secrets caches decrypted KEKs in memory by default)

- Use multi-region KMS keys for lower latency

- Review application logs for excessive KMS calls

### Migration completed but old master key still accessible

**Explanation**: This is expected and intentional. `MASTER_KEYS` contains BOTH the old plaintext master key AND the new KMS-encrypted master key. This allows:

- **Backward compatibility**: Old secrets encrypted with old KEKs can still be decrypted
- **Rollback capability**: You can revert to the old configuration if needed
- **Gradual transition**: Both keys co-exist during the migration period

**To remove old master key** (after migration stabilizes):

After 30+ days and confirming all secrets decrypt successfully:

```bash
# Update MASTER_KEYS to only include the new KMS-encrypted key
nano .env
# Change: MASTER_KEYS="old-key:xxx,new-key:yyy"
# To:     MASTER_KEYS="new-key:yyy"

# Restart application
docker-compose restart secrets
```

⚠️ **WARNING**: Do NOT remove the old master key if:

- You have database backups that rely on it
- Any secrets are still encrypted with old KEK versions
- Migration was completed less than 30 days ago

## See Also

- [KMS Setup Guide](setup.md) - Detailed KMS provider setup for AWS/GCP/Azure

- [Key Management Guide](key-management.md) - KEK lifecycle and best practices

- [Security Hardening Guide](../deployment/docker-hardened.md) - Master key security best practices

- [Backup and Restore Guide](../deployment/backup-restore.md) - Backup considerations with KMS

- [Disaster Recovery Runbook](../runbooks/disaster-recovery.md) - DR with KMS keys
