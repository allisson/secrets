# 🛡️ Hardened Docker Deployment

> Last updated: 2026-02-25

This guide covers the mandatory security configurations for deploying Secrets in production environments.

## 1. Container Hardening (Distroless)

Secrets uses Google's `distroless/static-debian13` base image for maximum security (zero shell, no package manager). When deploying:

- **Run as Non-Root**: The container must run as UID `65532` (`nonroot`).
- **Read-Only Filesystem**: Always run with `--read-only` to prevent runtime tampering.
- **Drop Capabilities**: Drop all Linux capabilities natively.
- **No New Privileges**: Prevent escalation by setting `no-new-privileges:true`.

**Strict Docker Compose Example**:

```yaml
services:
  secrets-api:
    image: allisson/secrets:v0.13.0
    user: "65532:65532"
    read_only: true
    cap_drop: ["ALL"]
    security_opt: ["no-new-privileges:true"]
    tmpfs:
      - /tmp:rw,noexec,nosuid,size=10m
    restart: unless-stopped
```

*(See [`examples/deployment/docker-compose.prod.yml`](../../examples/deployment/docker-compose.prod.yml) for a complete stack).*

## 2. Network & Transport Security

- **Reverse Proxy**: Never expose port `8080` directly to the internet. Bind to `127.0.0.1` and use a reverse proxy (e.g., Nginx, Envoy) for TLS termination.
- **Internal Only**: Keep the database and the `/metrics` endpoints strictly internal (use VPCs or Docker internal networks).
- **TLS Configuration**: Use TLS 1.2+ with strong ciphers (e.g. `ECDHE-ECDSA-AES256-GCM-SHA384`).

## 3. Database Security

- **Encrypted Connections**: Always connect to the database using TLS.
  - PostgreSQL: `sslmode=require` or `sslmode=verify-full`.
  - MySQL: `tls=true` or `tls=custom`.
- **Least Privilege**: The runtime database user must only have DML (`SELECT`, `INSERT`, `UPDATE`, `DELETE`) privileges. Use a separate user for schema migrations.

## 4. Abuse Prevention (Rate Limiting & Lockout)

Prevent brute-force and DoS attacks by enforcing limits:

- **Per-IP Token Limit**: `RATE_LIMIT_TOKEN_ENABLED=true` (Default: 5 req/sec).
- **Client Account Lockout**: `LOCKOUT_MAX_ATTEMPTS=10` (Default: 30 minutes).
- **Authenticated API Limit**: `RATE_LIMIT_ENABLED=true` (Default: 10 req/sec).

> **Important Setup Note**: If behind a reverse proxy, ensure standard headers like `X-Forwarded-For` are configured correctly so rate-limiting captures the true client IP instead of the proxy IP.

## 5. Secret Management

- **Inject via Environment**: Never hardcode credentials into the Dockerfile or commit files like `.env`.
- **Master Keys**: Use external KMS providers (AWS KMS, GCP KMS) in production rather than raw plaintext keys.
- **Volume Permissions**: Since Secrets runs as UID `65532`, any bind-mounted directories on the host must be owned by `65532`. (Using named Docker volumes natively resolves this).

## Deployment Checklist

- [ ] Immutable SHA256 image digest pinned
- [ ] Container capabilities dropped (`cap_drop: ALL`)
- [ ] Filesystem read-only
- [ ] HTTPS enforced at reverse proxy
- [ ] Database TLS enabled
- [ ] Master Keys stored in KMS mapping
- [ ] Audit logs exported to a secure SIEM/aggregator
