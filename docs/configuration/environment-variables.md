# ⚙️ Environment Variables

> Last updated: 2026-02-19

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
MASTER_KEYS=default:BASE64_32_BYTE_KEY
ACTIVE_MASTER_KEY_ID=default

# Authentication configuration
AUTH_TOKEN_EXPIRATION_SECONDS=14400

# Rate limiting configuration
RATE_LIMIT_ENABLED=true
RATE_LIMIT_REQUESTS_PER_SEC=10.0
RATE_LIMIT_BURST=20

# CORS configuration
CORS_ENABLED=false
CORS_ALLOW_ORIGINS=

# Metrics configuration
METRICS_ENABLED=true
METRICS_NAMESPACE=secrets
```

## Database configuration

### DB_DRIVER
Database driver to use. Supported values: `postgres`, `mysql`.

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

See [Security Hardening Guide](../operations/security-hardening.md#2-database-security) for complete guidance.

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
Comma-separated list of master keys in format `id1:base64key1,id2:base64key2`.

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

## Authentication configuration

### AUTH_TOKEN_EXPIRATION_SECONDS
Token expiration time in seconds (default: `14400` - 4 hours).

**⚠️ Migration Note:** Prior to v0.5.0, the default was 86400 seconds (24 hours). Review your token expiration settings and client refresh logic when upgrading from v0.4.x.

**Recommended settings:**
- High-security environments: `3600` (1 hour)
- Standard deployments: `14400` (4 hours) - **default**
- Low-security environments: `86400` (24 hours)

## Rate limiting configuration

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
```

Or with Docker image:

```bash
docker run --rm allisson/secrets:latest create-master-key --id default
```

## See also

- [Security hardening guide](../operations/security-hardening.md)
- [Production operations](../operations/production.md)
- [Monitoring](../operations/monitoring.md)
- [Docker getting started](../getting-started/docker.md)
- [Local development](../getting-started/local-development.md)
- [Testing guide](../development/testing.md)
