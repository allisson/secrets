# 🐳 Docker Compose Deployment Guide

> **Document version**: v0.12.0  
> Last updated: 2026-02-24  
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

```bash
# 1. Create docker-compose.yml
cat > docker-compose.yml <<'EOF'
version: '3.8'

services:
  postgres:
    image: postgres:16-alpine
    container_name: secrets-postgres
    environment:
      POSTGRES_USER: secrets
      POSTGRES_PASSWORD: secrets
      POSTGRES_DB: secrets
    ports:
      - "5432:5432"
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U secrets -d secrets"]
      interval: 5s
      timeout: 5s
      retries: 5
    networks:
      - secrets-net

  secrets-api:
    image: allisson/secrets:v0.12.0
    container_name: secrets-api
    depends_on:
      postgres:
        condition: service_healthy
    environment:
      DB_DRIVER: postgres
      DB_CONNECTION_STRING: postgresql://secrets:secrets@postgres:5432/secrets?sslmode=disable
      MASTER_KEY_PROVIDER: plaintext
      MASTER_KEY_PLAINTEXT: cGxlYXNlQ2hhbmdlVGhpc1RvQVJhbmRvbTMyQnl0ZUtleQo=
      LOG_LEVEL: info
      AUDIT_LOG_ENABLED: "true"
    ports:
      - "8080:8080"
    command: ["server"]
    networks:
      - secrets-net
    restart: unless-stopped

volumes:
  postgres-data:

networks:
  secrets-net:
    driver: bridge
EOF

# 2. Start stack
docker compose up -d

# 3. Verify
docker compose ps
curl http://localhost:8080/health
```

### MySQL Stack

```bash
# Create docker-compose.mysql.yml
cat > docker-compose.mysql.yml <<'EOF'
version: '3.8'

services:
  mysql:
    image: mysql:8.0
    container_name: secrets-mysql
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: secrets
      MYSQL_USER: secrets
      MYSQL_PASSWORD: secrets
    ports:
      - "3306:3306"
    volumes:
      - mysql-data:/var/lib/mysql
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost", "-u", "secrets", "-psecrets"]
      interval: 5s
      timeout: 5s
      retries: 5
    networks:
      - secrets-net

  secrets-api:
    image: allisson/secrets:v0.12.0
    container_name: secrets-api
    depends_on:
      mysql:
        condition: service_healthy
    environment:
      DB_DRIVER: mysql
      DB_CONNECTION_STRING: secrets:secrets@tcp(mysql:3306)/secrets?parseTime=true
      MASTER_KEY_PROVIDER: plaintext
      MASTER_KEY_PLAINTEXT: cGxlYXNlQ2hhbmdlVGhpc1RvQVJhbmRvbTMyQnl0ZUtleQo=
      LOG_LEVEL: info
      AUDIT_LOG_ENABLED: "true"
    ports:
      - "8080:8080"
    command: ["server"]
    networks:
      - secrets-net
    restart: unless-stopped

volumes:
  mysql-data:

networks:
  secrets-net:
    driver: bridge
EOF

# Start MySQL stack
docker compose -f docker-compose.mysql.yml up -d
```

---

## Production Configuration

### PostgreSQL Production Stack

**File: `docker-compose.prod.yml`**

```yaml
version: '3.8'

services:
  # PostgreSQL database
  postgres:
    image: postgres:16-alpine
    container_name: secrets-postgres
    env_file:
      - .env.postgres
    volumes:
      - postgres-data:/var/lib/postgresql/data
      # SSL certificates for TLS connections
      - ./certs/postgres:/var/lib/postgresql/certs:ro
    ports:
      - "127.0.0.1:5432:5432"  # Bind to localhost only
    command: >
      postgres
      -c ssl=on
      -c ssl_cert_file=/var/lib/postgresql/certs/server.crt
      -c ssl_key_file=/var/lib/postgresql/certs/server.key
      -c max_connections=100
      -c shared_buffers=256MB
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $$POSTGRES_USER -d $$POSTGRES_DB"]
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 10s
    networks:
      - secrets-backend
    restart: unless-stopped
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    cap_add:
      - CHOWN
      - DAC_OVERRIDE
      - SETUID
      - SETGID

  # Secrets API application
  secrets-api:
    image: allisson/secrets:v0.12.0
    container_name: secrets-api
    depends_on:
      postgres:
        condition: service_healthy
    env_file:
      - .env.secrets
    user: "65532:65532"  # Run as nonroot user
    command: ["server"]
    expose:
      - "8080"
    healthcheck:
      test: ["CMD-SHELL", "wget --spider -q http://localhost:8080/health || exit 1"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    networks:
      - secrets-backend
      - secrets-frontend
    restart: unless-stopped
    read_only: true
    tmpfs:
      - /tmp:rw,noexec,nosuid,size=10m
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 2G
        reservations:
          cpus: '0.5'
          memory: 512M

  # Health check sidecar (distroless has no shell for HEALTHCHECK)
  healthcheck:
    image: alpine:3.21
    container_name: secrets-healthcheck
    depends_on:
      - secrets-api
    command: >
      sh -c '
      while true; do
        if ! wget --spider -q -t 1 -T 5 http://secrets-api:8080/health; then
          echo "Health check failed at $$(date)"
          exit 1
        fi
        sleep 30
      done
      '
    networks:
      - secrets-backend
    restart: unless-stopped

  # Nginx reverse proxy (TLS termination)
  nginx:
    image: nginx:1.25-alpine
    container_name: secrets-nginx
    depends_on:
      - secrets-api
    ports:
      - "443:443"
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./certs/nginx:/etc/nginx/certs:ro
      - nginx-logs:/var/log/nginx
    networks:
      - secrets-frontend
    restart: unless-stopped
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    cap_add:
      - CHOWN
      - DAC_OVERRIDE
      - SETUID
      - SETGID
      - NET_BIND_SERVICE

volumes:
  postgres-data:
    driver: local
  nginx-logs:
    driver: local

networks:
  secrets-backend:
    driver: bridge
    internal: true  # No external access
  secrets-frontend:
    driver: bridge
```

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

**File: `nginx.conf`**

```nginx
events {
    worker_connections 1024;
}

http {
    # Security headers
    add_header X-Content-Type-Options nosniff always;
    add_header X-Frame-Options DENY always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    # Logging
    access_log /var/log/nginx/access.log;
    error_log /var/log/nginx/error.log;

    # Upstream
    upstream secrets_api {
        server secrets-api:8080;
    }

    # HTTP -> HTTPS redirect
    server {
        listen 80;
        server_name secrets.example.com;
        return 301 https://$server_name$request_uri;
    }

    # HTTPS server
    server {
        listen 443 ssl http2;
        server_name secrets.example.com;

        # TLS configuration
        ssl_certificate /etc/nginx/certs/server.crt;
        ssl_certificate_key /etc/nginx/certs/server.key;
        ssl_protocols TLSv1.2 TLSv1.3;
        ssl_ciphers HIGH:!aNULL:!MD5;
        ssl_prefer_server_ciphers on;

        # Client body size limit
        client_max_body_size 1M;

        # Timeouts
        proxy_connect_timeout 30s;
        proxy_send_timeout 30s;
        proxy_read_timeout 30s;

        location / {
            proxy_pass http://secrets_api;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }

        # Health check endpoint (no auth)
        location /health {
            proxy_pass http://secrets_api/health;
        }
    }
}
```

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
    image: allisson/secrets:v0.12.0
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

**File: `docker-compose.full.yml`**

```yaml
version: '3.8'

services:
  # Database
  postgres:
    image: postgres:16-alpine
    env_file: .env.postgres
    volumes:
      - postgres-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $$POSTGRES_USER"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - backend
    restart: unless-stopped

  # Application
  secrets-api:
    image: allisson/secrets:v0.12.0
    depends_on:
      postgres:
        condition: service_healthy
    env_file: .env.secrets
    user: "65532:65532"
    command: ["server"]
    expose:
      - "8080"
    networks:
      - backend
      - frontend
    restart: unless-stopped
    read_only: true
    tmpfs:
      - /tmp:rw,noexec,nosuid,size=10m
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL

  # Reverse proxy
  nginx:
    image: nginx:1.25-alpine
    depends_on:
      - secrets-api
    ports:
      - "443:443"
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./certs:/etc/nginx/certs:ro
    networks:
      - frontend
    restart: unless-stopped

  # Monitoring
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus-data:/prometheus
    networks:
      - backend

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      GF_SECURITY_ADMIN_PASSWORD: ${GRAFANA_PASSWORD:-admin}
    volumes:
      - grafana-data:/var/lib/grafana
    networks:
      - backend

volumes:
  postgres-data:
  prometheus-data:
  grafana-data:

networks:
  backend:
    internal: true
  frontend:
```

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
- [Production Deployment Guide](production.md) - General production best practices
- [Health Check Guide](../observability/health-checks.md) - Health check patterns
- [Volume Permissions Guide](../troubleshooting/volume-permissions.md) - Fix permission issues
- [Container Security Guide](../security/container-security.md) - Security hardening
