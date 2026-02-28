# KMS Setup Guide

> Last updated: 2026-02-28

This guide covers setting up Key Management Service (KMS) integration for encrypting master keys at rest. KMS mode is **required** in v0.19.0+ (breaking change).

## Table of Contents

- [Overview](#overview)

- [Quick Start (Local Development)](#quick-start-local-development)

- [Provider Setup](#provider-setup)

  - [Provider Quick Matrix](#provider-quick-matrix)

  - [Placeholders Legend](#placeholders-legend)

  - [Ciphertext Format Caveats](#ciphertext-format-caveats)

  - [Provider Preflight Validation](#provider-preflight-validation)

- [Runtime Injection Examples](#runtime-injection-examples)

- [Migrating Between KMS Providers](#migrating-between-kms-providers)

- [Key Rotation](#key-rotation)

## Overview

**KMS Mode** (required in v0.19.0+) encrypts master keys using external Key Management Services before storing them in environment variables. This provides:

- **Defense in Depth**: Master keys encrypted at rest, even if environment variables are compromised

- **Audit Trail**: KMS providers log all key access operations

- **Compliance**: Meets regulatory requirements for key management (e.g., PCI-DSS, HIPAA)

- **Centralized Management**: KMS keys managed separately from application secrets

For local development, use the `localsecrets` provider with `base64key://` URIs.

### Architecture

```text
Application Environment Variables
  ↓
MASTER_KEYS (KMS-encrypted ciphertext)
  ↓
KMS Decryption (at application startup)
  ↓
In-Memory Master Key Chain (plaintext)
  ↓
KEK Encryption/Decryption
  ↓
DEK Encryption/Decryption
  ↓
Data Encryption/Decryption

```

## Security Considerations

**KMS integration is critical infrastructure** - compromise of your KMS configuration leads to complete exposure of all encrypted data. Follow these security principles when deploying KMS.

### 🔒 Critical Security Requirements

#### 1. Never Use `base64key://` in Production

The `localsecrets` provider with `base64key://` embeds the encryption key directly in the `KMS_KEY_URI` environment variable.

```dotenv
# ❌ INSECURE - Development/testing only

KMS_PROVIDER=localsecrets
KMS_KEY_URI=base64key://smGbjm71Nxd1Ig5FS0wj9SlbzAIrnolCz9bQQ6uAhl4=

```

**Never use this in staging or production.** Instead, use cloud KMS providers:

```dotenv
# ✅ SECURE - Production (GCP KMS)

KMS_PROVIDER=gcpkms
KMS_KEY_URI=gcpkms://projects/my-prod-project/locations/us-central1/keyRings/secrets-keyring/cryptoKeys/master-key

# ✅ SECURE - Production (AWS KMS)

KMS_PROVIDER=awskms
KMS_KEY_URI=awskms:///alias/secrets-master-key

# ✅ SECURE - Production (Azure Key Vault)

KMS_PROVIDER=azurekeyvault
KMS_KEY_URI=azurekeyvault://my-prod-vault.vault.azure.net/keys/master-key

```

#### 2. Protect KMS_KEY_URI Like Passwords

The `KMS_KEY_URI` variable provides the path to decrypt all master keys. Treat it as a critical secret:

**Do:**

- ✅ Store in secrets manager (AWS Secrets Manager, GCP Secret Manager, Azure Key Vault, HashiCorp Vault)

- ✅ Use `.env` files excluded from git (`.env` is in `.gitignore`)

- ✅ Inject via CI/CD secrets for automated deployments

- ✅ Encrypt at rest in backups and disaster recovery systems

- ✅ Rotate KMS keys quarterly or per organizational policy

**Don't:**

- ❌ Commit to source control (even private repos)

- ❌ Store in plaintext configuration files

- ❌ Include in log output or error messages

- ❌ Share via email, Slack, or insecure channels

- ❌ Embed in Docker images or container layers

#### 3. Use Least Privilege IAM Permissions

Restrict KMS access to the minimum required permissions:

**Google Cloud KMS:**

```bash
# Grant ONLY encrypt/decrypt permissions (not admin)
gcloud kms keys add-iam-policy-binding master-key-encryption \
  --location=us-central1 \
  --keyring=secrets-keyring \
  --member="serviceAccount:secrets-app@project.iam.gserviceaccount.com" \
  --role="roles/cloudkms.cryptoKeyEncrypterDecrypter"

```

**AWS KMS:**

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "kms:Encrypt",
        "kms:Decrypt"
      ],
      "Resource": "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"
    }
  ]
}

```

**Azure Key Vault:**

```bash
# Grant ONLY encrypt/decrypt operations
az keyvault set-policy \
  --name secrets-kv-unique \
  --spn <app-service-principal> \
  --key-permissions encrypt decrypt

```

**HashiCorp Vault:**

```hcl
# Grant ONLY transit encrypt/decrypt
path "transit/encrypt/master-key-encryption" {
  capabilities = ["update"]
}

path "transit/decrypt/master-key-encryption" {
  capabilities = ["update"]
}

```

❌ **Do not grant**:

- Admin/owner permissions on KMS keys

- Key deletion permissions

- Key rotation permissions (unless specifically required for automation)

- Broad wildcard permissions (`kms:*`, `cloudkms.*`)

#### 4. Use Workload Identity / IAM Roles (Not Static Credentials)

Prefer cloud-native authentication over static credentials:

| Platform | Recommended Auth | Avoid |
|----------|------------------|-------|
| **GCP** | Workload Identity | Service account JSON keys |
| **AWS** | IAM Roles | IAM user access keys |
| **Azure** | Managed Identity | Service principal passwords |
| **HashiCorp Vault** | AppRole | Root tokens, long-lived tokens |

### Example: GCP Workload Identity

```bash
# Bind service account to GCP service account
gcloud iam service-accounts add-iam-policy-binding \
  secrets-kms-user@project.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:project.svc.id.goog[secrets/secrets-api]"

```

### Example: AWS IAM Roles

```bash
# Associate IAM role with application
aws iam create-role \
  --role-name SecretsKMSRole \
  --assume-role-policy-document file://trust-policy.json

aws iam attach-role-policy \
  --role-name SecretsKMSRole \
  --policy-arn arn:aws:iam::123456789012:policy/SecretsKMSPolicy

```

#### 5. Enable Audit Logging and Monitoring

Monitor KMS key access for security incidents:

**Google Cloud KMS:**

```bash
# Enable Cloud Audit Logs for KMS
gcloud logging read "protoPayload.serviceName=cloudkms.googleapis.com" --limit 10

```

**AWS KMS:**

```bash
# Enable CloudTrail for KMS (if not already enabled)
aws cloudtrail create-trail --name kms-audit --s3-bucket-name my-audit-bucket

# Query KMS events
aws cloudtrail lookup-events --lookup-attributes AttributeKey=ResourceType,AttributeValue=AWS::KMS::Key

```

**Azure Key Vault:**

```bash
# Enable diagnostic logs
az monitor diagnostic-settings create \
  --resource /subscriptions/<subscription-id>/resourceGroups/secrets-rg/providers/Microsoft.KeyVault/vaults/secrets-kv-unique \
  --name kms-audit \
  --logs '[{"category": "AuditEvent", "enabled": true}]' \
  --workspace <log-analytics-workspace-id>

```

**Alert on suspicious patterns:**

- Decrypt operations from unknown IPs or regions

- Failed authentication attempts

- Key access outside business hours

- Unusual spike in decrypt operations

#### 6. Implement Key Rotation

Rotate KMS keys regularly to limit exposure:

**Rotation frequency recommendations:**

- **High-security environments**: 90 days

- **Standard deployments**: 180 days

- **Low-risk environments**: 365 days

**Before rotating KMS keys**, ensure:

1. [ ] Old KMS key remains available for decrypting existing `MASTER_KEYS`
2. [ ] New KMS key created and permissions granted
3. [ ] Testing completed in staging environment
4. [ ] Rollback plan documented and tested

See [Key Rotation](#key-rotation) section below for detailed procedures.

#### 7. Backup and Disaster Recovery

**Backup strategy for KMS:**

- ✅ Document KMS key IDs/URIs in encrypted password manager

- ✅ Store KMS provider credentials in separate secrets manager

- ✅ Maintain offline encrypted backup of `MASTER_KEYS` ciphertext

- ✅ Test disaster recovery quarterly

**Disaster recovery checklist:**

- [ ] Can you recreate KMS keys from documented URIs?

- [ ] Can you restore `MASTER_KEYS` from backup?

- [ ] Can you authenticate to KMS provider (credential recovery process)?

- [ ] Can you decrypt at least one test secret end-to-end?

#### 8. Incident Response for KMS Compromise

If `KMS_KEY_URI` or KMS credentials are exposed:

**Immediate (within 1 hour):**

1. Revoke compromised credentials (service account keys, IAM access keys, tokens)
2. Disable or delete compromised KMS key (if supported by provider)
3. Create new KMS key with new credentials
4. Update incident log with timeline and exposure scope

**Within 24 hours:**

1. Generate new `MASTER_KEYS` using new KMS key
2. Deploy updated configuration to all environments
3. Rotate all KEKs using `rotate-kek` command
4. Audit database access logs during exposure window

**Within 1 week:**

1. Review and rotate all secrets that may have been accessed
2. Update runbooks with lessons learned
3. Implement additional controls (pre-commit hooks, automated secret scanning)
4. Conduct post-incident review with team

### Example: GCP KMS key rotation after compromise

```bash
# 1. Disable compromised key version
gcloud kms keys versions disable 1 \
  --key master-key-encryption \
  --keyring secrets-keyring \
  --location us-central1

# 2. Create new key version (automatic with GCP KMS)
gcloud kms keys update master-key-encryption \
  --keyring secrets-keyring \
  --location us-central1 \
  --default-algorithm google-symmetric-encryption

# 3. Generate new master key with new KMS key version
./bin/app create-master-key \
  --kms-provider=gcpkms \
  --kms-key-uri="gcpkms://projects/my-project/locations/us-central1/keyRings/secrets-keyring/cryptoKeys/master-key-encryption"

```

### Security Comparison: KMS Providers

| Provider | Security Level | Compliance Certifications | HSM Support | Cost (approx) |
|----------|---------------|---------------------------|-------------|---------------|
| `localsecrets` (`base64key://`) | ⚠️ Low (dev only) | None | No | Free |
| Google Cloud KMS | 🔒 High | SOC 2, ISO 27001, HIPAA | Yes (Cloud HSM) | ~$1/key/month + $0.03/10k ops |
| AWS KMS | 🔒 High | SOC 2, ISO 27001, PCI-DSS, HIPAA | Yes (CloudHSM) | ~$1/key/month + $0.03/10k ops |
| Azure Key Vault | 🔒 High | SOC 2, ISO 27001, HIPAA, FedRAMP | Yes (Premium tier) | ~$0.03/10k ops (Standard), ~$1/key/month (HSM) |
| HashiCorp Vault | 🔒 Medium-High | SOC 2 (Enterprise) | Yes (Enterprise) | Self-hosted or ~$0.03/hour (HCP) |

**Recommendations by environment:**

- **Production**: Cloud KMS (GCP/AWS/Azure) with HSM-backed keys

- **Staging**: Cloud KMS (standard tier acceptable)

- **Development**: `localsecrets` (`base64key://`) acceptable for local testing only

### Pre-Production Security Checklist

Before deploying KMS to production, verify:

**Configuration:**

- [ ] `KMS_PROVIDER` is NOT `localsecrets` (unless development)

- [ ] `KMS_KEY_URI` does NOT use `base64key://` (unless development)

- [ ] `KMS_KEY_URI` is stored in secrets manager, not committed to git

- [ ] `.env` file is in `.gitignore` and excluded from version control

**IAM/Permissions:**

- [ ] Service account/role has ONLY `encrypt` and `decrypt` permissions

- [ ] No admin or key management permissions granted

- [ ] Workload Identity / IAM Roles used instead of static credentials

- [ ] Credential rotation schedule documented (90-180 days)

**Monitoring:**

- [ ] KMS audit logging enabled (CloudTrail, Cloud Audit Logs, Azure Monitor)

- [ ] Alerts configured for failed decrypt attempts

- [ ] Alerts configured for unusual access patterns

- [ ] Monthly audit log review scheduled

**Disaster Recovery:**

- [ ] KMS key URIs documented in password manager

- [ ] `MASTER_KEYS` ciphertext backed up to encrypted storage

- [ ] Disaster recovery runbook tested in last 90 days

- [ ] Rollback plan documented and validated

**Incident Response:**

- [ ] KMS compromise incident response plan documented

- [ ] Rotation procedures tested in staging

- [ ] On-call team trained on KMS emergency procedures

- [ ] Post-incident review process defined

## Quick Start (Local Development)

For local testing without cloud KMS, use the `localsecrets` provider:

### 1. Generate a KMS Key

The KMS key is used to encrypt/decrypt master keys. Generate a 32-byte key:

```bash
# Generate random 32-byte key and encode as base64
openssl rand -base64 32
# Output: smGbjm71Nxd1Ig5FS0wj9SlbzAIrnolCz9bQQ6uAhl4=

```

**⚠️ Security**: Store this KMS key securely! In production, use cloud KMS instead of `localsecrets`.

### 2. Generate an Encrypted Master Key

```bash
./bin/app create-master-key \
  --kms-provider=localsecrets \
  --kms-key-uri="base64key://smGbjm71Nxd1Ig5FS0wj9SlbzAIrnolCz9bQQ6uAhl4="

```

Output:

```text
# KMS Mode: Encrypting master key with KMS
# KMS Provider: localsecrets

# Master Key Configuration (KMS Mode)
KMS_PROVIDER="localsecrets"
KMS_KEY_URI="base64key://smGbjm71Nxd1Ig5FS0wj9SlbzAIrnolCz9bQQ6uAhl4="
MASTER_KEYS="master-key-2026-02-27:ARiEeAASDiXKAxzOQCw2NxQfrHAc33CPP/7SsvuVjVvq1olzRBudplPoXRkquRWUXQ+CnEXi15LACqXuPGszLS+anJUrdn04"
ACTIVE_MASTER_KEY_ID="master-key-2026-02-27"

```

### 3. Configure Environment

Add to `.env`:

```bash
KMS_PROVIDER=localsecrets
KMS_KEY_URI=base64key://smGbjm71Nxd1Ig5FS0wj9SlbzAIrnolCz9bQQ6uAhl4=
MASTER_KEYS=master-key-2026-02-27:ARiEeAASDiXKAxzOQCw2NxQfrHAc33CPP/7SsvuVjVvq1olzRBudplPoXRkquRWUXQ+CnEXi15LACqXuPGszLS+anJUrdn04
ACTIVE_MASTER_KEY_ID=master-key-2026-02-27

```

### 4. Start the Application

```bash
./bin/app server

```

Check logs for successful KMS initialization:

```text
INFO KMS mode enabled provider=localsecrets
INFO master key decrypted via KMS key_id=master-key-2026-02-27
INFO master key chain loaded active_master_key_id=master-key-2026-02-27 total_keys=1

```

## Provider Setup

### Provider Quick Matrix

| Provider | URI format | Required auth | Minimum permission |
| --- | --- | --- | --- |
| `localsecrets` | `base64key://<base64-32-byte-key>` | none | local key only |
| `gcpkms` | `gcpkms://projects/<project>/locations/<location>/keyRings/<ring>/cryptoKeys/<key>` | `GOOGLE_APPLICATION_CREDENTIALS` | encrypt + decrypt |
| `awskms` | `awskms:///<key-id-or-alias>` | AWS SDK default chain (`AWS_ACCESS_KEY_ID`/role) | `kms:Encrypt`, `kms:Decrypt` |
| `azurekeyvault` | `azurekeyvault://<vault>.vault.azure.net/keys/<key>` | `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET` | key encrypt + decrypt |
| `hashivault` | `hashivault:///<transit-key-path>` | `VAULT_ADDR`, `VAULT_TOKEN` | transit encrypt + decrypt |

### Placeholders Legend

- `<provider>`: one of `localsecrets`, `gcpkms`, `awskms`, `azurekeyvault`, `hashivault`

- `<uri>`: provider-specific KMS URI shown in the matrix above

- `<generated-encrypted-key>`: full `ID:CIPHERTEXT` output from `create-master-key`

- `<kms-ciphertext-for-old-key>`: ciphertext produced by encrypting an existing legacy key with your KMS provider.

Treat placeholders as templates only; replace with exact runtime values before applying.

### Ciphertext Format Caveats

- `MASTER_KEYS` values in KMS mode must be ciphertext outputs from the selected provider.

- Do not assume provider outputs use the same encoding format:

  - Cloud KMS tooling often returns base64-like blobs.

  - Vault transit typically returns prefixed ciphertext (for example `vault:v1:...`).

- Keep each provider's ciphertext format as-is; do not transform to another format unless the

  provider documentation requires it.

- Never use plaintext values in `MASTER_KEYS` - as of v0.19.0, all values must be KMS-encrypted ciphertext.

### Provider Preflight Validation

Before rollout, validate credentials and permissions with a quick encrypt/decrypt round-trip.

Use an isolated temp folder and clean it up when done:

```bash
mkdir -p /tmp/secrets-kms-preflight

```

Google Cloud KMS:

```bash
printf 'kms-preflight' > /tmp/secrets-kms-preflight/input.txt
gcloud kms encrypt --project="$PROJECT_ID" --location="us-central1" --keyring="secrets-keyring" \
  --key="master-key-encryption" --plaintext-file="/tmp/secrets-kms-preflight/input.txt" \
  --ciphertext-file="/tmp/secrets-kms-preflight/cipher.bin"
gcloud kms decrypt --project="$PROJECT_ID" --location="us-central1" --keyring="secrets-keyring" \
  --key="master-key-encryption" --ciphertext-file="/tmp/secrets-kms-preflight/cipher.bin" \
  --plaintext-file="/tmp/secrets-kms-preflight/output.txt"
cmp /tmp/secrets-kms-preflight/input.txt /tmp/secrets-kms-preflight/output.txt

```

AWS KMS:

```bash
printf 'kms-preflight' > /tmp/secrets-kms-preflight/input.txt
CIPHERTEXT_B64="$(aws kms encrypt --key-id alias/secrets-master-key \
  --plaintext fileb:///tmp/secrets-kms-preflight/input.txt --query CiphertextBlob --output text)"
export CIPHERTEXT_B64

python3 - <<'PY'

import base64, os
data = base64.b64decode(os.environ["CIPHERTEXT_B64"])
open('/tmp/secrets-kms-preflight/cipher.bin', 'wb').write(data)
PY

DECRYPTED_B64="$(aws kms decrypt --ciphertext-blob fileb:///tmp/secrets-kms-preflight/cipher.bin \
  --query Plaintext --output text)"
export DECRYPTED_B64

python3 - <<'PY'

import base64, os
data = base64.b64decode(os.environ["DECRYPTED_B64"])
open('/tmp/secrets-kms-preflight/output.txt', 'wb').write(data)
PY

cmp /tmp/secrets-kms-preflight/input.txt /tmp/secrets-kms-preflight/output.txt

```

Azure Key Vault:

```bash
# Credential/permission preflight
az keyvault key show --vault-name secrets-kv-unique --name master-key-encryption

# Optional encrypt/decrypt smoke test (CLI/algorithm support may vary by key type)
az keyvault key encrypt --vault-name secrets-kv-unique --name master-key-encryption \
  --algorithm RSA-OAEP-256 --value "kms-preflight"

```

HashiCorp Vault Transit:

```bash
PLAINTEXT_B64="$(printf 'kms-preflight' | base64 | tr -d '\n')"
CIPHERTEXT="$(vault write -field=ciphertext transit/encrypt/master-key-encryption plaintext="$PLAINTEXT_B64")"
vault write -field=plaintext transit/decrypt/master-key-encryption ciphertext="$CIPHERTEXT" | \
  python3 -c 'import base64,sys;print(base64.b64decode(sys.stdin.read().strip()).decode(), end="")'

```

Cleanup:

```bash
rm -rf /tmp/secrets-kms-preflight

```

### Provider Authentication Requirements

Each provider requires specific environment variables or IAM roles to authenticate. See the official provider documentation for exact setup steps.

- **GCP**: Use Workload Identity or `GOOGLE_APPLICATION_CREDENTIALS`
- **AWS**: Use IAM Roles or `AWS_ACCESS_KEY_ID`+`AWS_SECRET_ACCESS_KEY`
- **Azure**: Use Managed Identities or `AZURE_TENANT_ID`+`AZURE_CLIENT_ID`+`AZURE_CLIENT_SECRET`
- **HashiCorp**: `VAULT_ADDR` and `VAULT_TOKEN`

## Runtime Injection Examples

Prefer secrets managers/orchestrator secrets over inline plaintext in deployment manifests.

Docker Compose example:

```yaml
services:
  secrets-api:
    image: allisson/secrets
    env_file:
      - .env

    environment:
      KMS_PROVIDER: gcpkms
      KMS_KEY_URI: gcpkms://projects/my-project/locations/us-central1/keyRings/secrets/cryptoKeys/master-key
      MASTER_KEYS: ${MASTER_KEYS}
      ACTIVE_MASTER_KEY_ID: ${ACTIVE_MASTER_KEY_ID}

```

## Migrating Between KMS Providers

To migrate master keys from one KMS provider to another (e.g., from `localsecrets` to `gcpkms` for production, or between cloud providers):

### Step 1: Set Up New KMS Provider

Follow provider-specific setup instructions above for your target KMS provider.

### Step 2: Generate New KMS-Encrypted Master Key

```bash
./bin/app create-master-key \
  --id=master-key-new-provider-2026 \
  --kms-provider=<new-provider> \
  --kms-key-uri=<new-provider-uri>

```

### Step 3: Re-encrypt Existing Master Keys with New Provider

You cannot mix master keys encrypted with different KMS providers in `MASTER_KEYS`. All entries must use the same `KMS_PROVIDER` and `KMS_KEY_URI`.

To re-encrypt an existing master key with a new provider:

#### Step 3a: Decrypt Existing Master Key Using Old Provider

First, extract the existing encrypted master key from your current `MASTER_KEYS`:

```bash
# Example: old-key:ARiEeAASDiXKAxzOQCw2NxQ...
OLD_KEY_ID="old-key"
OLD_KEY_CIPHERTEXT="ARiEeAASDiXKAxzOQCw2NxQ..."
```

Manually decrypt using the old KMS provider (requires old `KMS_KEY_URI` access):

**For localsecrets:**

```bash
# Decrypt using provider-specific CLI or equivalent tooling
# (Application-specific decryption required)
```

**For cloud providers:**
Use provider CLI to decrypt the ciphertext and obtain the 32-byte plaintext master key.

#### Step 3b: Encrypt Plaintext Key With New Provider

Once you have the plaintext 32-byte master key:

```bash
# Save plaintext key to temporary file
printf '%s' "$OLD_KEY_PLAINTEXT_B64" | base64 --decode > /tmp/old-master-key.bin
```

Google Cloud KMS:

```bash
gcloud kms encrypt \
  --project="my-gcp-project" \
  --location="us-central1" \
  --keyring="secrets-keyring" \
  --key="master-key-encryption" \
  --plaintext-file="/tmp/old-master-key.bin" \
  --ciphertext-file="/tmp/old-master-key.cipher"

OLD_KEY_NEW_KMS_CIPHERTEXT="$(base64 < /tmp/old-master-key.cipher | tr -d '\n')"
```

AWS KMS:

```bash
OLD_KEY_NEW_KMS_CIPHERTEXT="$(aws kms encrypt \
  --key-id alias/secrets-master-key \
  --plaintext fileb:///tmp/old-master-key.bin \
  --query CiphertextBlob \
  --output text)"
```

Azure Key Vault:

```bash
OLD_KEY_NEW_KMS_CIPHERTEXT="$(az keyvault key encrypt \
  --vault-name secrets-kv-unique \
  --name master-key-encryption \
  --algorithm RSA-OAEP-256 \
  --file /tmp/old-master-key.bin \
  --query result \
  --output tsv)"
```

HashiCorp Vault Transit:

```bash
OLD_KEY_PLAINTEXT_B64="$(cat /tmp/old-master-key.bin | base64 | tr -d '\n')"
OLD_KEY_NEW_KMS_CIPHERTEXT="$(vault write -field=ciphertext transit/encrypt/master-key-encryption \
  plaintext="$OLD_KEY_PLAINTEXT_B64")"
```

Clean up temporary files:

```bash
rm -f /tmp/old-master-key.bin /tmp/old-master-key.cipher
```

### Step 4: Update Environment with New Provider

Update environment with new KMS provider and re-encrypted master keys:

```bash
KMS_PROVIDER=<new-provider>
KMS_KEY_URI=<new-provider-uri>
MASTER_KEYS=old-key:<re-encrypted-with-new-provider>,master-key-new-provider-2026:<new-key-ciphertext>
ACTIVE_MASTER_KEY_ID=old-key
```

### Step 5: Restart Application

Verify both keys are loaded with the new provider:

```text
INFO KMS mode enabled provider=gcpkms
INFO master key decrypted via KMS key_id=old-key
INFO master key decrypted via KMS key_id=master-key-new-provider-2026
INFO master key chain loaded active_master_key_id=old-key total_keys=2
```

### Step 6: Rotate KEKs to New Master Key

```bash
# Switch active master key to new provider's key
export ACTIVE_MASTER_KEY_ID=master-key-new-provider-2026

# Restart application
./bin/app server

# Rotate all KEKs (re-encrypts with new master key)
./bin/app rotate-kek --algorithm aes-gcm
```

### Step 7: Remove Old Master Key

After verifying all KEKs are encrypted with the new master key:

```bash
# Remove old key from MASTER_KEYS
MASTER_KEYS=master-key-new-provider-2026:<kms-encrypted-ciphertext>
ACTIVE_MASTER_KEY_ID=master-key-new-provider-2026
```

### Migration Checklist

Use this checklist for migrating between KMS providers.

#### 1) Precheck

- [ ] Confirm target provider credentials and permissions are available
- [ ] Back up current environment configuration
- [ ] Confirm rollback owner and change window
- [ ] Confirm new KMS provider encrypt/decrypt permissions are granted
- [ ] Test new KMS provider connectivity (preflight validation)

#### 2) Build New KMS Key Chain

- [ ] Generate new master key with `create-master-key --kms-provider <new> --kms-key-uri <new-uri>`
- [ ] Decrypt existing master keys using old provider
- [ ] Re-encrypt existing master keys using new provider
- [ ] Build `MASTER_KEYS` with all keys encrypted by new provider
- [ ] Set `KMS_PROVIDER`, `KMS_KEY_URI`, and `ACTIVE_MASTER_KEY_ID`

#### 3) Rollout

- [ ] Restart API instances (rolling)
- [ ] Verify startup logs show new KMS provider and key decrypt lines
- [ ] Run baseline checks: `GET /health`, `GET /ready`
- [ ] Run key-dependent smoke checks: token issuance, secrets, transit

Reference: [Production rollout golden path](../deployment/production-rollout.md)

#### 4) Rotation and Cleanup

- [ ] Rotate KEK after switching to new KMS provider
- [ ] Verify reads/decrypt for existing data still succeed
- [ ] Remove old key entries from `MASTER_KEYS` only after verification
- [ ] Restart API instances again after key-chain cleanup

Reference: [Key management operations](../kms/key-management.md)

#### 5) Rollback Readiness

- [ ] Keep previous image tag available
- [ ] Keep pre-change env snapshot (including old KMS credentials)
- [ ] If rollback needed, revert app version first
- [ ] Re-validate health and smoke checks after rollback

Reference: [Release notes](../../releases/RELEASES.md)

## Key Rotation

Rotate master keys regularly (recommended: every 90-180 days).

### Generate New Master Key

```bash
./bin/app rotate-master-key --id=master-key-2026-02-27 \
  --kms-provider=gcpkms \
  --kms-key-uri="gcpkms://..."

```

Output includes combined configuration:

```bash
KMS_PROVIDER=gcpkms
KMS_KEY_URI=gcpkms://...
MASTER_KEYS=old-key:<old-ciphertext>,master-key-2026-02-27:<new-ciphertext>
ACTIVE_MASTER_KEY_ID=master-key-2026-02-27

```

### Rotation Workflow

```bash
# 1. Update environment variables with output from rotate-master-key
# 2. Restart application
./bin/app server

# 3. Verify both keys loaded
# Logs should show: total_keys=2

# 4. Rotate all KEKs to use new master key
./bin/app rotate-kek --algorithm aes-gcm

# 5. After KEK rotation complete, remove old master key
MASTER_KEYS=master-key-2026-02-27:<new-ciphertext>
ACTIVE_MASTER_KEY_ID=master-key-2026-02-27

# 6. Restart application
./bin/app server

```
