# 📊 Monitoring

This guide shows you how to set up monitoring for the Secrets application using Prometheus and Grafana.

**For a complete reference of all metrics**, see the **[Metrics Reference](../../observability/metrics-reference.md)**.

**Related guides**:

- **[Metrics Reference](../../observability/metrics-reference.md)** - Complete catalog of all metrics and Prometheus queries
- **[Health Check Endpoints](health-checks.md)** - Liveness and readiness probes for container orchestration
- **[Incident Response](incident-response.md)** - Troubleshooting production issues

## Table of Contents

- [Overview](#overview)
- [Configuration](#configuration)
- [Quickstart (Prometheus + Grafana)](#quickstart-prometheus--grafana)
- [Prometheus Configuration](#prometheus-configuration)
- [Grafana Dashboard](#grafana-dashboard)
- [Alerting](#alerting)
- [Disabling Metrics](#disabling-metrics)
- [Troubleshooting](#troubleshooting)
- [See Also](#see-also)

## Overview

The Secrets application exposes Prometheus-compatible metrics at `http://localhost:8081/metrics`. This guide walks you through:

1. Configuring metrics collection
2. Setting up Prometheus to scrape metrics
3. Creating Grafana dashboards
4. Configuring alerts

**For detailed information about available metrics**, see the **[Metrics Reference](../../observability/metrics-reference.md)**.

The application uses OpenTelemetry for metrics instrumentation with a Prometheus-compatible export endpoint. Metrics can be enabled/disabled via configuration and cover two main areas:

1. **HTTP Request Metrics** - Request counts, durations, and status codes for all API endpoints
2. **Business Operation Metrics** - Domain-specific operation counters and durations (auth, secrets, transit, tokenization)

**Health monitoring**: For liveness and readiness probes, see [Health Check Endpoints](health-checks.md).

## Configuration

### Environment Variables

```bash
# Enable or disable metrics collection
METRICS_ENABLED=true  # default: true

# Namespace prefix for all metrics
METRICS_NAMESPACE=secrets  # default: secrets
```

Update your `.env` file:

```bash
# Metrics configuration
METRICS_ENABLED=true
METRICS_NAMESPACE=secrets
```

## Quickstart (Prometheus + Grafana)

Use this minimal local stack to visualize Secrets metrics quickly:

1. Start Secrets with metrics enabled
2. Start Prometheus with a scrape config for `http://host.docker.internal:8081/metrics`
3. Open Grafana and create panels from Prometheus queries

Note: On Linux, replace `host.docker.internal` with the host IP reachable from your Docker network.

Minimal `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: "secrets-api"
    static_configs:
      - targets: ["host.docker.internal:8081"]
    metrics_path: "/metrics"
```

Quick run commands:

```bash
# Start Prometheus
docker run --rm -d --name prom \
  -p 9090:9090 \
  -v "$(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml:ro" \
  prom/prometheus

# Start Grafana
docker run --rm -d --name grafana \
  -p 3000:3000 \
  grafana/grafana
```

Suggested first panel query (requests/sec by route):

```promql
sum(rate(secrets_http_requests_total[5m])) by (method, path)
```

For more Prometheus queries, see the **[Prometheus Query Library](../../observability/metrics-reference.md#prometheus-query-library)** in the Metrics Reference.

## Prometheus Configuration

### Scrape Configuration

Add the Secrets application to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'secrets-api'
    static_configs:
      - targets: ['localhost:8081']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

**Configuration notes:**

- **Port 8081** - Metrics are exposed on a separate port from the main API (8080)
- **No authentication** - The `/metrics` endpoint is public (standard Prometheus practice)
- **Scrape interval** - 15-30 seconds is recommended for most deployments

For a complete list of available metrics, see the **[Metrics Reference](../../observability/metrics-reference.md#metric-catalog)**.

### Example Queries

This section provides commonly used queries for monitoring. For a comprehensive query library, see the **[Prometheus Query Library](../../observability/metrics-reference.md#prometheus-query-library)** in the Metrics Reference.

**Total requests per second (rate over 5 minutes):**

```promql
rate(secrets_http_requests_total[5m])
```

**95th percentile request latency:**

```promql
histogram_quantile(0.95, rate(secrets_http_request_duration_seconds_bucket[5m]))
```

**Error rate by domain:**

```promql
rate(secrets_operations_total{status="error"}[5m]) / rate(secrets_operations_total[5m])
```

**Slowest operations:**

```promql
topk(5, rate(secrets_operation_duration_seconds_sum[5m]) / rate(secrets_operation_duration_seconds_count[5m]))
```

For more query examples including rate limiting, tokenization-specific queries, and SLO queries, see the **[Metrics Reference](../../observability/metrics-reference.md#prometheus-query-library)**.

### Rate Limiting and Security Queries

**429 rate by route (5m):**

```promql
sum(rate(secrets_http_requests_total{status_code="429"}[5m])) by (path)
```

**429 ratio by route (5m):**

```promql
sum(rate(secrets_http_requests_total{status_code="429"}[5m])) by (path)
/
sum(rate(secrets_http_requests_total[5m])) by (path)
```

**Denied authorization rate (`403`) by route (5m):**

```promql
sum(rate(secrets_http_requests_total{status_code="403"}[5m])) by (path)
```

**Token endpoint `429` ratio (5m):**

```promql
sum(rate(secrets_http_requests_total{path="/v1/token",status_code="429"}[5m]))
/
sum(rate(secrets_http_requests_total{path="/v1/token"}[5m]))
```

**Token endpoint request rate by status (5m):**

```promql
sum(rate(secrets_http_requests_total{path="/v1/token"}[5m])) by (status_code)
```

**Additional rate limiting queries** including 429 analysis, token endpoint health, and throttle pressure metrics are available in the **[Metrics Reference](../../observability/metrics-reference.md#rate-limiting-queries)**.

## Grafana Dashboard

### Dashboard Artifacts

Starter Grafana dashboard JSON artifacts are available for quick setup:

**Available dashboards:**

- **`secrets-overview.json`** - Baseline request/error/latency view
- **`secrets-rate-limiting.json`** - 429 behavior and throttle pressure view

**Location:** `docs/operations/dashboards/`

**Import instructions:**

1. Open Grafana (default: `http://localhost:3000`)
2. Go to **Dashboards** → **Import**
3. Click **Upload JSON file**
4. Select a dashboard file from `docs/operations/dashboards/`
5. Select your Prometheus datasource
6. Click **Import**

**Notes:**

- Treat these dashboards as starter templates
- Adjust panel thresholds and time windows for your traffic profile

For detailed panel recommendations and configuration tips, see the **[Grafana Dashboards](../../observability/metrics-reference.md#grafana-dashboards)** section in the Metrics Reference.

### Recommended Panels

When creating custom dashboards, consider including:

| Panel Type | Metric | Purpose |
|------------|--------|---------|
| Time Series | `rate(secrets_http_requests_total[5m])` | Request rate by route |
| Time Series | `histogram_quantile(0.95, ...)` | p95 latency by route |
| Stat | 5xx error rate | Current server error rate |
| Gauge | API availability | Availability percentage with SLO threshold |
| Table | Top operations by volume | Identify hottest paths |
| Heatmap | Request duration buckets | Latency distribution visualization |

**Example panel query (request rate by route):**

```promql
sum(rate(secrets_http_requests_total[5m])) by (method, path)
```

For more panel examples and configuration guidance, see the **[Metrics Reference](../../observability/metrics-reference.md#grafana-dashboards)**.

## Alerting

### Recommended Alerts

This section provides production-ready alert rules for Prometheus Alertmanager.

**Alert configuration tips:**

- Start with warning thresholds and tune based on your environment
- Use appropriate `for` durations to avoid alert flapping
- Include actionable descriptions in annotations
- Link to runbooks or documentation in annotations

### Alert Ownership and Escalation

| Alert class | Default severity | Primary owner | Escalate if unresolved |
| --- | --- | --- | --- |
| API availability / 5xx surge | `critical` | Platform/on-call | 10 minutes |
| Token issuance failures | `critical` | Platform + IAM owner | 10 minutes |
| Sustained `429` ratio | `warning` | Service owner + platform | 30 minutes |
| Elevated `403` denied rate | `warning` | Security + service owner | 30 minutes |
| Metrics scrape failures | `warning` | Observability owner | 30 minutes |

Suggested escalation policy:

1. Page primary owner immediately for `critical`
2. Notify secondary owner if not acknowledged in 5 minutes
3. Escalate to incident commander when SLA/SLO at risk

#### High Error Rate

```yaml
- alert: HighErrorRate
  expr: rate(secrets_operations_total{status="error"}[5m]) > 0.1
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High error rate detected"
    description: "Error rate is {{ $value }} errors/sec"
```

#### High Latency

```yaml
- alert: HighLatency
  expr: histogram_quantile(0.95, rate(secrets_http_request_duration_seconds_bucket[5m])) > 1
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High request latency"
    description: "95th percentile latency is {{ $value }}s"
```

#### Excessive 429 Ratio

```yaml
- alert: ExcessiveRateLimit429Ratio
  expr: |
    (
      sum(rate(secrets_http_requests_total{status_code="429"}[10m]))
      /
      sum(rate(secrets_http_requests_total[10m]))
    ) > 0.05
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "High 429 ratio detected"
    description: "More than 5% of requests are being throttled"
```

#### 429 Burst On Critical Routes

```yaml
- alert: RateLimitBurstCriticalRoute
  expr: sum(rate(secrets_http_requests_total{status_code="429",path=~"/v1/secrets/.*|/v1/transit/.*"}[5m])) > 2
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Sustained 429 on secrets/transit routes"
    description: "Critical crypto routes are being throttled above threshold"
```

#### Token Endpoint 429 Ratio (Warning)

```yaml
- alert: TokenEndpoint429RatioWarning
  expr: |
    (
      sum(rate(secrets_http_requests_total{path="/v1/token",status_code="429"}[10m]))
      /
      sum(rate(secrets_http_requests_total{path="/v1/token"}[10m]))
    ) > 0.05
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "Token endpoint throttling elevated"
    description: "More than 5% of /v1/token requests are returning 429"
```

#### Token Endpoint 429 Ratio (Critical)

```yaml
- alert: TokenEndpoint429RatioCritical
  expr: |
    (
      sum(rate(secrets_http_requests_total{path="/v1/token",status_code="429"}[10m]))
      /
      sum(rate(secrets_http_requests_total{path="/v1/token"}[10m]))
    ) > 0.20
  for: 10m
  labels:
    severity: critical
  annotations:
    summary: "Token endpoint throttling critical"
    description: "More than 20% of /v1/token requests are returning 429"
```

## Disabling Metrics

To disable metrics collection, set `METRICS_ENABLED=false` in your environment:

```bash
export METRICS_ENABLED=false
```

When disabled:

- The `/metrics` endpoint is not registered (requests return 404 Not Found)
- No metrics are collected (zero overhead)
- HTTP metrics middleware is not applied
- Business metrics use a no-op implementation

## Troubleshooting

### Metrics endpoint returns 404

- Check that `METRICS_ENABLED=true` in your environment
- Verify that you are accessing the correct port (default is `8081`, e.g., `http://localhost:8081/metrics`)
- Verify the server is running and accessible

### Missing metrics

- Ensure Prometheus is scraping the `/metrics` endpoint
- Check that operations are actually being executed
- Verify the namespace matches your configuration (`METRICS_NAMESPACE`)

### High memory usage

- Review Prometheus scrape interval (recommend 15-30 seconds)
- Check for high cardinality labels (should not occur with proper configuration)
- Verify metrics are being scraped regularly

## See Also

- **[Metrics Reference](../../observability/metrics-reference.md)** - Complete catalog of all metrics, operations, and queries
- [Production Deployment](../deployment/docker-hardened.md)
- [Operator drills](../runbooks/README.md#operator-drills-quarterly)
- [Incident response guide](../observability/incident-response.md)
- [API rate limiting](../../concepts/api-fundamentals.md#rate-limiting)
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [Prometheus Documentation](https://prometheus.io/docs/)
