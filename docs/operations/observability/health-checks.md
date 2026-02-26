# 🏥 Health Check Endpoints

> **Document version**: v0.x
> Last updated: 2026-02-25
> **Audience**: Platform engineers, SRE teams, monitoring specialists

This guide covers the health check endpoints exposed by Secrets for container orchestration, monitoring, and operational readiness validation.

## Table of Contents

- [Overview](#overview)

- [Endpoints](#endpoints)

  - [GET /health (Liveness)](#get-health-liveness)

  - [GET /ready (Readiness)](#get-ready-readiness)

- [Response Format](#response-format)

- [Platform Integration](#platform-integration)

  - [Docker Compose](#docker-compose)

  - [Docker Swarm](#docker-swarm)

  - [AWS ECS](#aws-ecs)

  - [Google Cloud Run](#google-cloud-run)

- [Monitoring Integration](#monitoring-integration)

- [Troubleshooting](#troubleshooting)

- [Best Practices](#best-practices)

## Overview

Secrets exposes two HTTP endpoints for health monitoring:

| Endpoint | Purpose | Use Case | Checks |
|----------|---------|----------|--------|
| **`GET /health`** | Liveness probe | Restart unhealthy containers | Application running |
| **`GET /ready`** | Readiness probe | Route traffic to healthy instances | Application + database connectivity |

**Key differences**:

- **`/health`**: Fast, basic check (< 10ms). Returns 200 if the application process is running.

- **`/ready`**: Comprehensive check (< 100ms). Returns 200 only if application can handle requests (database accessible).

**When to use each**:

- **Liveness (`/health`)**: Detect deadlocks, crashes, or unrecoverable failures → restart container

- **Readiness (`/ready`)**: Detect temporary issues (DB connection loss, startup) → stop routing traffic until recovered

## Endpoints

### GET /health (Liveness)

**Purpose**: Verify the application process is alive and responsive.

**Response codes**:

- `200 OK`: Application is running

- `5xx`: Application is unresponsive or crashed (orchestrator should restart)

**Response body**:

```json
{
  "status": "healthy"
}

```

**Example request**:

```bash
curl -i http://localhost:8080/health

```

**Example response**:

```http
HTTP/1.1 200 OK
Content-Type: application/json
Date: Fri, 21 Feb 2026 10:30:00 GMT
Content-Length: 21

{"status":"healthy"}

```

**Typical response time**: < 10ms

**Use in orchestration**:

- Docker Compose `healthcheck` (via sidecar)

- AWS ECS `healthCheck` (container health)

- Google Cloud Run `liveness_check`

- Docker Swarm `HEALTHCHECK`

**When this fails**:

- Application crashed or deadlocked

- HTTP server not accepting connections

- Process killed or out of memory

**Recommended action**: Restart the container

### GET /ready (Readiness)

**Purpose**: Verify the application can handle requests (includes database connectivity check).

**Response codes**:

- `200 OK`: Application ready to handle requests

- `503 Service Unavailable`: Application not ready (database unreachable, startup in progress)

**Response body (success)**:

```json
{
  "status": "ready",
  "database": "ok"
}

```

**Response body (failure)**:

```json
{
  "status": "not_ready",
  "database": "unavailable",
  "error": "failed to ping database: connection refused"
}

```

**Example request**:

```bash
curl -i http://localhost:8080/ready

```

**Example response (ready)**:

```http
HTTP/1.1 200 OK
Content-Type: application/json
Date: Fri, 21 Feb 2026 10:30:00 GMT
Content-Length: 42

{"status":"ready","database":"ok"}

```

**Example response (not ready)**:

```http
HTTP/1.1 503 Service Unavailable
Content-Type: application/json
Date: Fri, 21 Feb 2026 10:30:00 GMT
Content-Length: 98

{"status":"not_ready","database":"unavailable","error":"failed to ping database: connection refused"}

```

**Typical response time**: < 100ms (includes database ping)

**Use in orchestration**:

- Docker Compose healthcheck readiness

- AWS ECS target group health checks

- Load balancer health checks (ALB, NLB, GCP LB)

- Google Cloud Run readiness checks

- AWS ECS `healthCheck` (load balancer target health)

- Google Cloud Run `startup_check`

- Load balancer health checks

**When this fails**:

- Database connection lost

- Database credentials invalid

- Network partition between app and database

- Application still starting up

**Recommended action**: Stop routing traffic, wait for recovery (do NOT restart)

## Response Format

Both endpoints return JSON with consistent structure:

**Success response schema**:

```json
{
  "status": "healthy" | "ready",
  "database": "ok"  // only in /ready
}

```

**Failure response schema**:

```json
{
  "status": "not_ready",
  "database": "unavailable",
  "error": "error message"
}

```

**HTTP status codes**:

| Endpoint | Success | Failure | Description |
|----------|---------|---------|-------------|
| `/health` | 200 OK | 5xx | Application liveness |
| `/ready` | 200 OK | 503 Service Unavailable | Application + dependencies |

## Platform Integration

### Docker Compose

**Problem**: Distroless images have no shell, so Docker's built-in `HEALTHCHECK` directive doesn't work.

**Solution 1: Healthcheck sidecar container** (recommended for development):

```yaml
version: '3.8'

services:
  secrets-api:
    image: allisson/secrets:<VERSION>
    container_name: secrets-api
    ports:
      - "8080:8080"

    environment:
      DB_DRIVER: postgres
      DB_CONNECTION_STRING: postgres://user:pass@db:5432/secrets?sslmode=disable
      MASTER_KEYS: default:bEu+O/9NOFAsWf1dhVB9aprmumKhhBcE6o7UPVmI43Y=
      ACTIVE_MASTER_KEY_ID: default
    depends_on:
      db:
        condition: service_healthy
    networks:
      - secrets-net

  # Healthcheck sidecar (monitors secrets-api health)
  healthcheck:
    image: curlimages/curl:latest
    container_name: secrets-healthcheck
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

  db:
    image: postgres:16-alpine
    container_name: secrets-db
    environment:
      POSTGRES_DB: secrets
      POSTGRES_USER: user
      POSTGRES_PASSWORD: pass
    volumes:
      - postgres-data:/var/lib/postgresql/data

    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U user -d secrets"]
      interval: 10s
      timeout: 3s
      retries: 3
    networks:
      - secrets-net

volumes:
  postgres-data:

networks:
  secrets-net:

```

**Solution 2: External monitoring** (recommended for production):

Use external tools like:

- **Prometheus Blackbox Exporter** (HTTP probes)

- **Uptime Kuma** (uptime monitoring dashboard)

- **Datadog / New Relic** (synthetic monitoring)

**Example: Prometheus Blackbox Exporter**:

```yaml
# docker-compose.yml
services:
  secrets-api:
    image: allisson/secrets:<VERSION>
    # ... config ...

  blackbox-exporter:
    image: prom/blackbox-exporter:latest
    ports:
      - "9115:9115"

    volumes:
      - ./blackbox.yml:/etc/blackbox_exporter/config.yml:ro

    networks:
      - secrets-net

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"

    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml:ro

    networks:
      - secrets-net

```

```yaml
# blackbox.yml
modules:
  http_2xx:
    prober: http
    timeout: 5s
    http:
      valid_http_versions: ["HTTP/1.1", "HTTP/2.0"]
      valid_status_codes: [200]
      method: GET
      fail_if_not_ssl: false

```

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'blackbox-health'

    metrics_path: /probe
    params:
      module: [http_2xx]
    static_configs:
      - targets:

        - http://secrets-api:8080/health

        - http://secrets-api:8080/ready

    relabel_configs:
      - source_labels: [__address__]

        target_label: __param_target
      - source_labels: [__param_target]

        target_label: instance
      - target_label: __address__

        replacement: blackbox-exporter:9115

```

### Docker Swarm

```yaml
version: '3.8'

services:
  secrets-api:
    image: allisson/secrets:<VERSION>
    deploy:
      replicas: 3
      update_config:
        parallelism: 1
        delay: 10s
        order: start-first
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
      # Swarm health check (uses external curl container)
      # Note: No native HEALTHCHECK support for distroless
      labels:
        - "traefik.enable=true"

        - "traefik.http.services.secrets.loadbalancer.healthcheck.path=/ready"

        - "traefik.http.services.secrets.loadbalancer.healthcheck.interval=10s"

    environment:
      DB_DRIVER: postgres
      DB_CONNECTION_STRING: postgres://user:pass@db:5432/secrets
    networks:
      - secrets-net

networks:
  secrets-net:
    driver: overlay

```

### AWS ECS

**Fargate Task Definition** (JSON):

```json
{
  "family": "secrets-api",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "256",
  "memory": "512",
  "containerDefinitions": [
    {
      "name": "secrets",
      "image": "allisson/secrets:<VERSION>",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "healthCheck": {
        "command": [
          "CMD-SHELL",
          "curl -f http://localhost:8080/health || exit 1"
        ],
        "interval": 30,
        "timeout": 5,
        "retries": 3,
        "startPeriod": 60
      },
      "environment": [
        {
          "name": "DB_DRIVER",
          "value": "postgres"
        }
      ],
      "secrets": [
        {
          "name": "DB_CONNECTION_STRING",
          "valueFrom": "arn:aws:secretsmanager:region:account:secret:secrets-db-conn"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/secrets-api",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}

```

**Note**: ECS health check uses `curl`, which requires a sidecar or external monitoring. For production, use Application Load Balancer target health checks instead:

**ALB Target Group Health Check**:

```bash
aws elbv2 create-target-group \
  --name secrets-api-tg \
  --protocol HTTP \
  --port 8080 \
  --vpc-id vpc-xxxxx \
  --health-check-enabled \
  --health-check-protocol HTTP \
  --health-check-path /ready \
  --health-check-interval-seconds 30 \
  --health-check-timeout-seconds 5 \
  --healthy-threshold-count 2 \
  --unhealthy-threshold-count 3

```

### Google Cloud Run

**Cloud Run service deployment**:

```yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: secrets-api
  namespace: default
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/minScale: "1"
        autoscaling.knative.dev/maxScale: "10"
    spec:
      containers:
        - image: gcr.io/my-project/secrets:v0.10.0

          ports:
            - containerPort: 8080

          env:
            - name: DB_DRIVER

              value: postgres
            - name: DB_CONNECTION_STRING

              valueFrom:
                secretKeyRef:
                  name: secrets-db
                  key: connection-string
          
          # Cloud Run health checks
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 30
            timeoutSeconds: 3
            failureThreshold: 3
          
          startupProbe:
            httpGet:
              path: /ready
              port: 8080
            initialDelaySeconds: 0
            periodSeconds: 5
            timeoutSeconds: 3
            failureThreshold: 30
          
          resources:
            limits:
              memory: 512Mi
              cpu: 1000m

```

**Deploy via gcloud**:

```bash
gcloud run deploy secrets-api \
  --image gcr.io/my-project/secrets:v0.10.0 \
  --platform managed \
  --region us-central1 \
  --port 8080 \
  --min-instances 1 \
  --max-instances 10 \
  --timeout 60s \
  --allow-unauthenticated

```

**Cloud Run automatically uses `/` for health checks by default**. To verify health endpoints:

```bash
SERVICE_URL=$(gcloud run services describe secrets-api --format='value(status.url)')
curl $SERVICE_URL/health
curl $SERVICE_URL/ready

```

## Monitoring Integration

### Prometheus Blackbox Exporter

Monitor health endpoints and alert on failures:

**Prometheus configuration**:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'secrets-health'

    metrics_path: /probe
    params:
      module: [http_2xx]
    static_configs:
      - targets:

        - http://secrets-api:8080/health

        - http://secrets-api:8080/ready

    relabel_configs:
      - source_labels: [__address__]

        target_label: __param_target
      - source_labels: [__param_target]

        target_label: instance
      - target_label: __address__

        replacement: blackbox-exporter:9115

```

**Alert rules**:

```yaml
# alerts.yml
groups:
  - name: secrets-health

    interval: 30s
    rules:
      - alert: SecretsAPIDown

        expr: probe_success{job="secrets-health",instance=~".*health"} == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Secrets API is down"
          description: "Liveness probe failed for {{ $labels.instance }}"
      
      - alert: SecretsAPINotReady

        expr: probe_success{job="secrets-health",instance=~".*ready"} == 0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Secrets API not ready"
          description: "Readiness probe failed for {{ $labels.instance }} - database may be unreachable"

```

### Datadog Synthetic Monitoring

```yaml
# datadog-synthetics.yml
api_version: v1
kind: synthetics
name: secrets-health-check
type: api
config:
  request:
    method: GET
    url: https://secrets.example.com/health
  assertions:
    - type: statusCode

      operator: is
      target: 200
    - type: responseTime

      operator: lessThan
      target: 100
  locations:
    - aws:us-east-1

    - aws:eu-west-1

  options:
    tick_every: 60
    min_failure_duration: 120
    min_location_failed: 1
  message: "Secrets API health check failed"
  tags:
    - "service:secrets"

    - "env:production"

```

### Uptime Kuma

Self-hosted monitoring dashboard:

```bash
# docker-compose.yml
services:
  uptime-kuma:
    image: louislam/uptime-kuma:latest
    ports:
      - "3001:3001"

    volumes:
      - uptime-kuma-data:/app/data

    restart: unless-stopped

```

**Add monitor in UI**:

1. Navigate to <http://localhost:3001>
2. Add monitor: HTTP(s)
3. URL: `http://secrets-api:8080/health`
4. Heartbeat interval: 60s
5. Retries: 3
6. Alert on failure

## Troubleshooting

### Health endpoint returns 404

**Symptom**: `curl http://localhost:8080/health` returns 404 Not Found

**Causes**:

1. Wrong URL path (e.g., `/healthz` instead of `/health`)
2. Application not running
3. Port mismatch (application on different port)

**Solution**:

```bash
# Verify correct paths
curl -i http://localhost:8080/health
curl -i http://localhost:8080/ready

# Check application logs
docker logs secrets-api

# Verify port binding
docker ps | grep secrets
netstat -tuln | grep 8080

```

### Readiness probe always fails (503)

**Symptom**: `/ready` returns 503, `/health` returns 200

**Cause**: Database connection failure

**Solution**:

```bash
# Check database connectivity from app container
docker exec secrets-api nc -zv db 5432

# Verify DB_CONNECTION_STRING
docker exec secrets-api env | grep DB_CONNECTION_STRING

# Check database logs
docker logs secrets-db

# Test database connection manually
docker exec secrets-db psql -U user -d secrets -c "SELECT 1"

```

**Common database issues**:

- Wrong credentials in `DB_CONNECTION_STRING`

- Database not ready yet (increase `initialDelaySeconds` in readiness probe)

- Network issue between app and database

- Database max connections exceeded

### Container restarts due to health check failures

**Symptom**: Containers restart with "health check failed" messages but application logs show no errors

**Causes**:

1. Health check timeout too short (< 3s)
2. Health check interval too aggressive
3. Initial delay too short (startup not complete)
4. Slow health endpoint (> 1s response time)

**Solution**:

Adjust health check configuration in docker-compose.yml or container orchestration:

```yaml
# Docker Compose example
healthcheck:
  test: ["CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"]
  interval: 30s          # Check every 30s
  timeout: 5s            # Increase from 3s
  retries: 3             # Allow 3 failures
  start_period: 30s      # Give 30s for startup

```

**Debug health check failures**:

```bash
# Check health endpoint response time
time curl http://localhost:8080/health

# Check container logs during probe failure
docker logs secrets-api --tail 100

# Test health endpoint manually
curl -v http://localhost:8080/health

```

### Health checks slow (> 1s)

**Symptom**: Health endpoints take > 1s to respond

**Causes**:

1. Database connectivity issues (affects `/ready` only)
2. High application load
3. Resource constraints (CPU throttling, memory pressure)

**Solution**:

```bash
# Check response times
time curl http://localhost:8080/health   # Should be < 10ms
time curl http://localhost:8080/ready    # Should be < 100ms

# Check application metrics
curl http://localhost:8080/metrics | grep http_request_duration

# Check Docker resource usage
docker stats secrets-api

# Increase container resource limits (Docker Compose)
# Edit docker-compose.yml:
# services:
#   secrets:
#     deploy:
#       resources:
#         limits:
#           cpus: '0.5'
#           memory: 512M

```

## Best Practices

### 1. Use Both Liveness and Readiness Checks

**Recommended**: Configure both health check types in your container orchestration

**Docker Compose example**:

```yaml
services:
  secrets:
    image: allisson/secrets:<VERSION>
    healthcheck:
      test: ["CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 30s

```

**Why**: Liveness detects crashes, readiness detects temporary issues.

### 2. Set Appropriate Timeouts

**Recommended values**:

| Check Type | Start Period | Interval | Timeout | Retries |
|------------|--------------|----------|---------|---------|
| Liveness | 30s | 30s | 5s | 3 |
| Readiness | 10s | 10s | 3s | 2 |

**Rationale**:

- **Liveness**: Conservative (avoid unnecessary restarts)

- **Readiness**: Responsive (quickly detect unhealthy instances)

- **Start Period**: Patient (allow time for migrations, warm-up)

### 3. Monitor Health Check Success Rate

**Prometheus query**:

```promql
# Health check success rate (last 5 minutes)
sum(rate(probe_success{job="secrets-health"}[5m])) by (instance)

# Alert on < 95% success rate
(
  sum(rate(probe_success{job="secrets-health"}[5m])) by (instance)
  /
  sum(rate(probe_duration_seconds_count{job="secrets-health"}[5m])) by (instance)
) < 0.95

```

### 4. Handle Slow Startups

**Problem**: Database migrations can take 30-60s, causing health checks to fail during startup.

**Solution**: Use appropriate start period in health check configuration:

**Docker Compose**:

```yaml
healthcheck:
  test: ["CMD-SHELL", "curl -f http://localhost:8080/ready || exit 1"]
  start_period: 60s    # Allow up to 60s for startup
  interval: 10s
  timeout: 3s
  retries: 3

```

**Effect**: Health checks are not enforced during the start period.

### 5. Separate Monitoring from Orchestration

**Do**:

- Use `/health` and `/ready` for container health checks

- Use Prometheus Blackbox Exporter for monitoring dashboards

- Configure separate alerting thresholds

**Why**: Orchestration needs fast decisions, monitoring needs historical data.

### 6. Test Health Checks in CI/CD

**Example GitHub Actions workflow**:

```yaml

- name: Test health endpoints

  run: |
    docker-compose up -d
    sleep 10
    
    # Test liveness
    curl -f http://localhost:8080/health || exit 1
    
    # Test readiness
    curl -f http://localhost:8080/ready || exit 1
    
    # Verify response format
    curl -s http://localhost:8080/health | jq -e '.status == "healthy"'
    curl -s http://localhost:8080/ready | jq -e '.status == "ready"'

```

### 7. Document Health Check Behavior

**Include in runbooks**:

- Expected response times (< 10ms for `/health`, < 100ms for `/ready`)

- Common failure scenarios and resolutions

- Escalation path when health checks fail

## See Also

- [Monitoring Guide](monitoring.md) - Prometheus metrics and Grafana dashboards

- [Incident Response](incident-response.md) - Troubleshooting production issues

- [Production Deployment](../deployment/docker-hardened.md) - Production deployment checklist

- [Container Security](../deployment/docker-hardened.md) - Security hardening for containers

- [Docker Compose Guide](../deployment/docker-compose.md) - Docker Compose deployment examples
