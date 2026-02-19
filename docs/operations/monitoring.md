# 📊 Monitoring

> Last updated: 2026-02-19

This document describes the metrics instrumentation and monitoring capabilities in the Secrets application.

## Overview

The application uses OpenTelemetry for metrics instrumentation with a Prometheus-compatible export endpoint. Metrics can be enabled/disabled via configuration and cover two main areas:

1. **Business Operations** - Domain-specific operation counters and durations
2. **HTTP Requests** - Request counts and response times

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
2. Start Prometheus with a scrape config for `http://host.docker.internal:8080/metrics`
3. Open Grafana and create panels from Prometheus queries

Note: On Linux, replace `host.docker.internal` with the host IP reachable from your Docker network.

Minimal `prometheus.yml`:

```yaml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: "secrets"
    static_configs:
      - targets: ["host.docker.internal:8080"]
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

## Metrics Endpoint

The metrics are exposed at the `/metrics` endpoint in Prometheus exposition format:

```bash
curl http://localhost:8080/metrics
```

**Key Points:**

- **No Authentication Required** - The `/metrics` endpoint is public (standard Prometheus practice)
- **Prometheus Compatible** - Supports both text format and OpenMetrics format
- **Located Outside API Versioning** - Available at `/metrics`, not `/v1/metrics`

## Available Metrics

### Metrics Naming Contract

These metrics are treated as stable for dashboard and alert compatibility.

| Metric | Type | Labels | Stability |
| --- | --- | --- | --- |
| `{namespace}_http_requests_total` | Counter | `method`, `path`, `status_code` | Stable |
| `{namespace}_http_request_duration_seconds` | Histogram | `method`, `path`, `status_code` | Stable |
| `{namespace}_operations_total` | Counter | `domain`, `operation`, `status` | Stable |
| `{namespace}_operation_duration_seconds` | Histogram | `domain`, `operation`, `status` | Stable |

Compatibility note:

- Renaming metric names or labels is considered a breaking observability change
- Additive changes (new metrics/new label values) are generally non-breaking

### Business Operation Metrics

#### `{namespace}_operations_total`

**Type:** Counter  
**Description:** Total number of business operations executed  
**Labels:**

- `domain` - Business domain (auth, secrets, transit, tokenization)
- `operation` - Operation name (e.g., client_create, secret_get, transit_encrypt)
- `status` - Operation result (success, error)

**Example:**

```prometheus
secrets_operations_total{domain="auth",operation="client_create",status="success"} 42
secrets_operations_total{domain="secrets",operation="secret_get",status="success"} 1337
secrets_operations_total{domain="transit",operation="transit_key_rotate",status="error"} 2
```

#### `{namespace}_operation_duration_seconds`

**Type:** Histogram  
**Description:** Duration of business operations in seconds  
**Labels:**

- `domain` - Business domain (auth, secrets, transit, tokenization)
- `operation` - Operation name
- `status` - Operation result (success, error)

**Example:**

```prometheus
secrets_operation_duration_seconds_bucket{domain="auth",operation="client_create",status="success",le="0.005"} 15
secrets_operation_duration_seconds_bucket{domain="auth",operation="client_create",status="success",le="0.01"} 28
secrets_operation_duration_seconds_sum{domain="auth",operation="client_create",status="success"} 1.25
secrets_operation_duration_seconds_count{domain="auth",operation="client_create",status="success"} 42
```

### HTTP Request Metrics

#### `{namespace}_http_requests_total`

**Type:** Counter  
**Description:** Total number of HTTP requests received  
**Labels:**

- `method` - HTTP method (GET, POST, PUT, DELETE)
- `path` - Route pattern (e.g., `/v1/secrets/*path`, `/v1/tokenization/keys/:name/tokenize`)
- `status_code` - HTTP status code (200, 404, 500, etc.)

Route-template note:

- OpenAPI pages use `{name}` parameter syntax
- Runtime HTTP metrics typically expose Gin-style patterns like `:name` and wildcard `*path`

**Example:**

```prometheus
secrets_http_requests_total{method="GET",path="/v1/secrets/*path",status_code="200"} 1234
secrets_http_requests_total{method="POST",path="/v1/clients",status_code="201"} 56
secrets_http_requests_total{method="GET",path="/health",status_code="200"} 9999
```

#### `{namespace}_http_request_duration_seconds`

**Type:** Histogram  
**Description:** Duration of HTTP requests in seconds  
**Labels:**

- `method` - HTTP method
- `path` - Route pattern
- `status_code` - HTTP status code

**Example:**

```prometheus
secrets_http_request_duration_seconds_bucket{method="GET",path="/v1/secrets/*path",status_code="200",le="0.005"} 800
secrets_http_request_duration_seconds_bucket{method="GET",path="/v1/secrets/*path",status_code="200",le="0.01"} 1100
secrets_http_request_duration_seconds_sum{method="GET",path="/v1/secrets/*path",status_code="200"} 6.789
secrets_http_request_duration_seconds_count{method="GET",path="/v1/secrets/*path",status_code="200"} 1234
```

## Business Domains and Operations

### Auth Domain

| Operation | Description |
|-----------|-------------|
| `client_create` | Create new API client |
| `client_get` | Retrieve client by ID |
| `client_update` | Update client configuration |
| `client_delete` | Delete API client |
| `client_list` | List all clients |
| `token_issue` | Issue authentication token |
| `token_authenticate` | Validate token |
| `audit_log_create` | Record audit log entry |
| `audit_log_list` | List audit logs |
| `audit_log_delete` | Delete audit logs older than retention |

### Secrets Domain

| Operation | Description |
|-----------|-------------|
| `secret_create` | Create or update secret |
| `secret_get` | Retrieve secret value |
| `secret_get_version` | Retrieve secret by explicit version |
| `secret_delete` | Delete secret |

### Transit Domain

| Operation | Description |
|-----------|-------------|
| `transit_key_create` | Create new transit key |
| `transit_key_rotate` | Rotate transit key to new version |
| `transit_key_delete` | Delete transit key |
| `transit_encrypt` | Encrypt data with transit key |
| `transit_decrypt` | Decrypt data with transit key |

### Tokenization Domain

| Operation | Description |
|-----------|-------------|
| `tokenization_key_create` | Create new tokenization key |
| `tokenization_key_rotate` | Rotate tokenization key to new version |
| `tokenization_key_delete` | Delete tokenization key |
| `tokenize` | Generate token for plaintext |
| `detokenize` | Resolve token back to plaintext |
| `validate` | Validate token lifecycle state |
| `revoke` | Revoke token |
| `cleanup_expired` | Delete expired tokens older than retention |

## Prometheus Configuration

### Scrape Configuration

Add the Secrets application to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'secrets'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 15s
```

### Example Queries

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

### Rate Limiting Observability Queries

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

Rate-limit interpretation notes:

- Stable low-volume `429` can be normal under bursty workloads
- Rising `429` with rising latency usually indicates saturation or mis-tuned clients
- Tune `RATE_LIMIT_REQUESTS_PER_SEC` and `RATE_LIMIT_BURST` only after retry behavior is verified

### Tokenization-focused Queries

**Detokenize error rate (5m):**

```promql
rate(secrets_operations_total{domain="tokenization",operation="detokenize",status="error"}[5m])
/
rate(secrets_operations_total{domain="tokenization",operation="detokenize"}[5m])
```

**Tokenization p95 latency (tokenize path):**

```promql
histogram_quantile(
  0.95,
  sum by (le) (
    rate(secrets_http_request_duration_seconds_bucket{path="/v1/tokenization/keys/:name/tokenize"}[5m])
  )
)
```

**Expired-token cleanup throughput (rows per second):**

```promql
rate(secrets_operations_total{domain="tokenization",operation="cleanup_expired",status="success"}[15m])
```

### SLO Starters (Tokenization)

- `POST /v1/tokenization/keys/:name/tokenize` latency: p95 < 300 ms
- `POST /v1/tokenization/detokenize` latency: p95 < 400 ms
- Tokenization server errors: < 0.2% across tokenization operations

## Grafana Dashboard

Starter dashboard artifacts:

- [Dashboard artifacts index](dashboards/README.md)
- [Secrets overview dashboard JSON](dashboards/secrets-overview.json)
- [Secrets rate-limiting dashboard JSON](dashboards/secrets-rate-limiting.json)

### Recommended Panels

1. **Request Rate** - Line graph showing HTTP requests/sec
2. **Error Rate** - Percentage of failed operations by domain
3. **Latency Heatmap** - Distribution of request durations
4. **Operation Counts** - Table showing top operations by volume
5. **Status Code Distribution** - Pie chart of HTTP status codes

### Example Panel Query (Request Rate)

```promql
sum(rate(secrets_http_requests_total[5m])) by (method, path)
```

## Alerting

### Recommended Alerts

### Ownership and Escalation Guidance

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

## Performance Considerations

- **Low Overhead** - OpenTelemetry metrics have minimal performance impact
- **Cardinality Control** - Labels are carefully chosen to avoid high cardinality
  - Paths use route patterns (e.g., `/v1/secrets/*path`) instead of actual values
  - Operations are predefined, not dynamic
  - Status values are limited to "success" and "error"
- **Memory Usage** - Metrics are stored in-memory until scraped by Prometheus

## Troubleshooting

### Metrics endpoint returns 404

- Check that `METRICS_ENABLED=true` in your environment
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

- [Production Deployment](production.md)
- [Operator drills](operator-drills.md)
- [Failure Playbooks](failure-playbooks.md)
- [API rate limiting](../api/rate-limiting.md)
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [Prometheus Documentation](https://prometheus.io/docs/)
