# ⚙️ Environment Variables

> Last updated: 2026-02-23

Secrets is configured through environment variables.

## Core configuration

```dotenv
# Database configuration
DB_DRIVER=postgres
DB_CONNECTION_STRING=postgres://user:password@localhost:5432/mydb?sslmode=disable
DB_MAX_OPEN_CONNECTIONS=25
DB_MAX_IDLE_CONNECTIONS=5
DB_CONN_MAX_LIFETIME=5

# Server configuration
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
LOG_LEVEL=info

# Master key configuration
KMS_PROVIDER=
KMS_KEY_URI=
MASTER_KEYS=default:BASE64_32_BYTE_KEY
ACTIVE_MASTER_KEY_ID=default

# Authentication configuration
AUTH_TOKEN_EXPIRATION_SECONDS=14400

# Rate limiting configuration (authenticated endpoints)
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_SEC=10.0
RATE_LIMIT_BURST=20

# Token endpoint rate limiting (IP-based, unauthenticated)
RATE_LIMIT_TOKEN_ENABLED=true
RATE_LIMIT_TOKEN_REQUESTS_PER_SEC=5.0
RATE_LIMIT_TOKEN_BURST=10

# CORS configuration
CORS_ENABLED=false
CORS_ALLOW_ORIGINS=

# Metrics configuration
METRICS_ENABLED=true
METRICS_NAMESPACE=secrets

# Account lockout (PCI DSS 8.3.4)
LOCKOUT_MAX_ATTEMPTS=10
LOCKOUT_DURATION_MINUTES=30

```

## Database configuration

### DB_DRIVER

Database driver to use. Supported values: `postgres`, `mysql`.

See [ADR 0004: Dual Database Support](adr/0004-dual-database-support.md) for the architectural rationale behind dual database support.

### DB_CONNECTION_STRING

Database connection string.

**⚠️ Security Warning:** `sslmode=disable` (PostgreSQL) and `tls=false` (MySQL) are for **development only**. Production deployments **must** use encrypted connections:

**PostgreSQL production:**

```dotenv
# Minimum: encrypted connection
DB_CONNECTION_STRING=postgres://user:password@db.example.com:5432/secrets?sslmode=require

# Recommended: encrypted connection with certificate verification
DB_CONNECTION_STRING=postgres://user:password@db.example.com:5432/secrets?sslmode=verify-full&sslrootcert=/path/to/ca.crt

```

**MySQL production:**

```dotenv
# Minimum: encrypted connection
DB_CONNECTION_STRING=user:password@tcp(db.example.com:3306)/secrets?tls=true

# Recommended: encrypted connection with certificate verification
DB_CONNECTION_STRING=user:password@tcp(db.example.com:3306)/secrets?tls=custom

```

See [Security Hardening Guide](operations/security/hardening.md#2-database-security) for complete guidance.

### DB_MAX_OPEN_CONNECTIONS

Maximum number of open database connections (default: `25`).

### DB_MAX_IDLE_CONNECTIONS

Maximum number of idle database connections (default: `5`).

### DB_CONN_MAX_LIFETIME

Maximum lifetime of a connection in minutes (default: `5`).

## Server configuration

### SERVER_HOST

Host address to bind the HTTP server (default: `0.0.0.0`).

### SERVER_PORT

Port to bind the HTTP server (default: `8080`).

### LOG_LEVEL

Logging level. Supported values: `debug`, `info`, `warn`, `error` (default: `info`).

## Master key configuration

### MASTER_KEYS

Comma-separated list of master keys in format `id1:value1,id2:value2`.

Value format depends on mode:

- Legacy mode: plaintext base64-encoded 32-byte keys

- KMS mode: base64-encoded KMS ciphertext for each 32-byte master key

- 📏 Each master key must represent exactly 32 bytes (256 bits)

- 🔐 Store in secrets manager, never commit to source control

- 🔄 After changing `MASTER_KEYS`, restart API servers to load new values

**Example:**

```dotenv
MASTER_KEYS=default:A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8S9T0U1V2W3X4Y5Z6==

```

### ACTIVE_MASTER_KEY_ID

ID of the master key to use for encrypting new KEKs (default: `default`).

- ⭐ Must match one of the IDs in `MASTER_KEYS`

- 🔄 After changing `ACTIVE_MASTER_KEY_ID`, restart API servers to load new value

### KMS_PROVIDER

Optional KMS provider for master key decryption at startup.

Supported values:

- `localsecrets`

- `gcpkms`

- `awskms`

- `azurekeyvault`

- `hashivault`

### KMS_KEY_URI

KMS key URI for the selected `KMS_PROVIDER`.

Examples:

- `base64key://<base64-32-byte-key>`

- `gcpkms://projects/<project>/locations/<location>/keyRings/<ring>/cryptoKeys/<key>`

- `awskms:///<key-id-or-alias>`

- `azurekeyvault://<vault-name>.vault.azure.net/keys/<key-name>`

- `hashivault:///<transit-key-path>`

**🔒 SECURITY WARNING:**

The `KMS_KEY_URI` variable contains **highly sensitive information** that controls access to all encrypted data in your Secrets deployment. Compromise of this value can lead to complete data exposure.

**Critical security requirements:**

1. **NEVER commit `KMS_KEY_URI` to source control**
   - Use secrets management (AWS Secrets Manager, GCP Secret Manager, Azure Key Vault, HashiCorp Vault)

   - Use environment-specific `.env` files excluded from git (`.env` is in `.gitignore`)

   - Use CI/CD secrets for automated deployments

2. **Restrict access using least privilege**
   - Limit access to personnel with operational requirements only

   - Use role-based access control (RBAC) in your secrets manager

   - Audit access to `KMS_KEY_URI` quarterly

3. **Use KMS provider authentication securely**
   - **GCP KMS**: Use Workload Identity (GKE) or service account keys with rotation

   - **AWS KMS**: Use IAM roles (ECS/EKS) or IAM users with MFA and rotation

   - **Azure Key Vault**: Use Managed Identity (AKS) or service principals with rotation

   - **HashiCorp Vault**: Use AppRole or token auth, never root tokens

4. **Rotate KMS keys regularly**
   - Follow your organization's key rotation policy (typically 90-365 days)

   - Test rotation procedures in staging before production

   - See [KMS setup guide](operations/kms/setup.md#key-rotation) for rotation workflow

5. **Monitor and audit KMS access**
   - Enable CloudTrail (AWS), Cloud Audit Logs (GCP), Azure Monitor (Azure)

   - Alert on unusual KMS key access patterns

   - Review KMS access logs monthly

6. **Use `base64key://` provider ONLY for local development**
   - The `base64key://` provider embeds the encryption key directly in `KMS_KEY_URI`

   - **NEVER use `base64key://` in staging or production environments**

   - Use cloud KMS providers (`gcpkms://`, `awskms://`, `azurekeyvault://`) for production

**Example of insecure vs secure configuration:**

```dotenv
# ❌ INSECURE - Never do this

KMS_PROVIDER=localsecrets
KMS_KEY_URI=base64key://A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8S9T0U1V2W3X4Y5Z6==  # PRODUCTION - DO NOT USE

# ✅ SECURE - Production example (GCP)

KMS_PROVIDER=gcpkms
KMS_KEY_URI=gcpkms://projects/my-prod-project/locations/us-central1/keyRings/secrets-keyring/cryptoKeys/secrets-master-key

# ✅ SECURE - Production example (AWS)

KMS_PROVIDER=awskms
KMS_KEY_URI=awskms:///alias/secrets-master-key

# ✅ SECURE - Production example (Azure)

KMS_PROVIDER=azurekeyvault
KMS_KEY_URI=azurekeyvault://my-prod-vault.vault.azure.net/keys/secrets-master-key

```

**Incident response:**

If `KMS_KEY_URI` is exposed (committed to git, leaked in logs, etc.):

1. **Immediate**: Rotate the KMS key using your cloud provider's console/CLI
2. **Within 24h**: Generate new `MASTER_KEYS` using the new KMS key
3. **Within 48h**: Re-encrypt all KEKs using `rotate-master-key` command
4. **Within 1 week**: Audit all secrets access during exposure window
5. **Post-incident**: Update runbooks, add pre-commit hooks to prevent future leaks

See [Security Hardening Guide](operations/security/hardening.md) and [KMS Setup Guide](operations/kms/setup.md) for complete guidance.

### Master key mode selection

- KMS mode: set both `KMS_PROVIDER` and `KMS_KEY_URI`

- Legacy mode: leave both unset/empty

- Invalid configuration: setting only one of the two variables fails startup

For provider setup and migration workflow, see [KMS setup guide](operations/kms/setup.md).

### KMS preflight checklist

Run this checklist before rolling to production:

1. `KMS_PROVIDER` and `KMS_KEY_URI` are both set (or both unset for legacy mode)
2. `MASTER_KEYS` entries match the selected mode:
   - KMS mode: all entries are KMS ciphertext

   - Legacy mode: all entries are plaintext base64 32-byte keys

3. `ACTIVE_MASTER_KEY_ID` exists in `MASTER_KEYS`
4. Runtime credentials for provider are present and valid
5. Startup logs show successful key loading before traffic cutover

## Authentication configuration

### AUTH_TOKEN_EXPIRATION_SECONDS

Token expiration time in seconds (default: `14400` - 4 hours).

**⚠️ Migration Note:** Prior to v0.5.0, the default was 86400 seconds (24 hours). Review your token expiration settings and client refresh logic when upgrading from v0.4.x.

**Recommended settings:**

- High-security environments: `3600` (1 hour)

- Standard deployments: `14400` (4 hours) - **default**

- Low-security environments: `86400` (24 hours)

## Rate limiting configuration

See [ADR 0006: Dual-Scope Rate Limiting Strategy](adr/0006-dual-scope-rate-limiting-strategy.md) for the architectural rationale behind dual-scope rate limiting.

### RATE_LIMIT_ENABLED

Enable per-client rate limiting (default: `true`).

**Security Note:** Rate limiting protects against abuse and denial-of-service attacks. Disable only for testing or if rate limiting is handled at a different layer.

### RATE_LIMIT_REQUESTS_PER_SEC

Maximum requests per second per authenticated client (default: `10.0`).

**Recommended settings:**

- High-volume API: `50.0`

- Standard application: `10.0` - **default**

- Sensitive operations: `1.0`

### RATE_LIMIT_BURST

Burst capacity for rate limiting (default: `20`).

Allows clients to temporarily exceed `RATE_LIMIT_REQUESTS_PER_SEC` up to the burst limit.

**Example:** With `RATE_LIMIT_REQUESTS_PER_SEC=10.0` and `RATE_LIMIT_BURST=20`, a client can make 20 requests instantly, then sustain 10 requests/second.

### Production presets (starting points)

| Profile | RATE_LIMIT_REQUESTS_PER_SEC | RATE_LIMIT_BURST | Typical use case |
| --- | --- | --- | --- |

| Conservative | `5.0` | `10` | Admin-heavy or sensitive workloads |
| Standard (default) | `10.0` | `20` | Most service-to-service integrations |
| High-throughput | `50.0` | `100` | High-volume internal API clients |

Tune based on observed `429` rates and client retry behavior.

### RATE_LIMIT_TOKEN_ENABLED

Enable per-IP rate limiting on token issuance endpoint `POST /v1/token` (default: `true`).

Use this protection to reduce credential stuffing and brute-force traffic on unauthenticated token
requests.

### RATE_LIMIT_TOKEN_REQUESTS_PER_SEC

Maximum token issuance requests per second per client IP (default: `5.0`).

### RATE_LIMIT_TOKEN_BURST

Burst capacity for token issuance per IP (default: `10`).

Allows short request spikes while preserving stricter controls for the unauthenticated token endpoint.

### Token endpoint presets (starting points)

| Profile | RATE_LIMIT_TOKEN_REQUESTS_PER_SEC | RATE_LIMIT_TOKEN_BURST | Typical use case |
| --- | --- | --- | --- |

| Strict (default) | `5.0` | `10` | Internet-facing token issuance |
| Shared-egress | `10.0` | `20` | Enterprise NAT/proxy callers |
| Internal trusted | `20.0` | `40` | Internal service mesh token broker |

Tune based on `POST /v1/token` `429` rates, NAT/proxy sharing patterns, and retry behavior.

## Account Lockout (PCI DSS 8.3.4)

Account lockout protects `POST /v1/token` against brute-force attacks by temporarily locking clients that exceed the failure threshold.

### LOCKOUT_MAX_ATTEMPTS

Number of consecutive failed authentication attempts before the client is locked (default: `10`).

PCI DSS 8.3.4 requires locking after ≤10 failed attempts; the default satisfies this requirement.

Set to `0` to disable lockout (not recommended for PCI DSS environments).

### LOCKOUT_DURATION_MINUTES

How long a locked client remains locked, in minutes (default: `30`).

PCI DSS 8.3.4 requires a minimum of 30 minutes; the default satisfies this requirement.

**Example:**

```dotenv
LOCKOUT_MAX_ATTEMPTS=10
LOCKOUT_DURATION_MINUTES=30
```

See [Authentication API: account lockout](api/auth/authentication.md#account-lockout) for behavior details and [Troubleshooting: 423 Locked](getting-started/troubleshooting.md#423-locked-account-lockout) for resolution steps.

## CORS configuration

### CORS_ENABLED

Enable Cross-Origin Resource Sharing (default: `false`).

**⚠️ Security Warning:** CORS is **disabled by default** because Secrets is designed as a server-to-server API. Enable only if browser-based access is required (e.g., single-page applications). Consider using a backend-for-frontend (BFF) pattern instead of exposing the API directly to browsers.

### CORS_ALLOW_ORIGINS

Comma-separated list of allowed origins for CORS requests.

**Security Best Practices:**

- Never use `*` (wildcard) in production

- List exact origins: `https://app.example.com,https://admin.example.com`

- Include protocol, domain, and port

- Review and prune origins quarterly

**Example:**

```dotenv
CORS_ENABLED=true
CORS_ALLOW_ORIGINS=https://app.example.com,https://admin.example.com

```

## Metrics configuration

### METRICS_ENABLED

Enable OpenTelemetry metrics collection (default: `true`).

- 📊 When enabled, exposes `/metrics` endpoint in Prometheus format

- 📉 When disabled, HTTP metrics middleware and `/metrics` route are disabled

**⚠️ Security Warning:** If metrics are enabled, restrict access to the `/metrics` endpoint using network policies or reverse proxy authentication. Never expose `/metrics` to the public internet.

### METRICS_NAMESPACE

Prefix for all metric names (default: `secrets`).

**Example:** With `METRICS_NAMESPACE=secrets`, metrics are named `secrets_http_requests_total`, `secrets_http_request_duration_seconds`, etc.

## Master key generation

```bash
./bin/app create-master-key --id default

# KMS mode (recommended for production)
./bin/app create-master-key --id default \
  --kms-provider=localsecrets \
  --kms-key-uri="base64key://<base64-32-byte-key>"

# Rotate master key (combines with existing MASTER_KEYS)
./bin/app rotate-master-key --id master-key-2026-08

```

Or with Docker image:

```bash
docker run --rm allisson/secrets create-master-key --id default

```

## See also

- [Security hardening guide](operations/security/hardening.md)

- [Production operations](operations/deployment/production.md)

- [Monitoring](operations/observability/monitoring.md)

- [Docker getting started](getting-started/docker.md)

- [Local development](getting-started/local-development.md)

- [Contributing guide](contributing.md#development-and-testing)
