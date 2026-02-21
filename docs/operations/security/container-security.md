# 🐳 Container Security Guide

> Last updated: 2026-02-21

This guide covers comprehensive container security best practices for running Secrets in production environments. It focuses on Docker-specific security hardening, image security, runtime protection, and deployment patterns for Docker Standalone and Docker Compose.

## 📑 Table of Contents

- [Quick Start](#quick-start)

- [1) Base Image Security](#1-base-image-security)

- [2) Runtime Security](#2-runtime-security)

- [3) Network Security](#3-network-security)

- [4) Secrets Management](#4-secrets-management)

- [5) Image Scanning](#5-image-scanning)

- [6) Health Checks and Observability](#6-health-checks-and-observability)

- [7) Build Security](#7-build-security)

- [8) Deployment Best Practices](#8-deployment-best-practices)

- [9) Security Checklist](#9-security-checklist)

## Quick Start

**🎯 Goal**: Deploy Secrets with production-grade security in < 15 minutes.

This quick start provides copy-paste commands for secure deployments. For detailed explanations, see the full sections below.

### Prerequisites

- Docker 20.10+

- Basic understanding of container security

- Access to container registry (Docker Hub, GCR, ECR, etc.)

### Option 1: Secure Docker Deployment (5 minutes)

```bash
# 1. Pull latest image
docker pull allisson/secrets:v0.10.0

# 2. Scan for vulnerabilities (optional but recommended)
docker scout cves allisson/secrets:v0.10.0
# or: trivy image allisson/secrets:v0.10.0

# 3. Create network
docker network create secrets-net

# 4. Start database with security hardening
docker run -d --name secrets-db \
  --network secrets-net \
  --cap-drop=ALL \
  --cap-add=CHOWN --cap-add=SETUID --cap-add=SETGID --cap-add=DAC_OVERRIDE \
  --read-only \
  --tmpfs /tmp \
  --tmpfs /var/run/postgresql \
  -e POSTGRES_USER=secrets \
  -e POSTGRES_PASSWORD=secure_password_here \
  -e POSTGRES_DB=secrets \
  -v postgres-data:/var/lib/postgresql/data \
  postgres:16-alpine

# 5. Create secure .env file (don't commit to git!)
cat > .env <<EOF
DB_DRIVER=postgres
DB_CONNECTION_STRING=postgres://secrets:secure_password_here@secrets-db:5432/secrets?sslmode=require
MASTER_KEYS=default:$(openssl rand -base64 32)
ACTIVE_MASTER_KEY_ID=default
EOF

# 6. Run migrations
docker run --rm \
  --network secrets-net \
  --env-file .env \
  allisson/secrets:v0.10.0 migrate

# 7. Create encryption key
docker run --rm \
  --network secrets-net \
  --env-file .env \
  allisson/secrets:v0.10.0 create-kek --algorithm aes-gcm

# 8. Start API with security hardening
docker run -d --name secrets-api \
  --network secrets-net \
  --env-file .env \
  --cap-drop=ALL \
  --read-only \
  --security-opt=no-new-privileges:true \
  --pids-limit=100 \
  -p 127.0.0.1:8080:8080 \
  allisson/secrets:v0.10.0 server

# 9. Verify deployment
curl http://127.0.0.1:8080/health
# Expected: {"status":"healthy"}

curl http://127.0.0.1:8080/ready
# Expected: {"status":"ready","database":"ok"}

```

**Security features applied**:

- ✅ Non-root user (UID 65532)

- ✅ Read-only filesystem

- ✅ All capabilities dropped

- ✅ No new privileges

- ✅ Process limit (100)

- ✅ Localhost-only binding (not exposed to external network)

### Option 2: Secure Docker Compose Deployment (7 minutes)

```yaml
# docker-compose.yml
version: '3.8'

services:
  db:
    image: postgres:16-alpine
    container_name: secrets-db
    environment:
      POSTGRES_DB: secrets
      POSTGRES_USER: secrets
      POSTGRES_PASSWORD: ${DB_PASSWORD}  # Set in .env file
    volumes:
      - postgres-data:/var/lib/postgresql/data

    networks:
      - secrets-net

    cap_drop:
      - ALL

    cap_add:
      - CHOWN

      - SETUID

      - SETGID

      - DAC_OVERRIDE

    read_only: true
    tmpfs:
      - /tmp

      - /var/run/postgresql

    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U secrets -d secrets"]
      interval: 10s
      timeout: 3s
      retries: 3

  secrets-api:
    image: allisson/secrets:v0.10.0
    container_name: secrets-api
    environment:
      DB_DRIVER: postgres
      DB_CONNECTION_STRING: postgres://secrets:${DB_PASSWORD}@db:5432/secrets?sslmode=require
      MASTER_KEYS: ${MASTER_KEYS}
      ACTIVE_MASTER_KEY_ID: default
    ports:
      - "127.0.0.1:8080:8080"  # Localhost only

    networks:
      - secrets-net

    depends_on:
      db:
        condition: service_healthy
    cap_drop:
      - ALL

    read_only: true
    security_opt:
      - no-new-privileges:true

    pids_limit: 100
    restart: unless-stopped

  # Healthcheck sidecar (distroless has no curl)
  healthcheck:
    image: curlimages/curl:latest
    container_name: secrets-healthcheck
    command: >
      sh -c 'while true; do
        curl -f http://secrets-api:8080/health || exit 1;
        sleep 30;
      done'
    networks:
      - secrets-net

    depends_on:
      - secrets-api

    restart: unless-stopped

volumes:
  postgres-data:

networks:
  secrets-net:
    driver: bridge

```

```bash
# .env file (don't commit to git!)
DB_PASSWORD=your_secure_password_here
MASTER_KEYS=default:your_base64_encoded_32_byte_key_here

```

```bash
# Deploy
docker-compose up -d

# Verify
docker-compose ps
curl http://localhost:8080/health

```

**Security features applied**:

- ✅ Read-only filesystem

- ✅ Capabilities dropped

- ✅ No new privileges

- ✅ Process limits

- ✅ Localhost-only binding

- ✅ Named volumes (no permission issues)

- ✅ Health monitoring

- ✅ Automatic restart

### Next Steps After Quick Start

Once deployed, complete these additional security hardening steps:

1. **Enable TLS** - See [Network Security](#3-network-security)

2. **Set up monitoring** - See [Health Checks](#6-health-checks-and-observability)

3. **Configure network policies** - See [Network Security](#3-network-security)

4. **Regular security scans** - See [Image Scanning](#5-image-scanning)

5. **Review security checklist** - See [Security Checklist](#9-security-checklist)

### Security Validation

After deployment, verify security posture:

```bash
# Docker: Check user is non-root
docker exec secrets-api id
# Expected: uid=65532(nonroot) gid=65532(nonroot)

# Docker: Verify read-only filesystem
docker exec secrets-api touch /test 2>&1
# Expected: "touch: /test: Read-only file system"

# Docker: Check capabilities
docker inspect secrets-api --format='{{.HostConfig.CapDrop}}'
# Expected: [ALL]

# Test health endpoints
curl http://localhost:8080/health
curl http://localhost:8080/ready

```

**Troubleshooting quick start issues**: See [Troubleshooting Guide](../../getting-started/troubleshooting.md).

---

## 1) Base Image Security

### Why Distroless?

Starting in **v0.10.0**, Secrets uses Google's [Distroless](https://github.com/GoogleContainerTools/distroless) base image for enhanced security:

**Security Benefits:**

- **Minimal attack surface**: No shell, package manager, or system utilities

- **Reduced CVE exposure**: Only includes runtime dependencies (glibc, CA certs, tzdata)

- **Regular security patches**: Maintained by Google with automated updates

- **Better CVE scanning**: Known base image with comprehensive vulnerability databases

- **Non-root by default**: Runs as UID 65532 (`nonroot:nonroot`)

**Comparison:**

| Base Image | Size | Shell | Package Manager | CVE Database | User |
|------------|------|-------|-----------------|--------------|------|
| `scratch` | ~0MB | No | No | Poor | root |
| `alpine` | ~5MB | Yes | apk | Good | root |
| `debian:slim` | ~70MB | Yes | apt | Excellent | root |
| **`distroless/static`** | **~2MB** | **No** | **No** | **Excellent** | **nonroot** |

### SHA256 Digest Pinning

Secrets uses **SHA256 digest pinning** for immutable builds:

```dockerfile
FROM gcr.io/distroless/static-debian13@sha256:d90359c7a3ad67b3c11ca44fd5f3f5208cbef546f2e692b0dc3410a869de46bf

```

**Benefits:**

- ✅ **Immutability**: Prevents supply chain attacks via tag poisoning

- ✅ **Reproducibility**: Same digest always produces identical builds

- ✅ **Auditability**: Exact base image version is traceable

**Updating Digests:**

When Google releases security patches, update the digest manually:

```bash
# Pull latest distroless image
docker pull gcr.io/distroless/static-debian13:latest

# Get new digest
docker inspect gcr.io/distroless/static-debian13:latest --format='{{index .RepoDigests 0}}'

# Update Dockerfile with new SHA256 digest
# Test build and security scan before committing

```

### Security Update Strategy

**Recommended schedule:**

- **Critical vulnerabilities**: Immediate update (within 24 hours)

- **High severity**: Weekly update (every Monday)

- **Medium/Low severity**: Monthly update (1st of each month)

**Automated monitoring:**

Use [Renovate](https://github.com/renovatebot/renovate) or [Dependabot](https://github.com/dependabot) to monitor base image updates:

```json
// renovate.json
{
  "extends": ["config:base"],
  "dockerfile": {
    "enabled": true,
    "pinDigests": true
  }
}

```

**Migrating to distroless?** If you're currently using Alpine, scratch, or Debian base images and want to migrate to distroless, see the comprehensive [Base Image Migration Guide](../deployment/base-image-migration.md).

## 2) Runtime Security

### Non-Root User Execution

Secrets **requires** running as non-root user (UID 65532):

```bash
docker run --rm \
  --user 65532:65532 \
  --read-only \
  --cap-drop=ALL \
  --security-opt=no-new-privileges:true \
  allisson/secrets:v0.10.0 server

```

#### Volume Permissions

When mounting host directories or persistent volumes, ensure they're readable/writable by UID 65532 (nonroot user).

**Common issue**: After upgrading to v0.10.0, volume permission errors occur because the non-root user cannot access directories owned by root or other users.

**Quick check**:

```bash
# Verify container runs as UID 65532
docker run --rm allisson/secrets:v0.10.0 id
# uid=65532(nonroot) gid=65532(nonroot)

# Check volume permissions (should be owned by 65532)
ls -la /path/to/volume

```

**Solutions**:

1. **Docker - Named volumes** (recommended):

   ```bash
   docker volume create secrets-data
   docker run -v secrets-data:/data allisson/secrets:v0.10.0
   # Docker automatically sets correct permissions
   ```

2. **Docker - Host directory**:

   ```bash
   sudo chown -R 65532:65532 /path/to/host/dir
   docker run -v /path/to/host/dir:/data allisson/secrets:v0.10.0
   ```

**For comprehensive troubleshooting**, see:

- [Volume Permission Troubleshooting Guide](../troubleshooting/volume-permissions.md)

### Read-Only Filesystem

Secrets supports **read-only root filesystem** (no writes at runtime):

```bash
# Docker
docker run --rm --read-only \
  --tmpfs /tmp:rw,noexec,nosuid,size=10m \
  allisson/secrets:v0.10.0 server

```

**Read-only filesystem behavior**:

- **No runtime writes**: The application binary is stateless and doesn't write to the filesystem during normal operation

- **Embedded migrations**: Database migrations are embedded in the binary (no migration files needed)

- **No temp files**: The application doesn't create temporary files under normal operation

- **`/tmp` volume**: The `--tmpfs /tmp` or `emptyDir` volume is **optional but recommended**:

  - **Why optional**: Application doesn't currently use `/tmp` for normal operations

  - **Why recommended**: Defense-in-depth for potential temporary file operations (Go runtime, DNS resolution cache, etc.)

  - **Security benefit**: If `/tmp` is needed, using `noexec` and `nosuid` flags prevents privilege escalation

- **Verification**: Test read-only filesystem works: `docker run --rm --read-only allisson/secrets:v0.10.0 --version`

**Security recommendations**:

1. **Always use `--read-only`** in production to prevent runtime tampering
2. **Add `--tmpfs /tmp`** with `noexec,nosuid` flags for defense-in-depth
3. **Verify with tests**: Include `--read-only` in integration tests to catch regressions

### Resource Limits

**Prevent resource exhaustion attacks:**

```bash
# Docker
docker run --rm \
  --cpus=0.5 \
  --memory=512m \
  --memory-swap=512m \
  --pids-limit=100 \
  allisson/secrets:v0.10.0 server

```

## 3) Network Security

### Port Exposure Strategy

#### Port Configuration

Secrets exposes only one port: 8080 (HTTP)

```dockerfile
EXPOSE 8080

```

**Best practices:**

- ✅ **Use reverse proxy**: Never expose Secrets directly to the internet

- ✅ **TLS termination**: Handle HTTPS at reverse proxy (Nginx, Envoy, Traefik)

- ✅ **Firewall rules**: Restrict access to known IP ranges

- ✅ **Docker networks**: Use custom bridge networks for service isolation

## 4) Secrets Management

### Environment Variable Injection

**Never hardcode secrets in Dockerfiles or images.**

**Docker run with env file:**

```bash
docker run --rm --env-file .env allisson/secrets:v0.10.0 server

```

**Docker Compose with environment variables:**

```yaml
services:
  secrets-api:
    image: allisson/secrets:v0.10.0
    env_file:
      - .env

    # Or use environment variables directly
    environment:
      DB_DRIVER: postgres
      DB_CONNECTION_STRING: ${DB_CONNECTION_STRING}
      MASTER_KEYS: ${MASTER_KEYS}
      ACTIVE_MASTER_KEY_ID: ${ACTIVE_MASTER_KEY_ID}

```

### External Secret Managers

For Docker deployments, you can integrate with external secret managers using environment variables or SDK-based solutions:

**AWS Secrets Manager (using AWS CLI):**

```bash
# Fetch secrets and export as environment variables
export DB_CONNECTION_STRING=$(aws secretsmanager get-secret-value \
  --secret-id prod/secrets/db-connection \
  --query SecretString --output text)

export MASTER_KEYS=$(aws secretsmanager get-secret-value \
  --secret-id prod/secrets/master-keys \
  --query SecretString --output text)

# Run container with exported variables
docker run --rm \
  -e DB_CONNECTION_STRING \
  -e MASTER_KEYS \
  allisson/secrets:v0.10.0 server

```

**Docker Secrets (Swarm mode):**

```bash
# Create secrets in Docker Swarm
echo "postgres://user:pass@db:5432/secrets" | docker secret create db_connection_string -
echo "default:BASE64_KEY" | docker secret create master_keys -

# Use secrets in service
docker service create \
  --name secrets-api \
  --secret db_connection_string \
  --secret master_keys \
  allisson/secrets:v0.10.0 server

```

### Volume Permissions

If mounting volumes, ensure proper ownership:

```bash
# Create directory with correct ownership
mkdir -p /data/secrets
chown 65532:65532 /data/secrets
chmod 750 /data/secrets

# Mount with proper permissions
docker run --rm \
  -v /data/secrets:/data:ro \
  --user 65532:65532 \
  allisson/secrets:v0.10.0 server

```

## 5) Image Scanning

**For comprehensive security scanning documentation**, including SBOM generation, CI/CD integration, continuous monitoring, and vulnerability triage workflows, see:

📖 **[Security Scanning Guide](scanning.md)**

**Quick examples below** (see full guide for advanced usage):

### Trivy Integration

**Scan images for vulnerabilities:**

```bash
# Install Trivy
brew install trivy  # macOS
# or
apt-get install trivy  # Debian/Ubuntu

# Scan image
trivy image allisson/secrets:v0.10.0

# Fail on HIGH/CRITICAL vulnerabilities
trivy image --severity HIGH,CRITICAL --exit-code 1 allisson/secrets:v0.10.0

# Generate SBOM
trivy image --format cyclonedx --output sbom.json allisson/secrets:v0.10.0

```

### Docker Scout

**Scan with Docker Scout:**

```bash
# Enable Docker Scout
docker scout enroll

# Quick scan
docker scout cves allisson/secrets:v0.10.0

# Compare with previous version
docker scout compare --to allisson/secrets:v0.9.0 allisson/secrets:v0.10.0

# View recommendations
docker scout recommendations allisson/secrets:v0.10.0

```

### GitHub Advanced Security

**Configure in GitHub Actions:**

```yaml
name: Container Security Scan

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      
      - name: Build image

        run: docker build -t secrets:test .
      
      - name: Run Trivy vulnerability scanner

        uses: aquasecurity/trivy-action@master
        with:
          image-ref: secrets:test
          format: sarif
          output: trivy-results.sarif
      
      - name: Upload Trivy results to GitHub Security

        uses: github/codeql-action/upload-sarif@v3
        with:
          sarif_file: trivy-results.sarif

```

### CI/CD Integration

**Prevent vulnerable images from deploying:**

```yaml
# .github/workflows/docker-push.yml
jobs:
  build-and-push:
    steps:
      # ... build steps ...
      
      - name: Scan image

        run: |
          trivy image --severity HIGH,CRITICAL --exit-code 1 \
            ${{ secrets.DOCKERHUB_USERNAME }}/secrets:${{ github.sha }}
      
      - name: Push only if scan passes

        if: success()
        run: docker push ${{ secrets.DOCKERHUB_USERNAME }}/secrets:${{ github.sha }}

```

## 6) Health Checks and Observability

### Health Check Endpoints

Secrets exposes two health endpoints for container orchestration:

- **`GET /health`**: Liveness probe (basic health check, < 10ms)

- **`GET /ready`**: Readiness probe (includes database connectivity, < 100ms)

**For complete health check documentation**, including response formats, monitoring integration, and troubleshooting, see:

📖 **[Health Check Endpoints Guide](../observability/health-checks.md)**

**Quick examples below** (see full guide for Docker-specific configurations):

### Docker Compose Health Check

**Workaround for distroless (no shell):**

```yaml
services:
  secrets-api:
    image: allisson/secrets:v0.10.0
    environment:
      - DB_CONNECTION_STRING=postgres://...

    ports:
      - "8080:8080"

    networks:
      - secrets-net

    depends_on:
      postgres:
        condition: service_healthy

  # Sidecar healthcheck container
  healthcheck:
    image: curlimages/curl:latest
    command: >
      sh -c 'while true; do
        curl -f http://secrets-api:8080/health || exit 1;
        sleep 30;
      done'
    depends_on:
      - secrets-api

    networks:
      - secrets-net

    restart: unless-stopped

```

### Prometheus Metrics

**Metrics endpoint** (no authentication required):

```http
GET /metrics

```

**Docker Prometheus scrape configuration:**

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'secrets-api'

    static_configs:
      - targets: ['secrets-api:8080']

    metrics_path: '/metrics'
    scrape_interval: 30s
    scrape_timeout: 10s

```

## 7) Build Security

### Multi-Stage Build Benefits

Secrets uses **multi-stage builds** to separate build and runtime environments:

```dockerfile
# Stage 1: Builder (includes build tools, source code)
FROM golang:1.25.5-trixie AS builder
# ... build steps ...

# Stage 2: Runtime (minimal, only binary)
FROM gcr.io/distroless/static-debian13@sha256:...
COPY --from=builder /app/bin/app /app

```

**Benefits:**

- ✅ **Smaller images**: Final image only contains binary (~12-18MB)

- ✅ **No build tools**: Compiler, source code not in final image

- ✅ **Reduced attack surface**: No unnecessary packages or files

### Build Argument Validation

**Validate build args to prevent injection attacks:**

```dockerfile
ARG VERSION=dev
ARG BUILD_DATE
ARG COMMIT_SHA

# Validate VERSION format (semver or "dev")
RUN if [ "$VERSION" != "dev" ] && ! echo "$VERSION" | grep -Eq '^v[0-9]+\.[0-9]+\.[0-9]+$'; then \
      echo "Invalid VERSION format: $VERSION" && exit 1; \
    fi

```

### Supply Chain Security (SBOM)

**Generate Software Bill of Materials:**

```bash
# Using Syft
syft allisson/secrets:v0.10.0 -o cyclonedx-json > sbom.json

# Using Docker Scout
docker scout sbom allisson/secrets:v0.10.0 --format cyclonedx > sbom.json

# Sign SBOM with Cosign
cosign sign-blob --key cosign.key sbom.json > sbom.json.sig

```

**Note**: The Secrets image includes comprehensive OCI labels that enrich SBOM reports with version metadata, base image provenance, and build information. See [OCI Labels Reference](../deployment/oci-labels.md) for details.

**Verify image signatures:**

```bash
# Sign image with Cosign
cosign sign --key cosign.key allisson/secrets:v0.10.0

# Verify signature
cosign verify --key cosign.pub allisson/secrets:v0.10.0

```

## 8) Deployment Best Practices

### Docker Compose High Availability

**Deploy multiple replicas with load balancing:**

```yaml
version: '3.8'

services:
  secrets-api:
    image: allisson/secrets:v0.10.0
    deploy:
      replicas: 3
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
        reservations:
          cpus: '0.1'
          memory: 128M
    # ... other configuration ...

  # Load balancer (nginx)
  nginx:
    image: nginx:alpine
    ports:
      - "443:443"

    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro

    depends_on:
      - secrets-api

```

### Docker Swarm Deployment

**Scale across multiple nodes:**

```bash
# Initialize swarm
docker swarm init

# Create overlay network
docker network create --driver overlay secrets-net

# Deploy stack with scaling
docker stack deploy -c docker-compose.yml secrets

# Scale service
docker service scale secrets_secrets-api=5

# Update service with zero downtime
docker service update --image allisson/secrets:v0.10.1 secrets_secrets-api

```

## 9) Security Checklist

This comprehensive checklist covers security verification for Docker deployments. Use the **Platform-Specific Checklists** section for your deployment type (Docker Standalone or Docker Compose), then complete the **Common Security Verification** section.

### Platform-Specific Checklists

Choose your deployment type and complete all verification steps before deploying to production.

#### Docker Standalone Checklist

**Pre-Deployment:**

- [ ] **Image verification**:

  - [ ] Base image uses latest distroless digest (`docker inspect --format='{{.Config.Image}}'`)

  - [ ] No HIGH/CRITICAL vulnerabilities (`trivy image allisson/secrets:v0.10.0`)

  - [ ] Image signature verified (if using Docker Content Trust)

- [ ] **Container configuration**:

  - [ ] Non-root user: `docker inspect --format='{{.Config.User}}' allisson/secrets:v0.10.0` shows `65532:65532`

  - [ ] Read-only filesystem tested: `docker run --rm --read-only -v /tmp allisson/secrets:v0.10.0 --version`

  - [ ] Version metadata correct: `docker run --rm allisson/secrets:v0.10.0 --version` shows `v0.10.0`

- [ ] **Volume permissions**:

  - [ ] Named volumes used (not bind mounts) OR bind mount permissions set to UID 65532

  - [ ] Volume permissions tested: `docker run --rm -v secrets-data:/data allisson/secrets:v0.10.0 sh -c 'touch /data/test'`

- [ ] **Network security**:

  - [ ] TLS termination configured (reverse proxy or external load balancer)

  - [ ] Container only exposes port 8080 (HTTP)

  - [ ] Docker network isolation configured (custom bridge network)

- [ ] **Secrets management**:

  - [ ] Environment variables use Docker secrets or external secret manager

  - [ ] No hardcoded credentials in run command or docker-compose.yml

  - [ ] Master key uses KMS provider (not plaintext) for production

**Runtime Monitoring:**

- [ ] **Health checks**:

  - [ ] External health check configured (Docker Compose, monitoring system, or cron job)

  - [ ] Liveness probe tested: `curl -f http://localhost:8080/health`

  - [ ] Readiness probe tested: `curl -f http://localhost:8080/ready`

- [ ] **Resource limits**:

  - [ ] CPU limits set: `--cpus="2.0"`

  - [ ] Memory limits set: `--memory="2g" --memory-swap="2g"`

  - [ ] Restart policy configured: `--restart=unless-stopped`

- [ ] **Security options**:

  - [ ] Capabilities dropped: `--cap-drop=ALL`

  - [ ] No new privileges: `--security-opt=no-new-privileges:true`

  - [ ] AppArmor/SELinux profile applied (if available)

- [ ] **Logging**:

  - [ ] Log driver configured: `--log-driver=json-file --log-opt=max-size=10m --log-opt=max-file=3`

  - [ ] Logs aggregated to central system (optional)

  - [ ] Application audit logs enabled (`AUDIT_LOG_ENABLED=true`)

#### Docker Compose Checklist

**Pre-Deployment:**

- [ ] **Image verification** (same as Docker Standalone):

  - [ ] Base image uses latest distroless digest

  - [ ] No HIGH/CRITICAL vulnerabilities

  - [ ] Image signature verified (if applicable)

- [ ] **Service configuration**:

  - [ ] `user: "65532:65532"` specified in service definition

  - [ ] `read_only: true` with `tmpfs: [/tmp]` configured

  - [ ] `security_opt: [no-new-privileges:true]` set

  - [ ] `cap_drop: [ALL]` configured

- [ ] **Volume permissions**:

  - [ ] Named volumes defined in top-level `volumes:` section

  - [ ] Volume permissions configured using init container or manual setup

  - [ ] Volume mounts tested: `docker compose up -d && docker compose exec secrets-api ls -la /data`

- [ ] **Network security**:

  - [ ] Custom network defined (not default bridge)

  - [ ] Service isolation configured (separate networks for app, db, cache)

  - [ ] External access restricted (only reverse proxy exposed)

- [ ] **Secrets management**:

  - [ ] Secrets use `docker compose secrets` (Swarm mode) or `env_file` with restricted permissions

  - [ ] `.env` file permissions set to `0600`

  - [ ] Master key uses KMS provider (not plaintext)

**Runtime Monitoring:**

- [ ] **Health checks**:

  - [ ] `healthcheck:` stanza configured with external command (wget, curl via sidecar)

  - [ ] Health check interval appropriate: `interval: 30s`, `timeout: 10s`, `retries: 3`

  - [ ] Readiness check tested: `docker compose exec secrets-api curl -f http://localhost:8080/ready`

- [ ] **Resource limits**:

  - [ ] `deploy.resources.limits` configured (memory, cpus)

  - [ ] `deploy.resources.reservations` configured

  - [ ] OOM kill disable: `oom_kill_disable: false` (allow OOM killer)

- [ ] **Restart policies**:

  - [ ] `restart: unless-stopped` configured

  - [ ] Restart tested: `docker compose restart secrets-api`

- [ ] **Logging**:

  - [ ] Logging driver configured: `driver: json-file` with rotation options

  - [ ] Logs accessible: `docker compose logs -f secrets-api`

  - [ ] Application audit logs enabled

### Common Security Verification

**Complete these steps for both Docker Standalone and Docker Compose deployments:**

#### Image Security

- [ ] **Base image verification**:

  - [ ] Image uses `gcr.io/distroless/static-debian13:nonroot` base

  - [ ] Digest pinned (not floating tag): `@sha256:...`

  - [ ] Distroless digest updated within last 30 days

- [ ] **Vulnerability scanning**:

  - [ ] No HIGH vulnerabilities: `trivy image --severity HIGH,CRITICAL allisson/secrets:v0.10.0`

  - [ ] No CRITICAL vulnerabilities

  - [ ] Scan integrated into CI/CD pipeline (fails build on HIGH/CRITICAL)

  - [ ] Scheduled scans configured (weekly minimum)

- [ ] **Image verification**:

  - [ ] OCI labels present and correct:

    ```bash
    docker inspect allisson/secrets:v0.10.0 --format='{{json .Config.Labels}}' | jq
    ```

  - [ ] Version label matches release: `org.opencontainers.image.version=v0.10.0`

  - [ ] Build date within expected range

  - [ ] Commit SHA matches git tag

- [ ] **Build verification**:

  - [ ] Multi-stage build used (builder + runtime stages)

  - [ ] Build args injected correctly (VERSION, BUILD_DATE, COMMIT_SHA)

  - [ ] Application binary built with security flags: `-ldflags="-w -s"`

  - [ ] No build secrets leaked in image layers

#### Runtime Security

- [ ] **User and permissions**:

  - [ ] Container runs as UID 65532 (nonroot user)

  - [ ] No privilege escalation possible

  - [ ] All Linux capabilities dropped

  - [ ] Read-only root filesystem enforced (with writable `/tmp` volume)

- [ ] **Application configuration**:

  - [ ] Master key provider configured (KMS, not plaintext)

  - [ ] Database credentials externalized (environment variables or secrets manager)

  - [ ] TLS enabled for database connections (`sslmode=require` for PostgreSQL)

  - [ ] HTTP server only (HTTPS termination at reverse proxy/load balancer)

  - [ ] Server bind address: `0.0.0.0:8080` (containerized environment)

- [ ] **Network security**:

  - [ ] TLS termination at reverse proxy (nginx, Traefik, Ingress controller)

  - [ ] TLS version >= 1.2

  - [ ] Strong cipher suites configured

  - [ ] HSTS header enabled (reverse proxy)

  - [ ] Internal traffic (container <-> database) encrypted or network-isolated

- [ ] **Secrets management**:

  - [ ] No hardcoded secrets in image

  - [ ] Environment variables injected securely (Docker Secrets, env files with 0600 permissions)

  - [ ] Master key rotation procedure documented and tested

  - [ ] Client secrets rotated regularly (recommendation: every 90 days)

- [ ] **Volume security**:

  - [ ] Volumes have correct permissions (readable/writable by UID 65532)

  - [ ] Sensitive data volumes encrypted at rest (dm-crypt, cloud provider encryption)

  - [ ] Volume backup procedure documented and tested

  - [ ] Volume restore procedure tested

#### Monitoring and Observability

- [ ] **Health checks**:

  - [ ] Liveness probe responding: `curl -f http://container-ip:8080/health` returns 200

  - [ ] Readiness probe responding: `curl -f http://container-ip:8080/ready` returns 200

  - [ ] Health check failures trigger alerts

  - [ ] Health check false positives investigated (e.g., database connection timeouts)

- [ ] **Metrics and logging**:

  - [ ] Prometheus metrics exposed: `/metrics` endpoint accessible

  - [ ] Metrics scraping configured (Prometheus, Datadog, etc.)

  - [ ] Application logs structured (JSON format recommended)

  - [ ] Log level appropriate for environment (INFO for production, DEBUG for staging)

  - [ ] Audit logs enabled: `AUDIT_LOG_ENABLED=true`

  - [ ] Audit logs include all authentication, authorization, and data access events

- [ ] **Alerting**:

  - [ ] Alerts configured for:

    - [ ] Container restarts

    - [ ] Health check failures (liveness, readiness)

    - [ ] High error rates (HTTP 5xx responses)

    - [ ] High latency (P95/P99 > threshold)

    - [ ] Resource exhaustion (CPU, memory near limits)

    - [ ] Authentication failures (brute force attempts)

  - [ ] Alert routing configured (PagerDuty, Slack, email)

  - [ ] Alert runbooks documented

#### Incident Response

- [ ] **Vulnerability response**:

  - [ ] **Assess severity**:

    - [ ] Check CVE score (CVSS >= 7.0 is HIGH)

    - [ ] Determine exploitability (is service exposed to internet?)

    - [ ] Check if vulnerability affects running containers (review Trivy/Scout output)

  - [ ] **Patch procedure**:

    - [ ] Update base image digest in Dockerfile

    - [ ] Rebuild image with same version tag but new digest

    - [ ] Scan new image to verify patch: `trivy image allisson/secrets:v0.10.0`

    - [ ] Test in staging environment

    - [ ] Deploy to production using rolling update

  - [ ] **Hotfix deployment**:

    - [ ] Docker Compose: Update image tag in compose.yml, run `docker compose up -d`

    - [ ] Docker Standalone: Stop container, pull new image, start container

  - [ ] **Verification**:

    - [ ] All containers healthy

    - [ ] Health checks passing

    - [ ] No error spikes in logs

    - [ ] Application functionality tested (smoke tests)

  - [ ] **Documentation**:

    - [ ] Security incident documented (date, CVE, impact, resolution)

    - [ ] Post-mortem created (if HIGH/CRITICAL)

    - [ ] Lessons learned shared with team

- [ ] **Emergency rollback procedure**:

  - [ ] **Docker Compose**:

    ```bash
    # Update image tag to previous version in docker-compose.yml
    # Then restart service
    docker compose up -d secrets-api
    
    # Verify health
    docker compose ps
    docker compose logs secrets-api --tail=100
    curl -f http://localhost:8080/health
    ```

  - [ ] **Docker Standalone**:

    ```bash
    # Stop current container
    docker stop secrets-api
    
    # Pull previous version
    docker pull allisson/secrets:v0.9.0
    
    # Run previous version (use same run command)
    docker run -d --name secrets-api --restart=unless-stopped \
      -p 8080:8080 \
      --env-file .env \
      allisson/secrets:v0.9.0 server
    
    # Verify health
    docker ps
    docker logs secrets-api --tail=100
    curl -f http://localhost:8080/health
    ```

  - [ ] **Post-rollback verification**:

    - [ ] Health checks passing

    - [ ] Application functionality tested

    - [ ] Database connectivity verified

    - [ ] No error spikes in logs

    - [ ] Root cause documented

    - [ ] Forward fix planned

#### Pre-Production Final Checks

- [ ] **Security testing**:

  - [ ] Vulnerability scan passed (no HIGH/CRITICAL)

  - [ ] Penetration testing completed (if required)

  - [ ] Security audit completed (if required)

  - [ ] OWASP Top 10 mitigations verified

- [ ] **Operational readiness**:

  - [ ] Runbooks documented (deployment, rollback, incident response)

  - [ ] On-call rotation configured

  - [ ] Escalation procedures documented

  - [ ] Disaster recovery plan tested

- [ ] **Compliance** (if applicable):

  - [ ] Audit logging meets compliance requirements (SOC 2, HIPAA, GDPR, etc.)

  - [ ] Data encryption at rest and in transit verified

  - [ ] Access controls documented and enforced

  - [ ] Compliance evidence collected (audit logs, scan reports, test results)

### Post-Deployment Verification

**Complete within 24 hours of production deployment:**

- [ ] **Deployment verification**:

  - [ ] All containers running and healthy

  - [ ] Health checks passing (liveness, readiness)

  - [ ] No error spikes in logs (check first 1 hour of logs)

  - [ ] Application metrics baseline established (latency, throughput, error rate)

- [ ] **Functional testing**:

  - [ ] Smoke tests passed (API endpoints responding correctly)

  - [ ] Integration tests passed (database, KMS, external dependencies)

  - [ ] End-to-end critical paths tested (authentication, secret creation, retrieval)

- [ ] **Performance verification**:

  - [ ] Response times within SLA (P50, P95, P99)

  - [ ] Resource usage within expected range (CPU, memory)

  - [ ] Database connection pool healthy

  - [ ] No resource contention (throttling, OOM kills)

- [ ] **Security verification**:

  - [ ] TLS configured and working (test with `curl -v https://api.example.com/health`)

  - [ ] Authentication working (test with valid and invalid credentials)

  - [ ] Authorization working (test with different client policies)

  - [ ] Audit logs being generated (check logs for audit events)

  - [ ] No security alerts triggered

### Ongoing Security Maintenance

**Monthly:**

- [ ] Review and update base image digest (check for security patches)

- [ ] Scan running containers for new vulnerabilities

- [ ] Review audit logs for anomalies

- [ ] Review and rotate client secrets (every 90 days recommended)

- [ ] Test disaster recovery procedures (backup/restore)

**Quarterly:**

- [ ] Review and update security policies

- [ ] Conduct security training for team

- [ ] Review incident response procedures

- [ ] Test rollback procedures in production-like environment

- [ ] Review and update compliance documentation

**Annually:**

- [ ] Conduct security audit (internal or external)

- [ ] Penetration testing (if required)

- [ ] Review and update security architecture

- [ ] Evaluate new security tools and practices

## See Also

- [Security Hardening Guide](hardening.md) - Application-level security

- [Docker Quick Start](../../getting-started/docker.md) - Basic Docker setup

- [Production Deployment Guide](../deployment/production.md) - Production best practices
