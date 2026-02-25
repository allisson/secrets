# 🐳 Docker Compose Deployment Guide

> **Document version**: v0.13.0  
> Last updated: 2026-02-25  
> **Audience**: Developers, DevOps engineers deploying with Docker Compose

## Table of Contents

- [Overview](#overview)
- [Quick Start (Development)](#quick-start-development)
- [Production Configuration](#production-configuration)
- [Volume Permissions (v0.10.0+)](#volume-permissions-v0100)
- [Health Checks](#health-checks)
- [Monitoring and Logging](#monitoring-and-logging)
- [Complete Production Stack (All-in-One)](#complete-production-stack-all-in-one)
- [Deployment Workflow](#deployment-workflow)
- [Production Checklist](#production-checklist)
- [Troubleshooting](#troubleshooting)
- [See Also](#see-also)

## Overview

This guide provides production-ready Docker Compose configurations for deploying Secrets with PostgreSQL or MySQL, including security best practices, health checks, and monitoring.

**⚠️ IMPORTANT**: These Docker Compose files are **UNTESTED** in production environments. They are provided as reference examples based on Docker Compose best practices and the Secrets application architecture. **Test thoroughly in a non-production environment** before deploying to production.

**What's included:**

- Complete development stack (Secrets + PostgreSQL/MySQL)
- Production-ready configuration with security hardening
- Health check monitoring with sidecar pattern
- TLS termination with nginx reverse proxy
- Secrets management with environment files
- Volume permission handling for non-root user

---

## Quick Start (Development)

### PostgreSQL Stack

See [`examples/deployment/docker-compose.dev.yml`](../../examples/deployment/docker-compose.dev.yml) for the full code.

```bash
# 1. Download the example file
curl -O https://raw.githubusercontent.com/allisson/secrets/main/docs/examples/deployment/docker-compose.dev.yml
mv docker-compose.dev.yml docker-compose.yml

# 2. Start stack
docker compose up -d

# 3. Verify
docker compose ps
curl http://localhost:8080/health
```

### MySQL Stack

See [`examples/deployment/docker-compose.mysql.yml`](../../examples/deployment/docker-compose.mysql.yml) for the full configuration.

```bash
# 1. Download the example
curl -O https://raw.githubusercontent.com/allisson/secrets/main/docs/examples/deployment/docker-compose.mysql.yml

# 2. Start MySQL stack
docker compose -f docker-compose.mysql.yml up -d
```

---

## Production Configuration

### PostgreSQL Production Stack

See [`examples/deployment/docker-compose.prod.yml`](../../examples/deployment/docker-compose.prod.yml).

**File: `.env.postgres`**

```bash
# PostgreSQL configuration
POSTGRES_USER=secrets
POSTGRES_PASSWORD=<CHANGE_ME_STRONG_PASSWORD>
POSTGRES_DB=secrets

# PostgreSQL performance tuning
POSTGRES_INITDB_ARGS=--encoding=UTF-8 --locale=en_US.UTF-8
```

**File: `.env.secrets`**

```bash
# Database configuration
DB_DRIVER=postgres
DB_CONNECTION_STRING=postgresql://secrets:<CHANGE_ME>@postgres:5432/secrets?sslmode=require

# Master key provider (PRODUCTION: use KMS, not plaintext)
MASTER_KEY_PROVIDER=aws-kms
KMS_KEY_URI=arn:aws:kms:us-east-1:123456789012:key/abc-123...

# Alternative KMS providers:
# MASTER_KEY_PROVIDER=gcp-kms
# KMS_KEY_URI=projects/my-project/locations/us/keyRings/secrets/cryptoKeys/master
#
# MASTER_KEY_PROVIDER=azure-kv
# KMS_KEY_URI=https://my-vault.vault.azure.net/keys/master-key/version

# Server configuration
SERVER_ADDRESS=0.0.0.0:8080
SERVER_READ_TIMEOUT=30s
SERVER_WRITE_TIMEOUT=30s

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Audit logging
AUDIT_LOG_ENABLED=true

# CORS (adjust for your domains)
CORS_ENABLED=true
CORS_ALLOWED_ORIGINS=https://app.example.com,https://admin.example.com
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS=Authorization,Content-Type

# Rate limiting
RATE_LIMIT_ENABLED=true
RATE_LIMIT_MAX_REQUESTS=10
RATE_LIMIT_DURATION=60
```

See [`examples/deployment/nginx.conf`](../../examples/deployment/nginx.conf).

### Deploy Production Stack

```bash
# 1. Create .env files (see above)
vim .env.postgres
vim .env.secrets

# Set correct permissions
chmod 600 .env.postgres .env.secrets

# 2. Generate TLS certificates (self-signed for testing, use Let's Encrypt for production)
mkdir -p certs/nginx certs/postgres

# Generate nginx certs
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout certs/nginx/server.key \
  -out certs/nginx/server.crt \
  -subj "/CN=secrets.example.com"

# Generate postgres certs
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout certs/postgres/server.key \
  -out certs/postgres/server.crt \
  -subj "/CN=postgres"
chmod 600 certs/postgres/server.key

# 3. Start production stack
docker compose -f docker-compose.prod.yml up -d

# 4. Run migrations
docker compose -f docker-compose.prod.yml exec secrets-api /app migrate

# 5. Verify
docker compose -f docker-compose.prod.yml ps
curl -k https://localhost/health
```

---

## Volume Permissions (v0.10.0+)

v0.10.0 runs as non-root user (UID 65532). If using bind mounts, fix permissions:

### Option 1: Use Named Volumes (Recommended)

```yaml
# docker-compose.yml
volumes:
  secrets-data:
    driver: local

services:
  secrets-api:
    volumes:
      - secrets-data:/data
```

Docker manages permissions automatically.

### Option 2: Fix Host Directory Permissions

```bash
# Create directory with correct ownership
mkdir -p /data/secrets
sudo chown -R 65532:65532 /data/secrets

# Use in compose
services:
  secrets-api:
    volumes:
      - /data/secrets:/data
```

### Option 3: Init Container Pattern

```yaml
services:
  secrets-init:
    image: alpine:3.21
    command: chown -R 65532:65532 /data
    volumes:
      - secrets-data:/data

  secrets-api:
    depends_on:
      - secrets-init
    volumes:
      - secrets-data:/data
```

**See also**: [Volume Permission Troubleshooting Guide](../troubleshooting/volume-permissions.md)

---

## Health Checks

### External Health Check (Sidecar Pattern)

Since distroless images have no shell, use an external container for health checking:

```yaml
services:
  secrets-api:
    image: allisson/secrets:v0.13.0
    # No HEALTHCHECK instruction (distroless has no shell)

  healthcheck:
    image: alpine:3.21
    depends_on:
      - secrets-api
    command: >
      sh -c '
      while true; do
        if ! wget --spider -q http://secrets-api:8080/health; then
          echo "FAILED: Health check at $$(date)"
          exit 1
        fi
        sleep 30
      done
      '
    restart: unless-stopped
```

### Monitoring Integration

Use external monitoring tools:

**Uptime Kuma:**

```yaml
services:
  uptime-kuma:
    image: louislam/uptime-kuma:1
    ports:
      - "3001:3001"
    volumes:
      - uptime-kuma-data:/app/data
    # Add secrets-api to Uptime Kuma monitors
```

**Prometheus + Blackbox Exporter:**

```yaml
services:
  blackbox-exporter:
    image: prom/blackbox-exporter:latest
    ports:
      - "9115:9115"
    volumes:
      - ./blackbox.yml:/etc/blackbox_exporter/config.yml

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
```

**See also**: [Health Check Endpoints Guide](../observability/health-checks.md)

---

## Monitoring and Logging

### Prometheus Metrics

```yaml
services:
  prometheus:
    image: prom/prometheus:latest
    container_name: prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
    networks:
      - secrets-backend
```

**File: `prometheus.yml`**

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'secrets-api'
    static_configs:
      - targets: ['secrets-api:8080']
```

### Grafana Dashboards

```yaml
services:
  grafana:
    image: grafana/grafana:latest
    container_name: grafana
    ports:
      - "3000:3000"
    environment:
      GF_SECURITY_ADMIN_PASSWORD: admin
    volumes:
      - grafana-data:/var/lib/grafana
    networks:
      - secrets-backend
```

### Log Aggregation (Loki)

```yaml
services:
  loki:
    image: grafana/loki:latest
    ports:
      - "3100:3100"
    volumes:
      - loki-data:/loki

  promtail:
    image: grafana/promtail:latest
    volumes:
      - /var/lib/docker/containers:/var/lib/docker/containers:ro
      - ./promtail-config.yml:/etc/promtail/config.yml
    command: -config.file=/etc/promtail/config.yml
```

---

## Complete Production Stack (All-in-One)

See [`examples/deployment/docker-compose.full.yml`](../../examples/deployment/docker-compose.full.yml) for the full all-in-one stack.

---

## Deployment Workflow

### Initial Deployment

```bash
# 1. Clone repository or create compose files
mkdir secrets-deployment && cd secrets-deployment

# 2. Create .env files
cat > .env.postgres <<EOF
POSTGRES_USER=secrets
POSTGRES_PASSWORD=$(openssl rand -base64 32)
POSTGRES_DB=secrets
EOF

cat > .env.secrets <<EOF
DB_DRIVER=postgres
DB_CONNECTION_STRING=postgresql://secrets:\$(grep POSTGRES_PASSWORD .env.postgres | cut -d= -f2)@postgres:5432/secrets?sslmode=disable
MASTER_KEY_PROVIDER=plaintext
MASTER_KEY_PLAINTEXT=$(openssl rand -base64 32)
LOG_LEVEL=info
AUDIT_LOG_ENABLED=true
EOF

chmod 600 .env.*

# 3. Start stack
docker compose up -d

# 4. Run migrations
docker compose exec secrets-api /app migrate

# 5. Verify
docker compose ps
curl http://localhost:8080/health
```

### Upgrade

```bash
# 1. Pull new image
docker compose pull secrets-api

# 2. Stop old version
docker compose stop secrets-api

# 3. Run migrations (if needed)
docker compose run --rm secrets-api migrate

# 4. Start new version
docker compose up -d secrets-api

# 5. Verify
docker compose logs secrets-api --tail=50
curl http://localhost:8080/health
```

### Rollback

```bash
# 1. Stop current version
docker compose stop secrets-api

# 2. Revert to previous image
# Edit docker-compose.yml: change image tag to previous version
vim docker-compose.yml

# 3. Start previous version
docker compose up -d secrets-api

# 4. Verify
docker compose logs secrets-api --tail=50
curl http://localhost:8080/health
```

---

## Production Checklist

- [ ] **Security**:
  - [ ] `.env` files have 600 permissions
  - [ ] Strong passwords generated (use `openssl rand -base64 32`)
  - [ ] KMS provider configured (not plaintext)
  - [ ] TLS certificates configured (not self-signed)
  - [ ] Security options enabled (`no-new-privileges`, `cap_drop: ALL`)
  - [ ] Read-only filesystem enabled
- [ ] **High Availability**:
  - [ ] Database backups configured
  - [ ] Restart policy: `unless-stopped`
  - [ ] Health checks configured
  - [ ] Resource limits configured
- [ ] **Networking**:
  - [ ] Backend network is internal (no external access)
  - [ ] Only nginx exposed to internet
  - [ ] TLS termination at nginx
- [ ] **Monitoring**:
  - [ ] Prometheus scraping configured
  - [ ] Grafana dashboards set up
  - [ ] Alerts configured
- [ ] **Volumes**:
  - [ ] Named volumes used (not bind mounts)
  - [ ] Volume backups configured
  - [ ] Volume permissions verified (UID 65532)

---

## Troubleshooting

### Container Fails to Start

```bash
# Check logs
docker compose logs secrets-api --tail=100

# Common issues:
# - Database connection failure: check DB_CONNECTION_STRING
# - Permission denied: check volume permissions (UID 65532)
# - KMS authentication failure: check KMS credentials
```

### Health Check Failing

```bash
# Test health endpoint
docker compose exec secrets-api wget --spider -q http://localhost:8080/health
# Or
curl http://localhost:8080/health

# Check database connectivity
docker compose exec postgres pg_isready -U secrets
```

### Volume Permission Errors

See [Volume Permission Troubleshooting Guide](../troubleshooting/volume-permissions.md).

---

## See Also

- [Docker Quick Start](../../getting-started/docker.md) - Basic Docker usage
- [Production Deployment Guide](docker-hardened.md) - General production best practices
- [Health Check Guide](../observability/health-checks.md) - Health check patterns
- [Volume Permissions Guide](../troubleshooting/volume-permissions.md) - Fix permission issues
- [Container Security Guide](docker-hardened.md) - Security hardening
