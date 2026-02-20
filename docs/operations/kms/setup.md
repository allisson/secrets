# KMS Setup Guide

> Last updated: 2026-02-20

This guide covers setting up Key Management Service (KMS) integration for encrypting master keys at rest. KMS mode provides an additional security layer by ensuring master keys are never stored in plaintext.

## Table of Contents

- [Overview](#overview)
- [Quick Start (Local Development)](#quick-start-local-development)
- [Provider Setup](#provider-setup)
  - [Provider Quick Matrix](#provider-quick-matrix)
  - [Placeholders Legend](#placeholders-legend)
  - [Ciphertext Format Caveats](#ciphertext-format-caveats)
  - [Provider Preflight Validation](#provider-preflight-validation)
  - [Google Cloud KMS](#google-cloud-kms)
  - [AWS KMS](#aws-kms)
  - [Azure Key Vault](#azure-key-vault)
  - [HashiCorp Vault](#hashicorp-vault)
- [Runtime Injection Examples](#runtime-injection-examples)
- [Migration from Legacy Mode](#migration-from-legacy-mode)
- [Key Rotation](#key-rotation)
- [Troubleshooting](#troubleshooting)

## Overview

**KMS Mode** encrypts master keys using external Key Management Services before storing them in environment variables. This provides:

- **Defense in Depth**: Master keys encrypted at rest, even if environment variables are compromised
- **Audit Trail**: KMS providers log all key access operations
- **Compliance**: Meets regulatory requirements for key management (e.g., PCI-DSS, HIPAA)
- **Centralized Management**: KMS keys managed separately from application secrets

**Legacy Mode** stores master keys as plaintext base64-encoded values. This is **only suitable for development and testing**.

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
MASTER_KEYS="master-key-2026-02-19:ARiEeAASDiXKAxzOQCw2NxQfrHAc33CPP/7SsvuVjVvq1olzRBudplPoXRkquRWUXQ+CnEXi15LACqXuPGszLS+anJUrdn04"
ACTIVE_MASTER_KEY_ID="master-key-2026-02-19"
```

### 3. Configure Environment

Add to `.env`:

```bash
KMS_PROVIDER=localsecrets
KMS_KEY_URI=base64key://smGbjm71Nxd1Ig5FS0wj9SlbzAIrnolCz9bQQ6uAhl4=
MASTER_KEYS=master-key-2026-02-19:ARiEeAASDiXKAxzOQCw2NxQfrHAc33CPP/7SsvuVjVvq1olzRBudplPoXRkquRWUXQ+CnEXi15LACqXuPGszLS+anJUrdn04
ACTIVE_MASTER_KEY_ID=master-key-2026-02-19
```

### 4. Start the Application

```bash
./bin/app server
```

Check logs for successful KMS initialization:

```text
INFO KMS mode enabled provider=localsecrets
INFO master key decrypted via KMS key_id=master-key-2026-02-19
INFO master key chain loaded active_master_key_id=master-key-2026-02-19 total_keys=1
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
- `<generated-encrypted-key>`: full `id:ciphertext` output from `create-master-key`
- `<kms-ciphertext-for-old-key>`: ciphertext produced by encrypting an existing legacy key with your KMS

Treat placeholders as templates only; replace with exact runtime values before applying.

### Ciphertext Format Caveats

- `MASTER_KEYS` values in KMS mode must be ciphertext outputs from the selected provider.
- Do not assume provider outputs use the same encoding format:
  - Cloud KMS tooling often returns base64-like blobs.
  - Vault transit typically returns prefixed ciphertext (for example `vault:v1:...`).
- Keep each provider's ciphertext format as-is; do not transform to another format unless the
  provider documentation requires it.
- Never mix plaintext legacy values and KMS ciphertext values in `MASTER_KEYS` when KMS mode is enabled.

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

### Google Cloud KMS

#### GCP Prerequisites

1. **GCP Project**: Active Google Cloud project with billing enabled
2. **API Enabled**: Enable Cloud KMS API
3. **Credentials**: Service account with `cloudkms.cryptoKeyVersions.useToEncrypt` and `cloudkms.cryptoKeyVersions.useToDecrypt` permissions

#### GCP Setup Steps

```bash
# 1. Set project ID
export PROJECT_ID="my-gcp-project"

# 2. Enable Cloud KMS API
gcloud services enable cloudkms.googleapis.com --project=$PROJECT_ID

# 3. Create key ring
gcloud kms keyrings create secrets-keyring \
  --location=us-central1 \
  --project=$PROJECT_ID

# 4. Create crypto key
gcloud kms keys create master-key-encryption \
  --location=us-central1 \
  --keyring=secrets-keyring \
  --purpose=encryption \
  --project=$PROJECT_ID

# 5. Create service account
gcloud iam service-accounts create secrets-kms-user \
  --display-name="Secrets KMS User" \
  --project=$PROJECT_ID

# 6. Grant permissions
gcloud kms keys add-iam-policy-binding master-key-encryption \
  --location=us-central1 \
  --keyring=secrets-keyring \
  --member="serviceAccount:secrets-kms-user@$PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/cloudkms.cryptoKeyEncrypterDecrypter" \
  --project=$PROJECT_ID

# 7. Generate service account key
gcloud iam service-accounts keys create gcp-kms-key.json \
  --iam-account=secrets-kms-user@$PROJECT_ID.iam.gserviceaccount.com \
  --project=$PROJECT_ID
```

#### GCP Generate Encrypted Master Key

```bash
# Set GCP credentials
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/gcp-kms-key.json"

# Generate encrypted master key
./bin/app create-master-key \
  --kms-provider=gcpkms \
  --kms-key-uri="gcpkms://projects/$PROJECT_ID/locations/us-central1/keyRings/secrets-keyring/cryptoKeys/master-key-encryption"
```

#### GCP Environment Configuration

```bash
# Application environment
GOOGLE_APPLICATION_CREDENTIALS=/path/to/gcp-kms-key.json
KMS_PROVIDER=gcpkms
KMS_KEY_URI=gcpkms://projects/my-gcp-project/locations/us-central1/keyRings/secrets-keyring/cryptoKeys/master-key-encryption
MASTER_KEYS=<generated-encrypted-key>
ACTIVE_MASTER_KEY_ID=<key-id>
```

### AWS KMS

#### AWS Prerequisites

1. **AWS Account**: Active AWS account with appropriate permissions
2. **IAM User/Role**: With `kms:Encrypt` and `kms:Decrypt` permissions
3. **AWS Credentials**: Configured via AWS CLI or environment variables

#### AWS Setup Steps

```bash
# 1. Create KMS key
aws kms create-key \
  --description "Secrets Master Key Encryption" \
  --key-usage ENCRYPT_DECRYPT \
  --origin AWS_KMS \
  --region us-east-1

# Output: Copy the KeyId from response
# Example: arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012

# 2. Create alias for easier reference
aws kms create-alias \
  --alias-name alias/secrets-master-key \
  --target-key-id <KeyId-from-step-1> \
  --region us-east-1

# 3. Grant IAM permissions
# Attach this policy to your application's IAM role/user:
cat > secrets-kms-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "kms:Encrypt",
        "kms:Decrypt",
        "kms:DescribeKey"
      ],
      "Resource": "arn:aws:kms:us-east-1:123456789012:key/12345678-1234-1234-1234-123456789012"
    }
  ]
}
EOF

aws iam put-user-policy \
  --user-name secrets-app \
  --policy-name SecretsKMSPolicy \
  --policy-document file://secrets-kms-policy.json
```

#### AWS Generate Encrypted Master Key

```bash
# Set AWS credentials
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_REGION="us-east-1"

# Option 1: Use KMS key ID
./bin/app create-master-key \
  --kms-provider=awskms \
  --kms-key-uri="awskms:///12345678-1234-1234-1234-123456789012"

# Option 2: Use alias
./bin/app create-master-key \
  --kms-provider=awskms \
  --kms-key-uri="awskms:///alias/secrets-master-key"
```

#### AWS Environment Configuration

```bash
AWS_ACCESS_KEY_ID=your-access-key
AWS_SECRET_ACCESS_KEY=your-secret-key
AWS_REGION=us-east-1
KMS_PROVIDER=awskms
KMS_KEY_URI=awskms:///alias/secrets-master-key
MASTER_KEYS=<generated-encrypted-key>
ACTIVE_MASTER_KEY_ID=<key-id>
```

### Azure Key Vault

#### Azure Prerequisites

1. **Azure Subscription**: Active Azure subscription
2. **Key Vault**: Azure Key Vault instance created
3. **Service Principal**: With `keys/encrypt` and `keys/decrypt` permissions

#### Azure Setup Steps

```bash
# 1. Create resource group
az group create \
  --name secrets-rg \
  --location eastus

# 2. Create Key Vault
az keyvault create \
  --name secrets-kv-unique \
  --resource-group secrets-rg \
  --location eastus \
  --sku standard

# 3. Create key
az keyvault key create \
  --vault-name secrets-kv-unique \
  --name master-key-encryption \
  --protection software \
  --size 2048 \
  --kty RSA

# 4. Create service principal
az ad sp create-for-rbac \
  --name secrets-kms-sp \
  --role "Key Vault Crypto User" \
  --scopes /subscriptions/<subscription-id>/resourceGroups/secrets-rg/providers/Microsoft.KeyVault/vaults/secrets-kv-unique

# Output: Save tenantId, appId, password

# 5. Set access policy
az keyvault set-policy \
  --name secrets-kv-unique \
  --spn <appId-from-step-4> \
  --key-permissions encrypt decrypt
```

#### Azure Generate Encrypted Master Key

```bash
# Set Azure credentials
export AZURE_TENANT_ID="your-tenant-id"
export AZURE_CLIENT_ID="your-client-id"
export AZURE_CLIENT_SECRET="your-client-secret"

# Generate encrypted master key
./bin/app create-master-key \
  --kms-provider=azurekeyvault \
  --kms-key-uri="azurekeyvault://secrets-kv-unique.vault.azure.net/keys/master-key-encryption"
```

#### Azure Environment Configuration

```bash
AZURE_TENANT_ID=your-tenant-id
AZURE_CLIENT_ID=your-client-id
AZURE_CLIENT_SECRET=your-client-secret
KMS_PROVIDER=azurekeyvault
KMS_KEY_URI=azurekeyvault://secrets-kv-unique.vault.azure.net/keys/master-key-encryption
MASTER_KEYS=<generated-encrypted-key>
ACTIVE_MASTER_KEY_ID=<key-id>
```

### HashiCorp Vault

#### Vault Prerequisites

1. **Vault Server**: Running HashiCorp Vault instance
2. **Transit Engine**: Enabled transit secrets engine
3. **Token/AppRole**: Authentication credentials with `encrypt` and `decrypt` permissions

#### Vault Setup Steps

```bash
# 1. Enable transit secrets engine
vault secrets enable transit

# 2. Create encryption key
vault write -f transit/keys/master-key-encryption

# 3. Create policy
cat > secrets-kms-policy.hcl <<EOF
path "transit/encrypt/master-key-encryption" {
  capabilities = ["update"]
}

path "transit/decrypt/master-key-encryption" {
  capabilities = ["update"]
}
EOF

vault policy write secrets-kms secrets-kms-policy.hcl

# 4. Create token with policy
vault token create -policy=secrets-kms
# Output: Save the token
```

#### Vault Generate Encrypted Master Key

```bash
# Set Vault credentials
export VAULT_ADDR="https://vault.example.com:8200"
export VAULT_TOKEN="your-vault-token"

# Generate encrypted master key
./bin/app create-master-key \
  --kms-provider=hashivault \
  --kms-key-uri="hashivault:///transit/keys/master-key-encryption"
```

#### Vault Environment Configuration

```bash
VAULT_ADDR=https://vault.example.com:8200
VAULT_TOKEN=your-vault-token
KMS_PROVIDER=hashivault
KMS_KEY_URI=hashivault:///transit/keys/master-key-encryption
MASTER_KEYS=<generated-encrypted-key>
ACTIVE_MASTER_KEY_ID=<key-id>
```

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

Kubernetes example:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: secrets-api
spec:
  template:
    spec:
      containers:
        - name: app
          image: allisson/secrets
          env:
            - name: KMS_PROVIDER
              value: gcpkms
            - name: KMS_KEY_URI
              valueFrom:
                secretKeyRef:
                  name: secrets-kms
                  key: kms-key-uri
            - name: MASTER_KEYS
              valueFrom:
                secretKeyRef:
                  name: secrets-master-keys
                  key: master-keys
            - name: ACTIVE_MASTER_KEY_ID
              valueFrom:
                secretKeyRef:
                  name: secrets-master-keys
                  key: active-master-key-id
```

## Migration from Legacy Mode

To migrate from plaintext master keys to KMS mode:

### Step 1: Set Up KMS Provider

Follow provider-specific setup instructions above.

### Step 2: Generate New KMS-Encrypted Master Key

```bash
./bin/app create-master-key \
  --id=master-key-kms-2026 \
  --kms-provider=<provider> \
  --kms-key-uri=<uri>
```

### Step 3: Re-encode Existing Master Keys for KMS

Do not mix plaintext and KMS-encrypted entries in `MASTER_KEYS` when KMS mode is enabled.

Unsupported (do not use):

```bash
MASTER_KEYS=old-plaintext-key:<plaintext-base64>,new-key:<kms-ciphertext>
```

Supported KMS mode input: all entries must be KMS-encrypted ciphertext.

```bash
# Example shape (all values are KMS-encrypted ciphertext)
MASTER_KEYS=old-key:<kms-ciphertext-for-old-key>,master-key-kms-2026:<kms-ciphertext-for-new-key>
ACTIVE_MASTER_KEY_ID=old-key
KMS_PROVIDER=<provider>
KMS_KEY_URI=<uri>
```

To produce `<kms-ciphertext-for-old-key>`, use your provider's native encrypt API with the
existing plaintext 32-byte key material.

Provider examples for re-encoding an existing plaintext key:

```bash
# Input: old plaintext key as base64 string (from legacy MASTER_KEYS value)
OLD_KEY_B64="bEu+O/9NOFAsWf1dhVB9aprmumKhhBcE6o7UPVmI43Y="
printf '%s' "$OLD_KEY_B64" | base64 --decode > /tmp/old-master-key.bin
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

OLD_KEY_KMS_CIPHERTEXT="$(base64 < /tmp/old-master-key.cipher | tr -d '\n')"
```

AWS KMS:

```bash
OLD_KEY_KMS_CIPHERTEXT="$(aws kms encrypt \
  --key-id alias/secrets-master-key \
  --plaintext fileb:///tmp/old-master-key.bin \
  --query CiphertextBlob \
  --output text)"
```

Azure Key Vault:

```bash
OLD_KEY_KMS_CIPHERTEXT="$(az keyvault key encrypt \
  --vault-name secrets-kv-unique \
  --name master-key-encryption \
  --algorithm RSA-OAEP-256 \
  --file /tmp/old-master-key.bin \
  --query result \
  --output tsv)"
```

HashiCorp Vault Transit:

```bash
OLD_KEY_KMS_CIPHERTEXT="$(vault write -field=ciphertext transit/encrypt/master-key-encryption \
  plaintext="$OLD_KEY_B64")"
```

Then build your KMS-only chain:

```bash
MASTER_KEYS="old-key:${OLD_KEY_KMS_CIPHERTEXT},master-key-kms-2026:<kms-ciphertext-for-new-key>"
```

### Step 4: Update Environment (Encrypted-Only Chain)

Update environment with only KMS-encrypted `MASTER_KEYS` entries:

```bash
KMS_PROVIDER=<provider>
KMS_KEY_URI=<uri>
MASTER_KEYS=old-key:<kms-ciphertext-for-old-key>,master-key-kms-2026:<kms-ciphertext-for-new-key>
ACTIVE_MASTER_KEY_ID=old-key
```

### Step 5: Restart Application

Verify both keys are loaded:

```text
INFO KMS mode enabled provider=gcpkms
INFO master key decrypted via KMS key_id=old-key
INFO master key decrypted via KMS key_id=master-key-kms-2026
INFO master key chain loaded active_master_key_id=old-key total_keys=2
```

### Step 6: Rotate KEKs to New Master Key

```bash
# Switch active master key to KMS version
export ACTIVE_MASTER_KEY_ID=master-key-kms-2026

# Restart application
./bin/app server

# Rotate all KEKs (re-encrypts with new master key)
./bin/app rotate-kek --algorithm aes-gcm
```

### Step 7: Remove Old Master Key

After verifying all KEKs are encrypted with the new master key:

```bash
# Remove old key from MASTER_KEYS
MASTER_KEYS=master-key-kms-2026:<kms-encrypted-ciphertext>
ACTIVE_MASTER_KEY_ID=master-key-kms-2026
```

### Migration Checklist

Use this checklist for migrating from legacy plaintext master keys to KMS mode.

#### 1) Precheck

- [ ] Confirm target release is v0.8.0 or newer
- [ ] Back up current environment configuration
- [ ] Confirm rollback owner and change window
- [ ] Confirm KMS provider credentials are available in runtime
- [ ] Confirm KMS encrypt/decrypt permissions are granted

#### 2) Build KMS key chain

- [ ] Generate new KMS-encrypted key with `create-master-key --kms-provider ... --kms-key-uri ...`
- [ ] Re-encode existing legacy plaintext keys into KMS ciphertext
- [ ] Build `MASTER_KEYS` with only KMS ciphertext entries (no plaintext mix)
- [ ] Set `KMS_PROVIDER`, `KMS_KEY_URI`, and `ACTIVE_MASTER_KEY_ID`

#### 3) Rollout

- [ ] Restart API instances (rolling)
- [ ] Verify startup logs show KMS mode and key decrypt lines
- [ ] Run baseline checks: `GET /health`, `GET /ready`
- [ ] Run key-dependent smoke checks: token issuance, secrets, transit

Reference: [Production rollout golden path](../deployment/production-rollout.md)

#### 4) Rotation and cleanup

- [ ] Rotate KEK after switching to KMS key chain
- [ ] Verify reads/decrypt for existing data still succeed
- [ ] Remove old key entries from `MASTER_KEYS` only after verification
- [ ] Restart API instances again after key-chain cleanup

Reference: [Key management operations](../kms/key-management.md)

#### 5) Rollback readiness

- [ ] Keep previous image tag available
- [ ] Keep pre-change env snapshot available
- [ ] If rollback needed, revert app version first
- [ ] Re-validate health and smoke checks after rollback

Reference: [Release notes](../../releases/RELEASES.md#070---2026-02-20)

## Key Rotation

Rotate master keys regularly (recommended: every 90-180 days).

### Generate New Master Key

```bash
./bin/app rotate-master-key --id=master-key-2026-08
```

Output includes combined configuration:

```bash
KMS_PROVIDER=gcpkms
KMS_KEY_URI=gcpkms://...
MASTER_KEYS=old-key:<old-ciphertext>,master-key-2026-08:<new-ciphertext>
ACTIVE_MASTER_KEY_ID=master-key-2026-08
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
MASTER_KEYS=master-key-2026-08:<new-ciphertext>
ACTIVE_MASTER_KEY_ID=master-key-2026-08

# 6. Restart application
./bin/app server
```

## Troubleshooting

### Issue: "KMS keeper does not support encryption"

**Cause**: Provider driver not imported correctly.

**Solution**: Verify provider import in code (should be automatic for supported providers).

### Issue: "failed to open KMS keeper: authentication failed"

**Cause**: Missing or invalid credentials for KMS provider.

**Solution**:

- **GCP**: Check `GOOGLE_APPLICATION_CREDENTIALS` points to valid service account key
- **AWS**: Verify `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` are set
- **Azure**: Confirm `AZURE_TENANT_ID`, `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET` are correct
- **Vault**: Ensure `VAULT_ADDR` and `VAULT_TOKEN` are valid

### Issue: "KMS_PROVIDER is set but KMS_KEY_URI is empty"

**Cause**: Incomplete KMS configuration.

**Solution**: Both `KMS_PROVIDER` and `KMS_KEY_URI` must be set together (or both empty for legacy mode).

### Issue: "failed to decrypt master key via KMS"

**Cause**: KMS key permissions insufficient or key deleted/disabled.

**Solution**:

- Verify IAM permissions include `decrypt` capability
- Check KMS key is enabled and not scheduled for deletion
- Confirm `KMS_KEY_URI` matches the key used during encryption

### Issue: Startup fails with mixed plaintext and KMS master keys

**Cause**: `MASTER_KEYS` contains a mix of plaintext base64 and KMS ciphertext entries while KMS mode is enabled.

**Solution**:

- Use plaintext entries only in legacy mode (both `KMS_PROVIDER` and `KMS_KEY_URI` unset)
- Use KMS ciphertext entries only in KMS mode (both KMS variables set)
- Re-encode legacy keys with provider-native encrypt APIs before enabling KMS mode

### Issue: Application slow to start with KMS enabled

**Cause**: KMS decryption happens at startup (network round-trip).

**Expected Behavior**: Startup delay of 100-500ms per master key (acceptable trade-off for security).

**Optimization**: Minimize number of master keys (typically 1-2 keys).

### Debug Logging

Enable debug logs to troubleshoot KMS issues:

```bash
LOG_LEVEL=debug ./bin/app server
```

Look for:

```text
DEBUG KMS keeper opened uri=gcpkms://...
DEBUG master key decrypted key_id=master-key-2026-02-19 ciphertext_length=64
```

## See Also

- [Key Management Guide](../kms/key-management.md) - KEK and DEK rotation procedures
- [Security Hardening](../security/hardening.md) - Production security best practices
- [Production Deployment](../deployment/production.md) - Production deployment checklist
