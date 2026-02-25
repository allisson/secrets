# 📦 Base Image Migration Guide

> **Document version**: v0.12.0  
> Last updated: 2026-02-24  
> **Audience**: DevOps engineers, SRE teams, platform engineers migrating to distroless base images

## Table of Contents

- [Overview](#overview)
- [Migration Scenarios](#migration-scenarios)
- [Breaking Changes and Solutions](#breaking-changes-and-solutions)
- [Migration Checklist](#migration-checklist)
- [Troubleshooting](#troubleshooting)
- [FAQ](#faq)
- [See Also](#see-also)

## Overview

This guide helps teams migrate from traditional base images (Alpine, scratch, Debian) to Google's **distroless** base images, which are used in Secrets v0.10.0+.

**What changed in v0.10.0:**

| Aspect | Before (< v0.10.0) | After (v0.10.0+) |
|--------|-------------------|------------------|
| **Base image** | `scratch` | `gcr.io/distroless/static-debian13:nonroot` |
| **User** | `root` (UID 0) | `nonroot` (UID 65532) |
| **Shell** | None (scratch) | None (distroless) |
| **Package manager** | None | None |
| **Libc** | None (static binary) | None (static distroless) |
| **CA certificates** | Manual COPY required | Included in distroless |
| **Timezone data** | Manual COPY required | Included in distroless |
| **Image size** | ~10 MB (binary only) | ~2.5 MB (distroless + binary) |
| **Security updates** | Manual rebuild required | Google-managed base layer updates |

**Why migrate to distroless:**

1. **Security patches**: Google maintains the base image with security updates (CVE fixes)
2. **Attack surface reduction**: No shell, package manager, or unnecessary binaries
3. **Supply chain security**: Reproducible builds with SHA256-pinned digests
4. **Compliance**: Smaller attack surface helps meet security compliance requirements (SOC 2, PCI-DSS, HIPAA)
5. **Reduced CVEs**: Fewer vulnerabilities compared to full distributions (Alpine, Debian, Ubuntu)

---

## Migration Scenarios

### Scenario 1: Migrating from Alpine Linux

**Common Alpine-based Dockerfile pattern:**

```dockerfile
# Before: Alpine-based image
FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git ca-certificates tzdata
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 go build -o app ./cmd/app

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /build/app /usr/local/bin/app
USER nobody
ENTRYPOINT ["/usr/local/bin/app"]
CMD ["server"]
```

**After: Distroless migration:**

```dockerfile
# After: Distroless-based image
FROM golang:1.25-alpine AS builder
# apk install no longer needed - distroless includes ca-certificates and tzdata
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags="-w -s" \
    -o app ./cmd/app

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=builder /build/app /app
ENTRYPOINT ["/app"]
CMD ["server"]
```

**Key changes:**

- ✅ Remove `apk add` commands (distroless includes ca-certificates and tzdata)
- ✅ Change final stage FROM to `gcr.io/distroless/static-debian13:nonroot`
- ✅ Remove explicit `USER nobody` (distroless `:nonroot` variant runs as UID 65532 by default)
- ✅ Simplify COPY path (distroless uses `/app` by convention)
- ✅ No need to install dependencies in final stage

**Testing migration:**

```bash
# Build new image
docker build -t myapp:distroless .

# Verify user
docker inspect myapp:distroless --format='{{.Config.User}}'
# Expected: 65532:65532

# Verify no shell
docker run --rm myapp:distroless sh
# Expected: Error - executable file not found

# Verify application works
docker run --rm -p 8080:8080 myapp:distroless server
curl http://localhost:8080/health
# Expected: {"status":"healthy"}
```

---

### Scenario 2: Migrating from Scratch

**Common scratch-based Dockerfile pattern:**

```dockerfile
# Before: Scratch-based image
FROM golang:1.25 AS builder
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 go build -o app ./cmd/app

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /build/app /app
USER 65532:65532
ENTRYPOINT ["/app"]
CMD ["server"]
```

**After: Distroless migration:**

```dockerfile
# After: Distroless-based image
FROM golang:1.25 AS builder
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags="-w -s" \
    -o app ./cmd/app

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=builder /build/app /app
ENTRYPOINT ["/app"]
CMD ["server"]
```

**Key changes:**

- ✅ Remove manual COPY of ca-certificates.crt (included in distroless)
- ✅ Remove manual COPY of timezone data (included in distroless)
- ✅ Remove explicit `USER 65532:65532` (distroless `:nonroot` sets this automatically)
- ✅ Gain security patch support (scratch has no patching mechanism)

**Benefits of distroless over scratch:**

| Feature | Scratch | Distroless |
|---------|---------|------------|
| **CA certificates** | Manual COPY required | ✅ Included |
| **Timezone data** | Manual COPY required | ✅ Included |
| **passwd/group files** | Manual COPY required | ✅ Included (for UID/GID resolution) |
| **Security patches** | ❌ No base layer to patch | ✅ Google-managed updates |
| **CVE scanning** | ❌ No base layer metadata | ✅ Full SBOM and CVE tracking |
| **Reproducibility** | Manual file management | ✅ SHA256-pinned digests |

---

### Scenario 3: Migrating from Debian/Ubuntu

**Common Debian-based Dockerfile pattern:**

```dockerfile
# Before: Debian-based image
FROM golang:1.25 AS builder
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 go build -o app ./cmd/app

FROM debian:bookworm-slim
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/* && \
    useradd -u 65532 -r -s /sbin/nologin appuser
COPY --from=builder /build/app /usr/local/bin/app
USER appuser
ENTRYPOINT ["/usr/local/bin/app"]
CMD ["server"]
```

**After: Distroless migration:**

```dockerfile
# After: Distroless-based image
FROM golang:1.25 AS builder
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 go build \
    -ldflags="-w -s" \
    -o app ./cmd/app

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=builder /build/app /app
ENTRYPOINT ["/app"]
CMD ["server"]
```

**Key changes:**

- ✅ Remove `apt-get` commands (no package manager in distroless)
- ✅ Remove `useradd` command (distroless `:nonroot` includes non-root user)
- ✅ Remove cleanup commands (no apt cache to clean)
- ✅ Reduce image size: ~80 MB (Debian slim) → ~2.5 MB (distroless)
- ✅ Reduce CVE count: ~20-50 CVEs (Debian slim) → 0-5 CVEs (distroless)

**Image size comparison:**

```bash
# Before: Debian slim
docker images debian:bookworm-slim
# REPOSITORY   TAG             SIZE
# debian       bookworm-slim   74.8 MB

# After: Distroless
docker images gcr.io/distroless/static-debian13
# REPOSITORY                          TAG       SIZE
# gcr.io/distroless/static-debian13   nonroot   2.34 MB
```

---

## Breaking Changes and Solutions

### 1. No Shell Available

**Problem**: Distroless images have no `/bin/sh` or `/bin/bash`, breaking shell-based health checks and debugging.

**Before (Alpine/Debian):**

```dockerfile
# Dockerfile
HEALTHCHECK CMD ["sh", "-c", "curl -f http://localhost:8080/health || exit 1"]
```

```yaml
# Docker Compose
services:
  app:
    image: myapp:alpine
    healthcheck:
      test: ["CMD", "sh", "-c", "wget --spider -q http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

**After (Distroless):**

```dockerfile
# Dockerfile: No HEALTHCHECK instruction (use orchestration-level probes)
# See docs/operations/observability/health-checks.md for alternatives
```

```yaml
# Docker Compose: Use external health check sidecar
services:
  app:
    image: myapp:distroless
    ports:
      - "8080:8080"
  
  healthcheck:
    image: curlimages/curl:latest
    depends_on:
      - app
    command: >
      sh -c "while true; do
        curl -f http://app:8080/health || exit 1;
        sleep 30;
      done"
```

**Solutions**:

1. **Docker Compose**: Use external health check sidecar (see example above)
2. **Docker Standalone**: Use external monitoring (cron + curl, Uptime Kuma, Prometheus Blackbox Exporter)
3. **Container Orchestration**: Use native HTTP health checks (e.g., ECS ALB target groups, Cloud Run HTTP probes)

**See also**: [Health Check Endpoints Guide](../observability/health-checks.md) for comprehensive solutions.

---

### 2. No Debugging Tools

**Problem**: Distroless has no `curl`, `wget`, `netstat`, `ps`, or `top` for debugging.

**Solutions**:

#### Option 1: Use Multi-Stage Debug Image

```dockerfile
# Build both production and debug images
FROM gcr.io/distroless/static-debian13:nonroot AS production
COPY --from=builder /build/app /app
ENTRYPOINT ["/app"]

FROM gcr.io/distroless/static-debian13:debug AS debug
COPY --from=builder /build/app /app
ENTRYPOINT ["/app"]
```

```bash
# Build production and debug variants
docker build --target=production -t myapp:latest .
docker build --target=debug -t myapp:debug .

# Use debug variant for troubleshooting
docker run --rm -it myapp:debug sh
# The :debug variant includes a minimal shell (busybox)
```

#### Option 2: Exec into Builder Stage (Local Development)

```bash
# Build and run builder stage for debugging
docker build --target=builder -t myapp:builder .
docker run --rm -it myapp:builder sh

# Inside builder container (has full Go toolchain)
go version
go tool pprof http://production-app:8080/debug/pprof/heap
```

#### Option 3: Debug with Docker Compose Sidecar

```yaml
# docker-compose.debug.yml
services:
  app:
    image: myapp:distroless
    ports:
      - "8080:8080"
    networks:
      - app-network

  debug-tools:
    image: nicolaka/netshoot
    depends_on:
      - app
    networks:
      - app-network
    command: sleep infinity

networks:
  app-network:
```

```bash
# Start application with debug sidecar
docker compose -f docker-compose.debug.yml up -d

# Debug from sidecar container
docker compose exec debug-tools sh
curl http://app:8080/health
netstat -tulpn
tcpdump -i eth0 port 8080
```

#### Option 4: Use External Monitoring Tools

- **Application metrics**: Expose `/metrics` endpoint, scrape with Prometheus
- **Log aggregation**: Ship logs to ELK/Loki/Splunk for analysis
- **Network traffic**: Use Wireshark/tcpdump on host
- **Process inspection**: Use `docker top <container>` or `docker stats <container>`

*### Example: Using docker commands for inspection**

```bash
# View running processes in container
docker top myapp-container

# View resource usage
docker stats myapp-container

# View container logs
docker logs -f myapp-container

# Inspect network connections (from host)
sudo netstat -tulpn | grep :8080
sudo tcpdump -i any port 8080
```

---

### 3. Volume Permissions with Non-Root User

**Problem**: Distroless runs as UID 65532, not root. Host bind mounts may have incompatible permissions.

**Before (Alpine/Debian as root):**

```bash
# Root user can write to any volume
docker run -v /host/data:/data myapp:alpine
```

**After (Distroless as UID 65532):**

```bash
# Volume must be writable by UID 65532
docker run -v /host/data:/data myapp:distroless
# Error: Permission denied
```

**Solutions**:

#### Docker Standalone

```bash
# Option 1: Use named volumes (Docker manages permissions)
docker volume create secrets-data
docker run -v secrets-data:/data myapp:distroless

# Option 2: Fix host bind mount permissions
sudo chown -R 65532:65532 /host/data
docker run -v /host/data:/data myapp:distroless

# Option 3: Run as root (NOT RECOMMENDED)
docker run --user=0:0 -v /host/data:/data myapp:distroless
```

#### Docker Compose

```yaml
version: '3.8'

services:
  app:
    image: myapp:distroless
    # Option 1: Named volumes (recommended)
    volumes:
      - secrets-data:/data
    
    # Option 2: Set user explicitly
    user: "65532:65532"
    
    # Option 3: Use tmpfs for ephemeral data
    tmpfs:
      - /tmp:mode=1777,size=100M

volumes:
  secrets-data:
```

**Fixing permissions for bind mounts:**

```bash
# Create directory with correct ownership
sudo mkdir -p /host/data
sudo chown 65532:65532 /host/data
sudo chmod 755 /host/data

# Verify permissions
ls -la /host/data
# drwxr-xr-x 2 65532 65532 4096 Feb 21 10:00 /host/data

# Now run container with bind mount
docker run -v /host/data:/data myapp:distroless
```

**Testing volume permissions:**

```bash
# Test if container can write to volume
docker run --rm -v /host/data:/data myapp:distroless sh -c "touch /data/test"
# If using distroless (no shell), mount a writable volume and check logs for errors

# Verify ownership inside container
docker run --rm -v /host/data:/data alpine:latest sh -c "ls -la /data"
```

**See also**: [Volume Permissions Troubleshooting Guide](../troubleshooting/volume-permissions.md)

---

### 4. No Package Manager for Runtime Dependencies

**Problem**: Can't install runtime dependencies with `apk add` or `apt-get install`.

**Before (Alpine):**

```dockerfile
FROM alpine:3.21
RUN apk add --no-cache ca-certificates curl jq
COPY app /app
ENTRYPOINT ["/app"]
```

**After (Distroless):**

```dockerfile
# Install dependencies in builder stage, copy to distroless
FROM alpine:3.21 AS builder
RUN apk add --no-cache ca-certificates
# Download static binaries if needed
RUN wget -O /usr/local/bin/jq https://github.com/jqlang/jq/releases/download/jq-1.7.1/jq-linux-amd64 && \
    chmod +x /usr/local/bin/jq

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/local/bin/jq /usr/local/bin/jq
COPY app /app
ENTRYPOINT ["/app"]
```

**Solution**: Install dependencies in builder stage, copy static binaries to final stage.

**Note**: Distroless **already includes** ca-certificates and tzdata, so you don't need to copy these.

---

## Migration Checklist

### Pre-Migration

- [ ] **Review current Dockerfile**:
  - [ ] Identify base image (Alpine, scratch, Debian, Ubuntu)
  - [ ] List runtime dependencies (packages installed with `apk`/`apt-get`)
  - [ ] Check for shell-based health checks (`HEALTHCHECK CMD ["sh", "-c", ...]`)
  - [ ] Identify debugging tools used (`curl`, `wget`, `ps`, `netstat`)
- [ ] **Check application requirements**:
  - [ ] Application is statically compiled (`CGO_ENABLED=0`)
  - [ ] No dynamic linking to system libraries (`ldd /path/to/binary` shows "not a dynamic executable")
  - [ ] No runtime file writes (except `/tmp`)
- [ ] **Review deployment manifests**:
  - [ ] Shell-based health checks in Docker Compose or ECS
  - [ ] Volume mount permissions assumptions
  - [ ] User/UID assumptions in deployment configurations

### Migration Steps

- [ ] **Update Dockerfile**:
  - [ ] Change final stage to `FROM gcr.io/distroless/static-debian13:nonroot`
  - [ ] Remove `apk add` / `apt-get install` from final stage
  - [ ] Remove manual `USER` directive (distroless sets UID 65532 automatically)
  - [ ] Remove manual COPY of ca-certificates (included in distroless)
  - [ ] Remove `HEALTHCHECK` instruction (use orchestration-level probes)
  - [ ] Pin distroless digest: `FROM gcr.io/distroless/static-debian13:nonroot@sha256:...`
- [ ] **Update health checks**:
  - [ ] Replace shell-based health checks with HTTP probes
  - [ ] Update Docker Compose health checks (use sidecar or external monitoring)
  - [ ] Update ECS task definition (use ALB target group health checks)
  - [ ] Configure external monitoring for standalone Docker deployments
- [ ] **Fix volume permissions**:
  - [ ] Use named volumes instead of bind mounts (Docker)
  - [ ] Fix existing bind mount permissions: `chown -R 65532:65532 /path`
  - [ ] Test volume writes: `docker exec <container> touch /data/test`
  - [ ] Configure volume permissions in Docker Compose files
- [ ] **Update debugging procedures**:
  - [ ] Create `:debug` variant image (optional)
  - [ ] Set up Docker Compose debugging sidecar
  - [ ] Set up external monitoring (Prometheus, log aggregation)
  - [ ] Document new debugging workflows for team

### Testing

- [ ] **Build and scan**:
  - [ ] Build new image: `docker build -t myapp:distroless .`
  - [ ] Verify user: `docker inspect myapp:distroless --format='{{.Config.User}}'` → `65532:65532`
  - [ ] Verify no shell: `docker run --rm myapp:distroless sh` → Error
  - [ ] Scan for vulnerabilities: `trivy image myapp:distroless`
  - [ ] Verify image size reduction (compare before/after)
- [ ] **Functional testing**:
  - [ ] Run application: `docker run --rm -p 8080:8080 myapp:distroless server`
  - [ ] Test health endpoints: `curl http://localhost:8080/health`
  - [ ] Test API endpoints: `curl http://localhost:8080/v1/...`
  - [ ] Test with read-only filesystem: `docker run --rm --read-only --tmpfs /tmp myapp:distroless server`
  - [ ] Test volume permissions (if applicable)
- [ ] **Integration testing**:
  - [ ] Deploy to staging environment
  - [ ] Run integration tests against staging
  - [ ] Verify database connectivity
  - [ ] Verify KMS provider connectivity
  - [ ] Monitor logs for errors (24 hour soak test)
  - [ ] Load test to verify performance (compare to previous version)
- [ ] **Rollback testing**:
  - [ ] Test rollback to previous version
  - [ ] Verify data compatibility (no schema changes in v0.10.0)
  - [ ] Document rollback procedure

### Deployment

- [ ] **Staging deployment**:
  - [ ] Deploy to staging environment
  - [ ] Verify all functionality works
  - [ ] Monitor for 24-48 hours
  - [ ] Load test under production-like traffic
- [ ] **Production deployment**:
  - [ ] Use rolling update strategy (zero downtime)
  - [ ] Monitor health checks during rollout
  - [ ] Monitor error rates (should not increase)
  - [ ] Verify volume permissions (check logs for permission errors)
  - [ ] Verify authentication/authorization working
  - [ ] Monitor for 24 hours post-deployment

### Post-Migration

- [ ] **Verify security improvements**:
  - [ ] Scan for CVEs: Compare before/after vulnerability counts
  - [ ] Verify non-root user: `docker exec -it <container> id` → `uid=65532(nonroot)`
  - [ ] Verify read-only filesystem working
  - [ ] Verify no privilege escalation possible
- [ ] **Documentation**:
  - [ ] Update team runbooks with new debugging procedures
  - [ ] Update deployment documentation
  - [ ] Share migration lessons learned
- [ ] **Monitor**:
  - [ ] Set up alerts for new CVEs in base image
  - [ ] Schedule monthly base image digest updates
  - [ ] Monitor application performance (compare to pre-migration baseline)

---

## Troubleshooting

### Issue: "exec /app: no such file or directory"

**Cause**: Binary not copied to distroless image, or wrong ENTRYPOINT path.

**Solution**:

```dockerfile
# Ensure binary is copied to expected path
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=builder /build/app /app
# Verify ENTRYPOINT matches COPY path
ENTRYPOINT ["/app"]
```

### Issue: "standard_init_linux.go: exec user process caused: no such file or directory"

**Cause**: Binary is dynamically linked, but distroless is static-only.

**Solution**:

```dockerfile
# Ensure CGO is disabled for static compilation
RUN CGO_ENABLED=0 go build -o app ./cmd/app

# Verify binary is static
RUN ldd /build/app
# Expected: "not a dynamic executable"
```

### Issue: Health checks failing after migration

**Cause**: Using shell-based health checks (e.g., `sh -c curl ...`).

**Solution**: Use HTTP-based health checks (see [Health Check Guide](../observability/health-checks.md)).

### Issue: Permission denied when writing to volumes

**Cause**: Volume owned by root (UID 0), but container runs as UID 65532.

**Solution**: See [Volume Permissions Guide](../troubleshooting/volume-permissions.md).

### Issue: Can't debug application (no shell)

**Cause**: Distroless has no shell or debugging tools.

**Solution**: Use `:debug` variant image or Docker Compose sidecar (see [No Debugging Tools](#2-no-debugging-tools) section).

---

## FAQ

### Q: Should I use `:debug` or `:nonroot` variant?

**A**: Use `:nonroot` for production, `:debug` only for troubleshooting.

- **`:nonroot`** (recommended): Minimal attack surface, runs as UID 65532, no shell
- **`:debug`**: Includes busybox shell for debugging, larger image, use only for troubleshooting

### Q: How do I update the distroless base image digest?

**A**: Use `docker pull` to get the latest digest, then update your Dockerfile:

```bash
# Pull latest distroless image
docker pull gcr.io/distroless/static-debian13:nonroot

# Get new digest
docker inspect gcr.io/distroless/static-debian13:nonroot --format='{{index .RepoDigests 0}}'
# Output: gcr.io/distroless/static-debian13:nonroot@sha256:NEW_DIGEST

# Update Dockerfile
FROM gcr.io/distroless/static-debian13:nonroot@sha256:NEW_DIGEST
```

### Q: Can I use distroless with CGO-enabled applications?

**A**: No, use `gcr.io/distroless/base-debian13:nonroot` instead (includes glibc).

```dockerfile
# For CGO applications
FROM gcr.io/distroless/base-debian13:nonroot
# Includes: glibc, libssl, openssl, ca-certificates, tzdata
```

### Q: How do I reduce image size even further?

**A**: Use build flags and strip symbols:

```dockerfile
RUN CGO_ENABLED=0 go build \
    -ldflags="-w -s" \
    -trimpath \
    -o app ./cmd/app
# -w: Omit DWARF symbol table
# -s: Omit symbol table and debug info
# -trimpath: Remove file system paths from binary
```

### Q: Can I run distroless as root if needed?

**A**: Yes, but **NOT RECOMMENDED**. Use the base tag without `:nonroot`:

```dockerfile
FROM gcr.io/distroless/static-debian13:latest
USER 0
# Runs as root (UID 0) - NOT RECOMMENDED for production
```

---

## See Also

- [Container Security Guide](../security/container-security.md) - Security best practices
- [Health Check Endpoints Guide](../observability/health-checks.md) - Health check patterns for distroless
- [Volume Permissions Troubleshooting](../troubleshooting/volume-permissions.md) - Fix permission issues
- [Production Deployment Guide](production.md) - Production deployment patterns
- [Distroless GitHub Repository](https://github.com/GoogleContainerTools/distroless) - Official distroless docs
