# 🔒 Security Hardening Guide

> Last updated: 2026-02-21

This guide covers comprehensive security hardening for production deployments of Secrets. These measures are essential for protecting sensitive data and maintaining operational security.

## 📑 Table of Contents

- [1) Transport Layer Security (TLS/HTTPS)](#1-transport-layer-security-tlshttps)
- [2) Database Security](#2-database-security)
- [3) Network Security](#3-network-security)
- [4) Rate Limiting](#4-rate-limiting)
- [5) Cross-Origin Resource Sharing (CORS)](#5-cross-origin-resource-sharing-cors)
- [6) Authentication and Token Management](#6-authentication-and-token-management)
- [7) Master Key Storage and Management](#7-master-key-storage-and-management)
- [8) Audit Logging and Monitoring](#8-audit-logging-and-monitoring)
- [9) Security Checklist](#9-security-checklist)

## 1) Transport Layer Security (TLS/HTTPS)

### Requirements

Secrets **must** run behind a reverse proxy that handles TLS termination. The application does not provide built-in TLS/HTTPS support by design.

### Reverse Proxy Configuration

**Supported reverse proxies:**

- Nginx
- Envoy
- Traefik
- HAProxy
- Cloud load balancers (AWS ALB/NLB, GCP Load Balancer, Azure Application Gateway)

**Minimum TLS configuration:**

```nginx
# Nginx example
server {
    listen 443 ssl http2;
    server_name secrets.example.com;

    # TLS certificate configuration
    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    # Modern TLS configuration
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384';
    ssl_prefer_server_ciphers off;

    # Security headers
    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;

    # Proxy to Secrets application
    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header X-Request-Id $request_id;

        # Timeouts and limits
        proxy_connect_timeout 5s;
        proxy_send_timeout 15s;
        proxy_read_timeout 15s;
        client_max_body_size 1m;
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name secrets.example.com;
    return 301 https://$server_name$request_uri;
}
```

**TLS certificate management:**

- Use automated certificate renewal (Let's Encrypt, cert-manager, ACM)
- Monitor certificate expiration (alert at 30 days remaining)
- Use strong private key protection (file permissions, HSM, KMS)
- Rotate certificates according to your security policy

### TLS Best Practices

1. **Protocol versions:** Use TLS 1.2 and TLS 1.3 only
2. **Cipher suites:** Prefer AEAD ciphers (GCM, ChaCha20-Poly1305)
3. **HSTS:** Enable HTTP Strict Transport Security with long max-age
4. **Certificate validation:** Use valid, non-self-signed certificates in production
5. **Forward secrecy:** Ensure cipher suites support perfect forward secrecy (PFS)

## 2) Database Security

### SSL/TLS Configuration

**PostgreSQL production connection string:**

```dotenv
# Required for production - encrypted connection
DB_CONNECTION_STRING=postgres://user:password@db.example.com:5432/secrets?sslmode=require

# Recommended - encrypted connection with certificate verification
DB_CONNECTION_STRING=postgres://user:password@db.example.com:5432/secrets?sslmode=verify-full&sslrootcert=/path/to/ca.crt
```

**MySQL production connection string:**

```dotenv
# Required for production - encrypted connection
DB_CONNECTION_STRING=user:password@tcp(db.example.com:3306)/secrets?tls=true

# Recommended - encrypted connection with certificate verification
DB_CONNECTION_STRING=user:password@tcp(db.example.com:3306)/secrets?tls=custom
```

**SSL mode comparison:**

| Mode | PostgreSQL | MySQL | Use Case |
| --- | --- | --- | --- |
| No encryption | `sslmode=disable` | `tls=false` | **Development only** |
| Encrypted | `sslmode=require` | `tls=true` | **Minimum for production** |
| Verified | `sslmode=verify-full` | `tls=custom` | **Recommended for production** |

**Warning:** `sslmode=disable` and `tls=false` transmit credentials and data in plaintext. Never use in production.

### Database Access Control

1. **Network isolation:**
   - Restrict database access to application servers only
   - Use VPC/VNET private subnets
   - Configure database firewall rules
   - Disable public internet access

2. **Authentication:**
   - Use strong, unique passwords (minimum 32 characters)
   - Rotate database credentials periodically
   - Use IAM authentication where available (AWS RDS, GCP Cloud SQL)
   - Disable default/test accounts

3. **Authorization:**
   - Grant minimum required privileges to application user
   - Use separate users for migrations and runtime operations
   - Restrict administrative access to trusted networks/users

4. **Encryption at rest:**
   - Enable database encryption at rest (LUKS, dm-crypt, cloud provider encryption)
   - Verify encrypted storage for backups
   - Use separate encryption keys per environment

### Database Hardening

```sql
-- PostgreSQL: Create application user with minimal privileges
CREATE USER secrets_app WITH PASSWORD 'strong_random_password';
GRANT CONNECT ON DATABASE secrets TO secrets_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO secrets_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO secrets_app;

-- PostgreSQL: Create migration user (separate from runtime user)
CREATE USER secrets_migrate WITH PASSWORD 'different_strong_password';
GRANT ALL PRIVILEGES ON DATABASE secrets TO secrets_migrate;
```

```sql
-- MySQL: Create application user with minimal privileges
CREATE USER 'secrets_app'@'%' IDENTIFIED BY 'strong_random_password';
GRANT SELECT, INSERT, UPDATE, DELETE ON secrets.* TO 'secrets_app'@'%';

-- MySQL: Create migration user (separate from runtime user)
CREATE USER 'secrets_migrate'@'%' IDENTIFIED BY 'different_strong_password';
GRANT ALL PRIVILEGES ON secrets.* TO 'secrets_migrate'@'%';
```

## 3) Network Security

### Firewall Rules

**Minimum ingress rules:**

| Port | Protocol | Source | Purpose |
| --- | --- | --- | --- |
| 443 | TCP | Internet/internal | HTTPS (reverse proxy) |
| 8080 | TCP | Reverse proxy only | Application server |
| 5432/3306 | TCP | Application servers | Database |

**Block all other inbound traffic by default.**

### Metrics Endpoint Protection

The `/metrics` endpoint exposes operational metrics that may contain sensitive information.

**Security measures:**

1. **Network restriction (recommended):**

   ```nginx
   # Nginx: Restrict /metrics to internal monitoring network
   location /metrics {
       allow 10.0.0.0/8;      # Internal network
       allow 172.16.0.0/12;   # Docker networks
       deny all;
       proxy_pass http://127.0.0.1:8080;
   }
   ```

2. **Authentication (alternative):**

   ```nginx
   # Nginx: Require basic auth for /metrics
   location /metrics {
       auth_basic "Metrics";
       auth_basic_user_file /etc/nginx/.htpasswd;
       proxy_pass http://127.0.0.1:8080;
   }
   ```

3. **Disable metrics (if unused):**

   ```dotenv
   METRICS_ENABLED=false
   ```

**Never expose `/metrics` to the public internet.**

### Internal Service Communication

1. **Use private networks:**
   - Deploy application and database in private subnets
   - Use security groups/network policies to restrict traffic
   - Avoid exposing internal services to public networks

2. **Service mesh (optional):**
   - Consider Istio, Linkerd, or Consul for mTLS between services
   - Enforce zero-trust networking policies
   - Enable distributed tracing for audit trails

## 4) Rate Limiting

Rate limiting protects against abuse, brute force attacks, and denial of service.

### Authenticated Endpoint Configuration

```dotenv
# Enable rate limiting (default: true)
RATE_LIMIT_ENABLED=true

# Requests per second per authenticated client (default: 10.0)
RATE_LIMIT_REQUESTS_PER_SEC=10.0

# Burst capacity (default: 20)
RATE_LIMIT_BURST=20
```

### How It Works

- **Scope:** Per-authenticated-client (not per-IP)
- **Algorithm:** Token bucket with automatic refill
- **Enforcement:** Applied after authentication, before handler execution
- **Response:** HTTP 429 with `Retry-After` header when limit exceeded

### Recommended Settings

| Workload | Requests/sec | Burst | Reasoning |
| --- | --- | --- | --- |
| High-volume API | 50.0 | 100 | Batch processing, high throughput |
| Standard application | 10.0 | 20 | **Default - suitable for most use cases** |
| Sensitive operations | 1.0 | 5 | Key rotation, admin operations |

### Excluded Endpoints

Authenticated per-client rate limiting does **not** apply to:

- `/health` - Health checks
- `/ready` - Readiness probes
- `/metrics` - Metrics collection

### Token Endpoint Configuration (IP-based)

```dotenv
# Enable token endpoint rate limiting (default: true)
RATE_LIMIT_TOKEN_ENABLED=true

# Requests per second per IP for POST /v1/token (default: 5.0)
RATE_LIMIT_TOKEN_REQUESTS_PER_SEC=5.0

# Burst capacity per IP (default: 10)
RATE_LIMIT_TOKEN_BURST=10
```

### Token Endpoint Notes

- **Scope:** Per-client-IP for unauthenticated `POST /v1/token`
- **Purpose:** Mitigate credential stuffing and brute-force token issuance attempts
- **Response:** HTTP `429` with `Retry-After` when exceeded
- **Operational caveat:** Shared NAT/proxy egress can require tuning `RATE_LIMIT_TOKEN_*`

### Trusted Proxy and IP Forwarding Safety

- Configure trusted proxies explicitly in production; do not trust arbitrary forwarded headers
- Ensure your edge proxy/load balancer sets client IP headers consistently
- If trusted proxy settings are incorrect, all token requests can appear from one IP and trigger false `429`
- If headers are over-trusted, attackers can spoof forwarded IPs to evade per-IP controls
- Use [Trusted proxy configuration](#trusted-proxy-configuration) for validation workflow and platform notes

### Tuning Guidance

**If you observe legitimate 429 responses:**

1. Review client request patterns in audit logs
2. Identify if requests can be batched or optimized
3. Increase `RATE_LIMIT_REQUESTS_PER_SEC` if sustained higher rates are justified
4. Increase `RATE_LIMIT_BURST` if traffic is bursty but averages within limits

**For defense-in-depth:**

- Combine application rate limiting with reverse proxy rate limiting
- Use reverse proxy for IP-based rate limiting
- Use application rate limiting for client-based rate limiting

## 5) Cross-Origin Resource Sharing (CORS)

Secrets is designed as a server-to-server API. CORS is **disabled by default** and should remain disabled for most deployments.

### When to Enable CORS

Enable CORS **only** if you need browser-based access to the API:

- Single-page applications (SPA) accessing Secrets directly
- Web-based admin interfaces
- Browser extensions

**Security note:** Exposing the Secrets API to browsers increases attack surface. Consider using a backend-for-frontend (BFF) pattern instead.

### Configuration

```dotenv
# Disable CORS (default: false - recommended)
CORS_ENABLED=false

# Enable CORS only if required
CORS_ENABLED=true
CORS_ALLOW_ORIGINS=https://app.example.com,https://admin.example.com
```

### CORS Best Practices

1. **Explicit origins only:**
   - Never use `*` (wildcard) in production
   - List exact origins (protocol + domain + port)
   - Validate origins match your application domains

2. **Minimal origin list:**
   - Include only origins that require access
   - Remove origins when no longer needed
   - Audit origin list quarterly

3. **Combined with authentication:**
   - CORS does not replace authentication
   - Always require Bearer token authentication
   - Use short-lived tokens (see next section)

**Example secure configuration:**

```dotenv
CORS_ENABLED=true
CORS_ALLOW_ORIGINS=https://admin.example.com
AUTH_TOKEN_EXPIRATION_SECONDS=3600  # 1 hour for browser-based access
```

## 6) Authentication and Token Management

### Token Expiration

#### Default token expiration: 4 hours (14400 seconds)

```dotenv
# Default (recommended for most deployments)
AUTH_TOKEN_EXPIRATION_SECONDS=14400

# High-security environments (1 hour)
AUTH_TOKEN_EXPIRATION_SECONDS=3600

# Low-security environments (24 hours)
AUTH_TOKEN_EXPIRATION_SECONDS=86400
```

**Migration note:** Prior to v0.5.0, the default was 24 hours (86400 seconds). Review your token expiration settings and client refresh logic when upgrading.

### Token Lifecycle Best Practices

1. **Token rotation:**
   - Implement token refresh logic in clients
   - Request new tokens before expiration
   - Handle 401 responses gracefully

2. **Token revocation:**
   - Deactivate clients immediately upon compromise
   - Revoke tokens when client credentials rotate
   - Audit active tokens periodically

3. **Token storage:**
   - Never log tokens in plaintext
   - Store tokens securely in client applications
   - Use environment variables or secrets managers
   - Never commit tokens to source control

### Client Management

1. **Least privilege policies:**
   - Grant minimum required capabilities per client
   - Use path restrictions to limit access scope
   - Review and prune unused policies quarterly

2. **Client credentials:**
   - Generate strong random secrets (use `/v1/token` endpoint)
   - Rotate client credentials on personnel changes
   - Use separate clients per application/environment

3. **Client lifecycle:**
   - Deactivate unused clients immediately
   - Monitor client usage via audit logs
   - Delete obsolete clients after deactivation period

**Example policy (least privilege):**

```json
{
  "policies": [
    {
      "path": "/v1/secrets/app/production/*",
      "capabilities": ["read", "write"]
    },
    {
      "path": "/v1/transit/keys/payment/encrypt",
      "capabilities": ["encrypt"]
    }
  ]
}
```

## 7) Master Key Storage and Management

Master keys are the root of trust in the envelope encryption hierarchy. Protect them accordingly.

### Storage Requirements

**Never:**

- Commit master keys to source control
- Include master keys in container images
- Store master keys in application configuration files
- Share master keys across environments
- Log master keys in plaintext

**Always:**

- Use environment variables for runtime injection
- Store master keys in secrets management systems
- Use distinct master keys per environment
- Encrypt master keys at rest
- Audit master key access

### Recommended Storage Solutions

| Solution | Use Case | Notes |
| --- | --- | --- |
| AWS Secrets Manager | AWS deployments | Use IAM roles for access control |
| GCP Secret Manager | GCP deployments | Use workload identity for access |
| Azure Key Vault | Azure deployments | Use managed identities for access |
| HashiCorp Vault | Multi-cloud/on-prem | Use AppRole or token auth |

### Master Key Rotation

1. **Generate new master key:**

   ```bash
   ./bin/app create-master-key --id master-key-2026-02
   ```

2. **Add new key to master key chain:**

   ```dotenv
   MASTER_KEYS=master-key-2026-01:OLD_BASE64_KEY,master-key-2026-02:NEW_BASE64_KEY
   ACTIVE_MASTER_KEY_ID=master-key-2026-02
   ```

3. **Restart all application servers:**
   - Use rolling restart to avoid downtime
   - Verify `/ready` endpoint after each restart
   - Confirm new KEKs use new master key ID

4. **Rotate KEKs encrypted with old master key:**

   ```bash
   ./bin/app rotate-kek --algorithm aes-gcm
   ```

5. **Remove old master key after migration period:**

   ```dotenv
   MASTER_KEYS=master-key-2026-02:NEW_BASE64_KEY
   ACTIVE_MASTER_KEY_ID=master-key-2026-02
   ```

**Recommended rotation schedule:**

- Routine rotation: Annually or per organizational policy
- Immediate rotation: Upon suspected compromise
- Audit rotation: Quarterly review of master key usage

### Master Key Generation

**Use the built-in generator:**

```bash
# Generate 32-byte (256-bit) master key
./bin/app create-master-key --id default

# Output format
MASTER_KEYS=default:A1B2C3D4E5F6G7H8I9J0K1L2M3N4O5P6Q7R8S9T0U1V2W3X4Y5Z6==
```

**Key properties:**

- Exactly 32 bytes (256 bits) of cryptographically secure random data
- Base64-encoded for safe environment variable storage
- Generated using `crypto/rand` (CSPRNG)

## 8) Audit Logging and Monitoring

### Audit Log Configuration

Audit logs record all API operations for security analysis and compliance.

**Coverage:**

- All authenticated requests (success and failure)
- Client identity and request path
- Timestamp, method, status code, duration
- Capability enforcement results

### Audit Log Retention

#### Recommended retention: 90 days

```bash
# Monthly cleanup routine

# 1) Preview audit logs older than 90 days
./bin/app clean-audit-logs --days 90 --dry-run --format json

# 2) Execute deletion
./bin/app clean-audit-logs --days 90 --format text
```

Adjust retention based on:

- Compliance requirements (SOC 2, PCI-DSS, HIPAA)
- Incident response window
- Storage capacity
- Forensic analysis needs

### Security Monitoring

**Alert on:**

1. **Authentication failures:**
   - Repeated 401 responses from same client/IP
   - Invalid token attempts
   - Threshold: 5 failures in 5 minutes

2. **Authorization failures:**
   - Repeated 403 responses from same client
   - Capability denied patterns
   - Threshold: 10 denials in 10 minutes

3. **Rate limiting:**
   - Frequent 429 responses
   - Potential abuse or misconfigured clients
   - Threshold: 100 rate limits in 1 hour

4. **Anomalous patterns:**
   - Client accessing new paths after long idle period
   - Unusual request volume from single client
   - Access outside normal business hours

5. **System health:**
   - Elevated error rates (5xx responses)
   - Database connection failures
   - Slow response times (p95 > 1s)

### Metrics Collection

**Key metrics to track:**

```promql
# Request rate by endpoint and status
rate(secrets_http_requests_total[5m])

# Request latency percentiles
histogram_quantile(0.95, secrets_http_request_duration_seconds)

# Error rate
rate(secrets_http_requests_total{status=~"5.."}[5m])

# Authentication failures
rate(secrets_http_requests_total{status="401"}[5m])

# Authorization failures
rate(secrets_http_requests_total{status="403"}[5m])

# Rate limit hits
rate(secrets_http_requests_total{status="429"}[5m])
```

**Example Prometheus alerts:**

```yaml
groups:
  - name: secrets_security
    rules:
      - alert: HighAuthFailureRate
        expr: rate(secrets_http_requests_total{status="401"}[5m]) > 0.1
        for: 5m
        annotations:
          summary: "High authentication failure rate detected"

      - alert: HighAuthzFailureRate
        expr: rate(secrets_http_requests_total{status="403"}[5m]) > 0.2
        for: 10m
        annotations:
          summary: "High authorization failure rate detected"

      - alert: RateLimitExceeded
        expr: rate(secrets_http_requests_total{status="429"}[5m]) > 1
        for: 5m
        annotations:
          summary: "Clients hitting rate limits frequently"
```

### Log Forwarding

**Forward logs to SIEM/log aggregation:**

- Splunk, Elasticsearch, Datadog, CloudWatch Logs
- Centralize logs from all application instances
- Correlate with network and infrastructure logs
- Enable long-term retention for compliance

**Structured logging:**

- Logs are JSON-formatted with consistent fields
- Request ID (`request_id`) for distributed tracing
- Client ID for cross-referencing with audit logs
- Timestamp in UTC for accurate correlation

## 9) Security Checklist

Use this checklist for production deployment validation.

### Transport Security

- [ ] HTTPS enforced via reverse proxy
- [ ] TLS 1.2+ configured
- [ ] HSTS header enabled
- [ ] HTTP to HTTPS redirect active
- [ ] Valid TLS certificate installed
- [ ] Certificate expiration monitoring configured

### Database Security

- [ ] Database SSL/TLS enabled (`sslmode=require` or `tls=true`)
- [ ] Database credentials rotated and stored securely
- [ ] Database access restricted to application network
- [ ] Encryption at rest enabled
- [ ] Database backups encrypted
- [ ] Minimal database privileges granted

### Network Security

- [ ] Firewall rules restrict inbound traffic
- [ ] `/metrics` endpoint not publicly accessible
- [ ] Database port not exposed to internet
- [ ] Application deployed in private subnet
- [ ] Security groups/network policies configured

### Authentication and Authorization

- [ ] Token expiration configured appropriately
- [ ] Client policies follow least privilege principle
- [ ] Default/test clients disabled or deleted
- [ ] Client credentials stored securely
- [ ] Rate limiting enabled
- [ ] CORS disabled (or explicitly required and configured)

### Master Key Management

- [ ] Master keys stored in secrets manager (not source control)
- [ ] Distinct master keys per environment
- [ ] Master key access audited
- [ ] Master key rotation schedule documented
- [ ] `MASTER_KEYS` not in container images

### Monitoring and Logging

- [ ] Audit log retention policy defined
- [ ] Security alerts configured
- [ ] Metrics collection enabled
- [ ] Log forwarding to SIEM configured
- [ ] Incident response runbook documented

### Operational Security

- [ ] Backup and restore tested
- [ ] Key rotation procedure documented and tested
- [ ] Incident response plan defined
- [ ] On-call contacts documented
- [ ] Security review scheduled (quarterly)

## Trusted Proxy Configuration

Use this section to validate source-IP forwarding for security controls that depend on caller IP
(for example token endpoint per-IP rate limiting on `POST /v1/token`).

### Why this matters

- If proxy trust is too broad, attackers may spoof `X-Forwarded-For`
- If proxy trust is too narrow/incorrect, many clients can collapse into one apparent IP
- Both cases can invalidate per-IP rate-limiting behavior

### Validation checklist

1. Only trusted edge proxies can set forwarded client-IP headers
2. Untrusted internet clients cannot inject arbitrary `X-Forwarded-For`
3. App-observed `client_ip` matches edge-proxy access logs for sampled requests
4. Multi-hop proxy behavior (if any) is documented and tested

### Nginx baseline forwarding

```nginx
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

Hardening notes:

- Do not accept forwarded headers directly from public clients
- Ensure only your reverse-proxy tier can reach application port `8080`

### AWS ALB / ELB notes

- ALB injects `X-Forwarded-For`; keep app reachable only from ALB/security group path
- Validate that downstream proxies preserve rather than overwrite trusted header chain
- Sample and compare ALB access logs with app `client_ip` logs

### Cloudflare / CDN edge notes

- Prefer single trusted edge path to origin
- If using CDN-specific client IP headers, keep mapping and validation documented
- Reject direct origin traffic from non-edge sources where possible

### Diagnostic quick test

1. Send a test request through edge proxy
2. Capture edge log source IP
3. Capture app log `client_ip` and request ID
4. Confirm both values refer to the same caller context

### Common failure patterns

- **All token requests share one IP:** likely NAT/proxy collapse or missing forwarded IP propagation
- **Frequent token `429` after proxy changes:** trust chain or source-IP extraction behavior drifted
- **Suspiciously diverse token caller IPs from one source:** potential forwarded-header spoofing

## See also

- [Production deployment guide](../deployment/production.md)
- [Environment variables](../../configuration.md)
- [Security model](../../concepts/security-model.md)
- [Monitoring](../observability/monitoring.md)
- [Policy management](../../api/auth/policies.md)
- [Troubleshooting](../../getting-started/troubleshooting.md)
