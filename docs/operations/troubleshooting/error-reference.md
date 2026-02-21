# ❌ Error Message Reference

> **Document version**: v0.10.0  
> Last updated: 2026-02-21  
> **Audience**: Developers, DevOps engineers, SRE teams troubleshooting Secrets errors

## Overview

This reference documents all error messages you might encounter when running Secrets, along with their causes and solutions. For step-by-step troubleshooting workflows, see the [Troubleshooting Guide](../../getting-started/troubleshooting.md).

**How to use this guide:**

1. **Find your error**: Use Ctrl+F to search for exact error message text
2. **Check the cause**: Understand why the error occurred
3. **Apply the solution**: Follow the remediation steps
4. **Verify the fix**: Test that the error is resolved

**Error categories:**

- [HTTP API Errors (4xx, 5xx)](#http-api-errors)
- [Database Errors](#database-errors)
- [KMS and Encryption Errors](#kms-and-encryption-errors)
- [Container and Runtime Errors](#container-and-runtime-errors)
- [Configuration Errors](#configuration-errors)
- [Validation Errors](#validation-errors)

---

## HTTP API Errors

### 400 Bad Request

**Error**: `400 Bad Request`

**Typical response body:**

```json

{
  "error": "invalid request",
  "details": "request body must be JSON"
}

```

**Causes:**

- Malformed JSON in request body
- Missing `Content-Type: application/json` header
- Invalid URL parameters (non-UUID where UUID expected)

**Solutions:**

```bash

# Wrong: invalid JSON (missing quotes)
curl -X POST http://localhost:8080/v1/secrets/test \
  -d '{value: dGVzdA==}'

# Correct: valid JSON
curl -X POST http://localhost:8080/v1/secrets/test \
  -H "Content-Type: application/json" \
  -d '{"value":"dGVzdA=="}'

```

**Related errors:**

- `422 Unprocessable Entity` - Valid JSON, but failed validation

---

### 401 Unauthorized

**Error**: `401 Unauthorized`

**Typical response body:**

```json

{
  "error": "unauthorized",
  "message": "missing or invalid token"
}

```

**Causes:**

1. Missing `Authorization` header
2. Invalid token format (not `Bearer <token>`)
3. Token expired (TTL exceeded)
4. Token signature invalid (forged token)
5. Client credentials incorrect (during token issuance)

**Solutions:**

**Missing token:**

```bash

# Wrong: no Authorization header
curl http://localhost:8080/v1/secrets/test

# Correct: include Bearer token
curl http://localhost:8080/v1/secrets/test \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..."

```

**Token expired:**

```bash

# Get new token (default TTL: 1 hour)
TOKEN=$(curl -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{"client_id":"your-client-id","client_secret":"your-secret"}' | \
  jq -r '.token')

# Use new token
curl http://localhost:8080/v1/secrets/test \
  -H "Authorization: Bearer $TOKEN"

```

**Invalid client credentials:**

```bash

# Verify client_id exists
psql -d secrets -c "SELECT id, name, is_active FROM clients WHERE id = 'your-client-id';"

# Regenerate client secret if lost (requires direct DB access)
# Client secrets are hashed and cannot be retrieved - must regenerate
```

**Related errors:**

- `403 Forbidden` - Token valid, but insufficient permissions
- `429 Too Many Requests` - Token endpoint rate limited

**See also**: [Authentication Guide](../../api/auth/authentication.md)

---

### 403 Forbidden

**Error**: `403 Forbidden`

**Typical response body:**

```json

{
  "error": "forbidden",
  "message": "insufficient permissions"
}

```

**Causes:**

1. Client policy doesn't grant required capability for the endpoint
2. Path pattern in policy doesn't match request path
3. Client is inactive (`is_active = false`)

**Solutions:**

**Check client policies:**

```bash

# Get token (to identify which client is making requests)
TOKEN="your-token-here"

# Decode token to see client_id (JWT)
echo "$TOKEN" | cut -d. -f2 | base64 -d | jq

# Query client policies (requires DB access)
psql -d secrets -c "
  SELECT p.path, p.capabilities, c.name as client_name
  FROM policies p
  JOIN clients c ON p.client_id = c.id
  WHERE c.id = 'your-client-id';
"

```

**Example permission fix:**

**Problem**: `POST /v1/secrets/prod/api-key` returns 403

**Cause**: Client policy only has `"read"` capability

```sql

-- Current policy (insufficient)
{"path": "/v1/secrets/*", "capabilities": ["read"]}

```

**Solution**: Add `"write"` capability

```sql

-- Update policy
UPDATE policies
SET capabilities = ARRAY['read', 'write']
WHERE client_id = 'your-client-id' AND path = '/v1/secrets/*';

```

**Related errors:**

- `401 Unauthorized` - No token or invalid token
- `404 Not Found` - Client doesn't have visibility to resource

**See also**: [Authorization Policies Guide](../../api/auth/policies.md)

---

### 404 Not Found

**Error**: `404 Not Found`

**Typical response body:**

```json

{
  "error": "not found",
  "message": "resource not found"
}

```

**Causes:**

1. Resource doesn't exist (secret, transit key, client)
2. Wrong resource ID in URL
3. Resource exists but client policy blocks access (authorization hiding)
4. Wrong API endpoint path (typo)

**Solutions:**

**Verify resource exists:**

```bash

# Check if secret exists (requires DB access)
psql -d secrets -c "SELECT id, name FROM secrets WHERE name = 'prod/api-key';"

# Check if transit key exists
psql -d secrets -c "SELECT id, name FROM transit_keys WHERE name = 'production';"

```

**Check for typos:**

```bash

# Wrong: typo in endpoint path
curl http://localhost:8080/v1/secret/test  # Missing 's' in 'secrets'

# Correct: proper endpoint path
curl http://localhost:8080/v1/secrets/test

```

**Authorization hiding**: Some endpoints return 404 instead of 403 to prevent information disclosure (e.g., secret existence).

**Related errors:**

- `403 Forbidden` - Resource exists, but access denied

---

### 409 Conflict

**Error**: `409 Conflict`

**Typical response body:**

```json

{
  "error": "conflict",
  "message": "resource already exists"
}

```

**Causes:**

1. Creating resource with duplicate unique identifier (secret name, client ID, transit key name)
2. Resource state conflict (e.g., rotating inactive transit key)

**Solutions:**

**Duplicate secret:**

```bash

# Wrong: creating secret that already exists
curl -X POST http://localhost:8080/v1/secrets/prod/api-key \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"value":"dGVzdA=="}'
# Error: 409 Conflict - secret 'prod/api-key' already exists

# Solution 1: Update existing secret instead
curl -X PUT http://localhost:8080/v1/secrets/prod/api-key \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"value":"bmV3VmFsdWU="}'

# Solution 2: Delete then recreate (DANGEROUS - loses secret history)
curl -X DELETE http://localhost:8080/v1/secrets/prod/api-key \
  -H "Authorization: Bearer $TOKEN"
curl -X POST http://localhost:8080/v1/secrets/prod/api-key \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"value":"dGVzdA=="}'

# Solution 3: Use different secret name
curl -X POST http://localhost:8080/v1/secrets/prod/api-key-v2 \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"value":"dGVzdA=="}'

```

**Related errors:**

- `422 Unprocessable Entity` - Validation failed before checking uniqueness

---

### 422 Unprocessable Entity

**Error**: `422 Unprocessable Entity`

**Typical response body:**

```json

{
  "error": "validation failed",
  "details": {
    "field": "value",
    "error": "value must be base64-encoded"
  }
}

```

**Causes:**

1. Request body fails validation (missing required fields, invalid format)
2. Query parameters fail validation (invalid page size, invalid filter)
3. Business logic validation failed (e.g., invalid capability name in policy)

**Solutions:**

**Missing required field:**

```bash

# Wrong: missing required 'value' field
curl -X POST http://localhost:8080/v1/secrets/test \
  -H "Authorization: Bearer $TOKEN" \
  -d '{}'
# Error: 422 - field 'value' is required

# Correct: include all required fields
curl -X POST http://localhost:8080/v1/secrets/test \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"value":"dGVzdA=="}'

```

**Invalid base64 encoding:**

```bash

# Wrong: plaintext value (not base64)
curl -X POST http://localhost:8080/v1/secrets/test \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"value":"plaintext"}'
# Error: 422 - value must be base64-encoded

# Correct: base64-encode value first
echo -n "plaintext" | base64  # cGxhaW50ZXh0
curl -X POST http://localhost:8080/v1/secrets/test \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"value":"cGxhaW50ZXh0"}'

```

**Invalid capability:**

```bash

# Wrong: invalid capability name
curl -X POST http://localhost:8080/v1/clients \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "app-client",
    "policies": [{"path": "/v1/secrets/*", "capabilities": ["read", "invalid"]}]
  }'
# Error: 422 - invalid capability 'invalid'

# Correct: use valid capabilities
# Valid: read, write, delete, encrypt, decrypt, rotate
curl -X POST http://localhost:8080/v1/clients \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "name": "app-client",
    "policies": [{"path": "/v1/secrets/*", "capabilities": ["read", "write"]}]
  }'

```

**Related errors:**

- `400 Bad Request` - Malformed JSON (before validation)

**See also**: [API Validation Rules](../../api/fundamentals.md)

---

### 429 Too Many Requests

**Error**: `429 Too Many Requests`

**Typical response headers:**

```text

HTTP/1.1 429 Too Many Requests
X-RateLimit-Limit: 10
X-RateLimit-Remaining: 0
X-RateLimit-Reset: 1645564800
Retry-After: 60

```

**Typical response body:**

```json

{
  "error": "rate limit exceeded",
  "message": "too many requests from this IP address"
}

```

**Causes:**

1. Token endpoint rate limited by IP address (default: 10 requests/minute)
2. Too many authentication attempts from same IP

**Solutions:**

**Wait and retry:**

```bash

# Check Retry-After header
curl -I http://localhost:8080/v1/token

# Wait specified seconds, then retry
sleep 60
curl -X POST http://localhost:8080/v1/token \
  -d '{"client_id":"...","client_secret":"..."}'

```

**Implement exponential backoff:**

```python

import time
import requests

def get_token_with_backoff(client_id, secret, max_retries=5):
    for attempt in range(max_retries):
        response = requests.post('http://localhost:8080/v1/token', json={
            'client_id': client_id,
            'client_secret': secret
        })
        
        if response.status_code == 200:
            return response.json()['token']
        elif response.status_code == 429:
            retry_after = int(response.headers.get('Retry-After', 60))
            print(f"Rate limited, waiting {retry_after}s...")
            time.sleep(retry_after)
        else:
            raise Exception(f"Token request failed: {response.status_code}")
    
    raise Exception("Max retries exceeded")

```

**Adjust rate limit** (requires configuration change):

```bash

# Increase rate limit (requires app restart)
# Edit .env or environment variables
RATE_LIMIT_MAX_REQUESTS=20  # default: 10
RATE_LIMIT_DURATION=60      # seconds, default: 60

# Restart application
docker restart secrets-api

```

**Related errors:**

- `401 Unauthorized` - Wrong credentials (will trigger rate limit after 10 attempts)

**See also**: [Rate Limiting Configuration](../../configuration.md#rate-limiting-configuration)

---

### 500 Internal Server Error

**Error**: `500 Internal Server Error`

**Typical response body:**

```json

{
  "error": "internal server error",
  "message": "an unexpected error occurred"
}

```

**Causes:**

1. Database connection failure
2. KMS provider unreachable or authentication failed
3. Encryption/decryption failure (corrupt master key)
4. Application panic (bug)

**Solutions:**

**Check application logs:**

```bash

# Docker
docker logs secrets-api --tail=100

# Docker Compose
docker compose logs secrets --tail=100

# Look for stack traces or error details
```

**Common 500 error patterns:**

#### Database connection lost

**Log pattern:**

```text

ERROR: database connection failed: dial tcp 127.0.0.1:5432: connect: connection refused

```

**Solution**:

```bash

# Verify database is running
docker ps | grep postgres

# Check database connectivity
psql -h localhost -U secrets -d secrets -c "SELECT 1;"

# Verify DB_CONNECTION_STRING
echo $DB_CONNECTION_STRING
# postgresql://secrets:password@localhost:5432/secrets?sslmode=disable

# Restart application (reconnects to database)
docker restart secrets-api

```

#### KMS provider unreachable

**Log pattern:**

```text

ERROR: failed to decrypt master key: kms: failed to call Decrypt: RequestError: send request failed

```

**Solution**:

```bash

# Check KMS provider configuration
echo $KMS_PROVIDER  # aws-kms, gcp-kms, azure-kv
echo $KMS_KEY_URI   # arn:aws:kms:..., projects/.../keys/..., https://...

# Verify network connectivity to KMS
# AWS KMS
aws kms describe-key --key-id $KMS_KEY_URI

# GCP KMS
gcloud kms keys describe ... --location=... --keyring=...

# Azure Key Vault
az keyvault key show --vault-name ... --name ...

# Check IAM/RBAC permissions (service account, IAM role, managed identity)
```

**Related errors:**

- `503 Service Unavailable` - Temporary issue, retry may succeed

**See also**: [Incident Response Guide](../observability/incident-response.md)

---

## Database Errors

### "connection refused" / "connection reset by peer"

**Error**:

```text

ERROR: database connection failed: dial tcp 127.0.0.1:5432: connect: connection refused

```

**Causes:**

1. Database server not running
2. Wrong host/port in connection string
3. Firewall blocking connection
4. Database not accepting connections (PostgreSQL: `listen_addresses` config)

**Solutions:**

**Verify database is running:**

```bash

# PostgreSQL
docker ps | grep postgres
systemctl status postgresql

# MySQL
docker ps | grep mysql
systemctl status mysql

# Cloud databases
# AWS RDS: Check RDS console
# Google Cloud SQL: Check Cloud SQL console
# Azure Database: Check Azure portal
```

**Test connection:**

```bash

# PostgreSQL
psql -h localhost -U secrets -d secrets -c "SELECT version();"

# MySQL
mysql -h localhost -u secrets -p -D secrets -e "SELECT VERSION();"

# If connection fails, check:
# - Host/port correct in DB_CONNECTION_STRING
# - Database credentials correct
# - Database allows remote connections
# - Firewall rules allow traffic on port 5432 (PostgreSQL) or 3306 (MySQL)
```

**Fix connection string:**

```bash

# Wrong: using 127.0.0.1 when database is in Docker network
DB_CONNECTION_STRING="postgresql://secrets:password@127.0.0.1:5432/secrets?sslmode=disable"

# Correct: using Docker service name
DB_CONNECTION_STRING="postgresql://secrets:password@postgres:5432/secrets?sslmode=disable"

# Restart application
docker restart secrets-api

```

---

### "role does not exist" / "access denied for user"

**Error** (PostgreSQL):

```text

ERROR: role "secrets" does not exist

```

**Error** (MySQL):

```text

ERROR 1045 (28000): Access denied for user 'secrets'@'localhost'

```

**Causes:**

- Malformed JSON in request body
- Missing `Content-Type: application/json` header
- Invalid URL parameters (non-UUID where UUID expected)

**Solutions:**

```bash

# Test connection manually
psql -h localhost -U secrets -d secrets
# If password prompt fails, password is wrong

# Update DB_CONNECTION_STRING with correct credentials
DB_CONNECTION_STRING="postgresql://secrets:correct-password@localhost:5432/secrets?sslmode=disable"

```

---

### "database does not exist"

**Error**:

```text

ERROR: database "secrets" does not exist

```

**Causes:**

1. Database not created
2. Wrong database name in connection string

**Solutions:**

```sql

-- PostgreSQL
CREATE DATABASE secrets;

-- MySQL
CREATE DATABASE secrets;

```

```bash

# Verify database exists
psql -l | grep secrets     # PostgreSQL
mysql -e "SHOW DATABASES;" # MySQL

# Run migrations to create schema
docker run --rm \
  -e DB_DRIVER=postgres \
  -e DB_CONNECTION_STRING="postgresql://secrets:password@postgres:5432/secrets?sslmode=disable" \
  allisson/secrets:v0.10.0 migrate

```

---

### "relation does not exist" / "table doesn't exist"

**Error** (PostgreSQL):

```text

ERROR: relation "clients" does not exist

```

**Error** (MySQL):

```text

ERROR 1146 (42S02): Table 'secrets.clients' doesn't exist

```

**Causes:**

1. Database migrations not run
2. Wrong database in connection string (connected to empty database)

**Solutions:**

**Run migrations:**

```bash

# Docker
docker run --rm \
  -e DB_DRIVER=postgres \
  -e DB_CONNECTION_STRING="$DB_CONNECTION_STRING" \
  allisson/secrets:v0.10.0 migrate

# Docker Compose
docker compose run --rm secrets-api migrate

```

## Verify migrations ran

```bash

```

**Check database schema:**

```bash

# PostgreSQL - list tables
psql -d secrets -c "\dt"

# Expected tables: clients, policies, secrets, transit_keys, audit_logs, schema_migrations

# MySQL - list tables
mysql -D secrets -e "SHOW TABLES;"

```

---

## KMS and Encryption Errors

### "master key not configured"

**Error**:

```text

FATAL: master key not configured: MASTER_KEY_PROVIDER must be set

```

**Causes:**

1. `MASTER_KEY_PROVIDER` environment variable not set
2. Wrong provider name (typo)

**Solutions:**

```bash

# Set provider (choose one)
MASTER_KEY_PROVIDER=plaintext         # Development only, NOT for production
MASTER_KEY_PROVIDER=aws-kms           # AWS KMS
MASTER_KEY_PROVIDER=gcp-kms           # Google Cloud KMS
MASTER_KEY_PROVIDER=azure-kv          # Azure Key Vault

# For plaintext provider (development)
MASTER_KEY_PLAINTEXT=$(openssl rand -base64 32)

# For cloud providers, set KMS_KEY_URI
KMS_KEY_URI="arn:aws:kms:us-east-1:123456789012:key/abc-123..."  # AWS
KMS_KEY_URI="projects/my-project/locations/us/keyRings/secrets/cryptoKeys/master"  # GCP
KMS_KEY_URI="https://my-vault.vault.azure.net/keys/master-key/abc123..."  # Azure

# Restart application
docker restart secrets-api

```

**See also**: [KMS Setup Guide](../kms/setup.md)

---

### "failed to decrypt master key"

**Error**:

```text

ERROR: failed to decrypt master key: kms: operation error KMS: Decrypt, https response error StatusCode: 403, AccessDeniedException: User is not authorized to perform: kms:Decrypt

```

**Causes:**

1. KMS key permissions incorrect (IAM role, service account, managed identity)
2. KMS key disabled or deleted
3. Wrong KMS_KEY_URI
4. Network connectivity issue to KMS provider

**Solutions:**

**Verify KMS key permissions:**

```bash

# AWS KMS - check IAM role attached to ECS task or EC2 instance
aws sts get-caller-identity  # Verify which IAM principal is being used
aws kms describe-key --key-id $KMS_KEY_URI  # Verify key exists
aws kms decrypt --key-id $KMS_KEY_URI --ciphertext-blob fileb://test.enc  # Test decrypt

# GCP KMS - check service account
gcloud auth list  # Verify which service account is active
gcloud kms keys describe ... --location=... --keyring=...  # Verify key exists
gcloud kms decrypt --key=... --location=... --keyring=... --ciphertext-file=test.enc --plaintext-file=-  # Test decrypt

# Azure Key Vault - check managed identity
az account show  # Verify which subscription/tenant
az keyvault key show --vault-name ... --name ...  # Verify key exists
az keyvault key decrypt --vault-name ... --name ... --algorithm RSA-OAEP-256 --value ...  # Test decrypt

```

**Grant KMS permissions:**

```bash

# AWS KMS - attach IAM policy to role
aws iam attach-role-policy \
  --role-name secrets-api-role \
  --policy-arn arn:aws:iam::aws:policy/AWSKeyManagementServicePowerUser

# Or use custom policy (least privilege)
aws iam put-role-policy --role-name secrets-api-role --policy-name kms-decrypt --policy-document '{
  "Version": "2012-10-17",
  "Statement": [{
    "Effect": "Allow",
    "Action": ["kms:Decrypt", "kms:DescribeKey"],
    "Resource": "arn:aws:kms:us-east-1:123456789012:key/*"
  }]
}'

# GCP KMS - grant Cloud KMS CryptoKey Decrypter role
gcloud kms keys add-iam-policy-binding master-key \
  --location=us \
  --keyring=secrets \
  --member="serviceAccount:secrets-api@my-project.iam.gserviceaccount.com" \
  --role="roles/cloudkms.cryptoKeyDecrypter"

# Azure Key Vault - assign Key Vault Crypto User role
az role assignment create \
  --assignee <managed-identity-principal-id> \
  --role "Key Vault Crypto User" \
  --scope /subscriptions/.../resourceGroups/.../providers/Microsoft.KeyVault/vaults/my-vault/keys/master-key

```

**See also**: [KMS Setup Guide](../kms/setup.md)

---

## Container and Runtime Errors

### "permission denied" (volume mounts)

**Error**:

```text

panic: open /data/app.db: permission denied

```

**Causes:**
v0.10.0+ runs as non-root user (UID 65532), but volume is owned by root or another user.

**Solutions:**

See dedicated guide: [Volume Permission Troubleshooting](volume-permissions.md)

---

### "exec format error" (wrong architecture)

**Error**:

```text

standard_init_linux.go:228: exec user process caused: exec format error

```

**Causes:**
Running ARM64 image on x86_64 host (or vice versa) without QEMU emulation.

**Solutions:**

```bash

# Force pull correct architecture
docker pull --platform linux/amd64 allisson/secrets:v0.10.0

# Or enable QEMU for cross-platform support
docker run --privileged --rm tonistiigi/binfmt --install all

# Verify architecture
docker inspect allisson/secrets:v0.10.0 --format='{{.Architecture}}'

```

**See also**: [Multi-Architecture Build Guide](../deployment/multi-arch-builds.md)

---

### "no such file or directory" (missing binary)

**Error**:

```text

docker: Error response from daemon: failed to create shim task: OCI runtime create failed: runc create failed: unable to start container process: exec: "/app": stat /app: no such file or directory

```

**Causes:**

1. Binary not copied to expected path in Dockerfile
2. Wrong ENTRYPOINT path
3. Dynamic binary on static-only distroless base

**Solutions:**

```dockerfile

# Verify binary is copied correctly
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=builder /build/app /app
ENTRYPOINT ["/app"]

# Verify binary is statically compiled
RUN CGO_ENABLED=0 go build -o app ./cmd/app
RUN ldd /build/app  # Should output: "not a dynamic executable"

```

---

## Configuration Errors

### "unknown configuration key"

**Error**:

```text

WARN: unknown configuration key: LOG_LEVL

```

**Causes:**
Typo in environment variable name.

**Solutions:**

See [Configuration Reference](../../configuration.md) for all valid environment variables.

**Common typos:**

- `LOG_LEVL` → `LOG_LEVEL`
- `DB_CONNNECTION_STRING` → `DB_CONNECTION_STRING`
- `MASTER_KEY_PROVIDOR` → `MASTER_KEY_PROVIDER`

---

## Validation Errors

### "value must be base64-encoded"

**Error**:

```json

{
  "error": "validation failed",
  "details": {
    "field": "value",
    "error": "value must be base64-encoded"
  }
}

```

**Causes:**
Secret value is not valid base64.

**Solutions:**

```bash

# Encode value to base64
echo -n "my-secret-value" | base64
# bXktc2VjcmV0LXZhbHVl

# Use in API request
curl -X POST http://localhost:8080/v1/secrets/test \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"value":"bXktc2VjcmV0LXZhbHVl"}'

```

---

### "path pattern not supported"

**Error**:

```json

{
  "error": "validation failed",
  "details": {
    "field": "path",
    "error": "path pattern 'prod-*' not supported (use '*', '/exact/path', or '/prefix/*')"
  }
}

```

**Causes:**
Policy path uses unsupported wildcard pattern.

**Supported patterns:**

- `*` - Match all paths
- `/v1/secrets/prod` - Exact path match
- `/v1/secrets/*` - Trailing wildcard (all paths under `/v1/secrets/`)
- `/v1/transit/keys/*/rotate` - Mid-path wildcard (single segment)

**Not supported:**

- `prod-*` - Prefix wildcard
- `*-prod` - Suffix wildcard
- `/v1/**` - Recursive wildcard
- `/v1/secrets/prod*` - Partial segment wildcard

**Solutions:**

```bash

# Wrong: partial segment wildcard
{"path": "prod-*", "capabilities": ["read"]}

# Correct: use trailing wildcard or exact path
{"path": "/v1/secrets/prod/*", "capabilities": ["read"]}
{"path": "/v1/secrets/production", "capabilities": ["read"]}

```

**See also**: [Authorization Policies Guide](../../api/auth/policies.md)

---

## Quick Reference: Error Code Summary

| HTTP Code | Meaning | Common Causes | Quick Fix |
|-----------|---------|---------------|-----------|
| **400** | Bad Request | Malformed JSON, invalid URL params | Check request format, add `Content-Type: application/json` |
| **401** | Unauthorized | Missing/invalid token | Get new token via `POST /v1/token` |
| **403** | Forbidden | Insufficient permissions | Update client policies with required capabilities |
| **404** | Not Found | Resource doesn't exist | Verify resource ID, check if resource was deleted |
| **409** | Conflict | Duplicate resource | Use different name/ID, or update existing resource |
| **422** | Unprocessable Entity | Validation failed | Check required fields, validate base64 encoding |
| **429** | Too Many Requests | Rate limit exceeded | Wait and retry, implement exponential backoff |
| **500** | Internal Server Error | Database/KMS failure, application bug | Check logs, verify database/KMS connectivity |
| **503** | Service Unavailable | Temporary overload | Retry with exponential backoff |

---

## See Also

- [Troubleshooting Guide](../../getting-started/troubleshooting.md) - Step-by-step troubleshooting workflows
- [Configuration Reference](../../configuration.md) - All environment variables
- [API Fundamentals](../../api/fundamentals.md) - API error handling patterns
- [Volume Permission Troubleshooting](volume-permissions.md) - v0.10.0+ permission issues
- [KMS Setup Guide](../kms/setup.md) - KMS provider configuration
- [Incident Response Guide](../observability/incident-response.md) - Production incident handling
