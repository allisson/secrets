# Metrics Reference

This document provides a complete reference for all Prometheus metrics exposed by the Secrets application.

**For setup and configuration**, see the [Monitoring Setup Guide](../operations/observability/monitoring.md).

## Table of Contents

- [Overview](#overview)
- [Metric Catalog](#metric-catalog)
  - [HTTP Metrics](#http-metrics)
  - [Business Operation Metrics](#business-operation-metrics)
- [Business Operations Reference](#business-operations-reference)
  - [Auth Domain](#auth-domain)
  - [Secrets Domain](#secrets-domain)
  - [Transit Domain](#transit-domain)
  - [Tokenization Domain](#tokenization-domain)
- [Prometheus Query Library](#prometheus-query-library)
  - [Request Rate Queries](#request-rate-queries)
  - [Latency Queries](#latency-queries)
  - [Error Rate Queries](#error-rate-queries)
  - [Rate Limiting Queries](#rate-limiting-queries)
  - [Tokenization Queries](#tokenization-queries)
  - [SLO Queries](#slo-queries)
- [Grafana Dashboards](#grafana-dashboards)
- [Metric Stability Contract](#metric-stability-contract)

## Overview

The Secrets application exposes metrics in Prometheus exposition format at `http://localhost:8081/metrics`. The metrics system uses OpenTelemetry for instrumentation with a Prometheus exporter.

**Key characteristics:**

- **Default namespace:** `secrets` (configurable via `METRICS_NAMESPACE`)
- **Low cardinality:** Labels are carefully chosen to prevent metric explosion
- **Stability:** Core metrics follow a stability contract (see below)
- **Zero overhead when disabled:** Set `METRICS_ENABLED=false` to disable

**Metric categories:**

1. **HTTP Metrics** - Request counts and durations for all API endpoints
2. **Business Operation Metrics** - Operation counts and durations for domain logic

## Metric Catalog

### HTTP Metrics

#### `{namespace}_http_requests_total`

| Field | Value |
|-------|-------|
| **Type** | Counter |
| **Description** | Total number of HTTP requests received by the server |
| **Unit** | Requests |
| **Labels** | `method` (GET, POST, PUT, DELETE), `path` (route pattern, e.g., `/v1/secrets/*path`), `status_code` (200, 201, 400, 404, 500, etc.) |
| **Cardinality** | Low (~50-100 combinations) |
| **Stability** | Stable |

**Example output:**

```prometheus
secrets_http_requests_total{method="GET",path="/v1/secrets/*path",status_code="200"} 1234
secrets_http_requests_total{method="POST",path="/v1/clients",status_code="201"} 56
secrets_http_requests_total{method="GET",path="/health",status_code="200"} 9999
secrets_http_requests_total{method="POST",path="/v1/token",status_code="429"} 12
```

**Common queries:**

```promql
# Total requests per second
rate(secrets_http_requests_total[5m])

# Requests per second by route
sum(rate(secrets_http_requests_total[5m])) by (path)

# Requests per second by status code
sum(rate(secrets_http_requests_total[5m])) by (status_code)

# Error rate (4xx and 5xx)
sum(rate(secrets_http_requests_total{status_code=~"4..|5.."}[5m]))

# Success rate percentage
sum(rate(secrets_http_requests_total{status_code=~"2.."}[5m])) / sum(rate(secrets_http_requests_total[5m])) * 100
```

---

#### `{namespace}_http_request_duration_seconds`

| Field | Value |
|-------|-------|
| **Type** | Histogram |
| **Description** | Duration of HTTP requests from receipt to response completion |
| **Unit** | Seconds |
| **Labels** | `method` (GET, POST, PUT, DELETE), `path` (route pattern), `status_code` (HTTP status code) |
| **Buckets** | Default OpenTelemetry histogram buckets |
| **Cardinality** | Low (~50-100 combinations) |
| **Stability** | Stable |

**Example output:**

```prometheus
secrets_http_request_duration_seconds_bucket{method="GET",path="/v1/secrets/*path",status_code="200",le="0.005"} 800
secrets_http_request_duration_seconds_bucket{method="GET",path="/v1/secrets/*path",status_code="200",le="0.01"} 1100
secrets_http_request_duration_seconds_bucket{method="GET",path="/v1/secrets/*path",status_code="200",le="0.025"} 1180
secrets_http_request_duration_seconds_bucket{method="GET",path="/v1/secrets/*path",status_code="200",le="0.05"} 1220
secrets_http_request_duration_seconds_bucket{method="GET",path="/v1/secrets/*path",status_code="200",le="0.1"} 1230
secrets_http_request_duration_seconds_bucket{method="GET",path="/v1/secrets/*path",status_code="200",le="+Inf"} 1234
secrets_http_request_duration_seconds_sum{method="GET",path="/v1/secrets/*path",status_code="200"} 6.789
secrets_http_request_duration_seconds_count{method="GET",path="/v1/secrets/*path",status_code="200"} 1234
```

**Common queries:**

```promql
# p50 latency across all routes
histogram_quantile(0.50, rate(secrets_http_request_duration_seconds_bucket[5m]))

# p95 latency by route
histogram_quantile(0.95, sum(rate(secrets_http_request_duration_seconds_bucket[5m])) by (le, path))

# p99 latency by route
histogram_quantile(0.99, sum(rate(secrets_http_request_duration_seconds_bucket[5m])) by (le, path))

# Average latency by route
rate(secrets_http_request_duration_seconds_sum[5m]) / rate(secrets_http_request_duration_seconds_count[5m])

# Slowest routes (by average latency)
topk(5, rate(secrets_http_request_duration_seconds_sum[5m]) / rate(secrets_http_request_duration_seconds_count[5m]))
```

---

### Business Operation Metrics

#### `{namespace}_operations_total`

| Field | Value |
|-------|-------|
| **Type** | Counter |
| **Description** | Total number of business operations executed (domain use case layer) |
| **Unit** | Operations |
| **Labels** | `domain` (auth, secrets, transit, tokenization), `operation` (e.g., client_create, secret_get, transit_encrypt), `status` (success, error) |
| **Cardinality** | Low (~60-80 combinations: 31 operations × 2 statuses) |
| **Stability** | Stable |

**Example output:**

```prometheus
secrets_operations_total{domain="auth",operation="client_create",status="success"} 42
secrets_operations_total{domain="auth",operation="client_create",status="error"} 3
secrets_operations_total{domain="secrets",operation="secret_get",status="success"} 1337
secrets_operations_total{domain="transit",operation="transit_encrypt",status="success"} 5678
secrets_operations_total{domain="tokenization",operation="tokenize",status="success"} 9012
```

**Common queries:**

```promql
# Operations per second by domain
sum(rate(secrets_operations_total[5m])) by (domain)

# Operations per second by operation
sum(rate(secrets_operations_total[5m])) by (operation)

# Error rate by domain
sum(rate(secrets_operations_total{status="error"}[5m])) by (domain)

# Error ratio by operation
sum(rate(secrets_operations_total{status="error"}[5m])) by (operation) 
/ 
sum(rate(secrets_operations_total[5m])) by (operation)

# Top 10 operations by volume
topk(10, sum(rate(secrets_operations_total[5m])) by (operation))
```

---

#### `{namespace}_operation_duration_seconds`

| Field | Value |
|-------|-------|
| **Type** | Histogram |
| **Description** | Duration of business operations (domain use case execution time) |
| **Unit** | Seconds |
| **Labels** | `domain` (auth, secrets, transit, tokenization), `operation` (operation name), `status` (success, error) |
| **Buckets** | Default OpenTelemetry histogram buckets |
| **Cardinality** | Low (~60-80 combinations) |
| **Stability** | Stable |

**Example output:**

```prometheus
secrets_operation_duration_seconds_bucket{domain="auth",operation="client_create",status="success",le="0.005"} 15
secrets_operation_duration_seconds_bucket{domain="auth",operation="client_create",status="success",le="0.01"} 28
secrets_operation_duration_seconds_bucket{domain="auth",operation="client_create",status="success",le="0.025"} 38
secrets_operation_duration_seconds_bucket{domain="auth",operation="client_create",status="success",le="0.05"} 41
secrets_operation_duration_seconds_bucket{domain="auth",operation="client_create",status="success",le="0.1"} 42
secrets_operation_duration_seconds_bucket{domain="auth",operation="client_create",status="success",le="+Inf"} 42
secrets_operation_duration_seconds_sum{domain="auth",operation="client_create",status="success"} 1.25
secrets_operation_duration_seconds_count{domain="auth",operation="client_create",status="success"} 42
```

**Common queries:**

```promql
# p95 operation latency by domain
histogram_quantile(0.95, sum(rate(secrets_operation_duration_seconds_bucket[5m])) by (le, domain))

# p95 operation latency by operation
histogram_quantile(0.95, sum(rate(secrets_operation_duration_seconds_bucket[5m])) by (le, operation))

# Average operation duration by operation
rate(secrets_operation_duration_seconds_sum[5m]) / rate(secrets_operation_duration_seconds_count[5m])

# Slowest operations (by average duration)
topk(5, rate(secrets_operation_duration_seconds_sum[5m]) / rate(secrets_operation_duration_seconds_count[5m]))

# Operations slower than 100ms (p95)
histogram_quantile(0.95, sum(rate(secrets_operation_duration_seconds_bucket[5m])) by (le, operation)) > 0.1
```

---

## Business Operations Reference

This section lists all 31 business operations instrumented across the 4 domains. The "Typical p95 Latency" values are approximate and may vary based on database performance, KMS latency, and workload characteristics.

### Auth Domain

| Operation | Description | Typical p95 Latency | Notes |
|-----------|-------------|---------------------|-------|
| `client_create` | Create new API client | < 50ms | Database write + password hashing (Argon2id) |
| `client_get` | Retrieve client by ID | < 20ms | Single database read |
| `client_update` | Update client configuration | < 40ms | Database write |
| `client_delete` | Delete API client | < 30ms | Database delete |
| `client_list` | List all clients with pagination | < 50ms | Database query, varies with page size |
| `client_unlock` | Unlock locked-out client account | < 30ms | Database write |
| `token_issue` | Issue authentication token | < 100ms | Password verification (Argon2id) + token generation |
| `token_authenticate` | Validate authentication token | < 20ms | Database lookup + token validation |
| `audit_log_create` | Record audit log entry | < 30ms | Database write + HMAC signature |
| `audit_log_list` | List audit logs with pagination | < 50ms | Database query, varies with page size |
| `audit_log_delete` | Delete audit logs older than retention | < 100ms | Bulk delete, varies with row count |
| `audit_log_verify` | Verify single audit log signature | < 10ms | HMAC verification (no database) |
| `audit_log_verify_batch` | Verify batch of audit log signatures | < 50ms | Multiple HMAC verifications |

#### Total Operations: 13

---

### Secrets Domain

| Operation | Description | Typical p95 Latency | Notes |
|-----------|-------------|---------------------|-------|
| `secret_create` | Create or update secret (new version) | < 80ms | KMS encrypt + database write |
| `secret_get` | Retrieve latest version of secret | < 60ms | Database read + KMS decrypt |
| `secret_get_version` | Retrieve secret by explicit version number | < 60ms | Database read + KMS decrypt |
| `secret_delete` | Soft-delete secret (sets deleted_at) | < 30ms | Database update |
| `secret_list` | List all secrets with pagination | < 50ms | Database query, varies with page size |
| `secret_purge` | Hard-delete soft-deleted secrets | < 100ms | Bulk delete, varies with row count |

#### Total Operations: 6

#### KMS Latency Impact

Add 10-50ms for KMS operations depending on provider (GCP KMS, AWS KMS, Azure Key Vault, etc.)

---

### Transit Domain

| Operation | Description | Typical p95 Latency | Notes |
|-----------|-------------|---------------------|-------|
| `transit_key_create` | Create new transit encryption key | < 80ms | KMS encrypt DEK + database write |
| `transit_key_rotate` | Rotate key to new version | < 80ms | KMS encrypt new DEK + database write |
| `transit_key_delete` | Delete transit key and all versions | < 40ms | Database delete |
| `transit_key_list` | List all transit keys with pagination | < 50ms | Database query, varies with page size |
| `transit_encrypt` | Encrypt data with transit key | < 60ms | Database read (DEK) + KMS decrypt DEK + AES-GCM encrypt |
| `transit_decrypt` | Decrypt data with transit key | < 60ms | Database read (DEK) + KMS decrypt DEK + AES-GCM decrypt |

#### Total Operations: 6

#### Encryption Overhead

AES-GCM and ChaCha20-Poly1305 are fast (~5ms for typical payloads < 1KB)

---

### Tokenization Domain

| Operation | Description | Typical p95 Latency | Notes |
|-----------|-------------|---------------------|-------|
| `tokenization_key_create` | Create new tokenization key | < 80ms | KMS encrypt DEK + database write |
| `tokenization_key_rotate` | Rotate key to new version | < 80ms | KMS encrypt new DEK + database write |
| `tokenization_key_delete` | Delete tokenization key | < 40ms | Database delete |
| `tokenization_key_list` | List tokenization keys with pagination | < 50ms | Database query, varies with page size |
| `tokenize` | Generate token for plaintext value | < 70ms | Database read (DEK) + KMS decrypt + encrypt + database write |
| `detokenize` | Resolve token back to plaintext | < 60ms | Database read (token + DEK) + KMS decrypt + decrypt |
| `validate` | Validate token lifecycle state | < 20ms | Database read |
| `revoke` | Revoke token (mark as revoked) | < 30ms | Database update |
| `cleanup_expired` | Delete expired tokens older than retention | < 100ms | Bulk delete, varies with row count |

#### Total Operations: 9

---

## Prometheus Query Library

This section provides copy-paste ready Prometheus queries organized by use case.

### Request Rate Queries

**Total requests per second:**

```promql
rate(secrets_http_requests_total[5m])
```

**Requests per second by route:**

```promql
sum(rate(secrets_http_requests_total[5m])) by (path)
```

**Requests per second by HTTP method:**

```promql
sum(rate(secrets_http_requests_total[5m])) by (method)
```

**Requests per second by status code:**

```promql
sum(rate(secrets_http_requests_total[5m])) by (status_code)
```

**Success rate (2xx responses) as percentage:**

```promql
sum(rate(secrets_http_requests_total{status_code=~"2.."}[5m])) 
/ 
sum(rate(secrets_http_requests_total[5m])) * 100
```

---

### Latency Queries

**p50 latency across all routes:**

```promql
histogram_quantile(0.50, rate(secrets_http_request_duration_seconds_bucket[5m]))
```

**p95 latency by route:**

```promql
histogram_quantile(0.95, sum(rate(secrets_http_request_duration_seconds_bucket[5m])) by (le, path))
```

**p99 latency by route:**

```promql
histogram_quantile(0.99, sum(rate(secrets_http_request_duration_seconds_bucket[5m])) by (le, path))
```

**Average latency by route:**

```promql
rate(secrets_http_request_duration_seconds_sum[5m]) / rate(secrets_http_request_duration_seconds_count[5m])
```

**Top 5 slowest routes (by average latency):**

```promql
topk(5, rate(secrets_http_request_duration_seconds_sum[5m]) / rate(secrets_http_request_duration_seconds_count[5m]))
```

**p95 operation latency by domain:**

```promql
histogram_quantile(0.95, sum(rate(secrets_operation_duration_seconds_bucket[5m])) by (le, domain))
```

**p95 operation latency by operation:**

```promql
histogram_quantile(0.95, sum(rate(secrets_operation_duration_seconds_bucket[5m])) by (le, operation))
```

**Operations with p95 latency > 100ms:**

```promql
histogram_quantile(0.95, sum(rate(secrets_operation_duration_seconds_bucket[5m])) by (le, operation)) > 0.1
```

---

### Error Rate Queries

**Total error rate (4xx and 5xx):**

```promql
sum(rate(secrets_http_requests_total{status_code=~"4..|5.."}[5m]))
```

**5xx error rate (server errors):**

```promql
sum(rate(secrets_http_requests_total{status_code=~"5.."}[5m]))
```

**4xx error rate (client errors):**

```promql
sum(rate(secrets_http_requests_total{status_code=~"4.."}[5m]))
```

**Error rate by route:**

```promql
sum(rate(secrets_http_requests_total{status_code=~"4..|5.."}[5m])) by (path)
```

**Error ratio (percentage of requests that are errors):**

```promql
sum(rate(secrets_http_requests_total{status_code=~"4..|5.."}[5m])) 
/ 
sum(rate(secrets_http_requests_total[5m])) * 100
```

**Business operation error rate by domain:**

```promql
sum(rate(secrets_operations_total{status="error"}[5m])) by (domain)
```

**Business operation error ratio by operation:**

```promql
sum(rate(secrets_operations_total{status="error"}[5m])) by (operation) 
/ 
sum(rate(secrets_operations_total[5m])) by (operation)
```

**Top 10 operations by error count:**

```promql
topk(10, sum(rate(secrets_operations_total{status="error"}[5m])) by (operation))
```

---

### Rate Limiting Queries

**429 rate (throttled requests) by route:**

```promql
sum(rate(secrets_http_requests_total{status_code="429"}[5m])) by (path)
```

**429 ratio (percentage of requests throttled) by route:**

```promql
sum(rate(secrets_http_requests_total{status_code="429"}[5m])) by (path)
/
sum(rate(secrets_http_requests_total[5m])) by (path)
```

**Global 429 ratio:**

```promql
sum(rate(secrets_http_requests_total{status_code="429"}[5m]))
/
sum(rate(secrets_http_requests_total[5m]))
```

**Token endpoint 429 ratio:**

```promql
sum(rate(secrets_http_requests_total{path="/v1/token",status_code="429"}[5m]))
/
sum(rate(secrets_http_requests_total{path="/v1/token"}[5m]))
```

**Token endpoint request rate by status:**

```promql
sum(rate(secrets_http_requests_total{path="/v1/token"}[5m])) by (status_code)
```

**Token issuance success ratio:**

```promql
sum(rate(secrets_http_requests_total{path="/v1/token",status_code="201"}[5m]))
/
sum(rate(secrets_http_requests_total{path="/v1/token"}[5m]))
```

**403 (Forbidden/denied authorization) rate by route:**

```promql
sum(rate(secrets_http_requests_total{status_code="403"}[5m])) by (path)
```

---

### Tokenization Queries

**Tokenization operations per second:**

```promql
sum(rate(secrets_operations_total{domain="tokenization"}[5m])) by (operation)
```

**Tokenize error rate:**

```promql
rate(secrets_operations_total{domain="tokenization",operation="tokenize",status="error"}[5m])
/
rate(secrets_operations_total{domain="tokenization",operation="tokenize"}[5m])
```

**Detokenize error rate:**

```promql
rate(secrets_operations_total{domain="tokenization",operation="detokenize",status="error"}[5m])
/
rate(secrets_operations_total{domain="tokenization",operation="detokenize"}[5m])
```

**Tokenization p95 latency (tokenize endpoint):**

```promql
histogram_quantile(
  0.95,
  sum by (le) (
    rate(secrets_http_request_duration_seconds_bucket{path="/v1/tokenization/keys/:name/tokenize"}[5m])
  )
)
```

**Detokenization p95 latency:**

```promql
histogram_quantile(
  0.95,
  sum by (le) (
    rate(secrets_http_request_duration_seconds_bucket{path="/v1/tokenization/detokenize"}[5m])
  )
)
```

**Expired token cleanup throughput (operations per second):**

```promql
rate(secrets_operations_total{domain="tokenization",operation="cleanup_expired",status="success"}[15m])
```

**Token revocation rate:**

```promql
rate(secrets_operations_total{domain="tokenization",operation="revoke",status="success"}[5m])
```

---

### SLO Queries

**API availability (percentage of non-5xx responses):**

```promql
sum(rate(secrets_http_requests_total{status_code!~"5.."}[5m]))
/
sum(rate(secrets_http_requests_total[5m])) * 100
```

**Secrets engine availability (success rate):**

```promql
sum(rate(secrets_operations_total{domain="secrets",status="success"}[5m]))
/
sum(rate(secrets_operations_total{domain="secrets"}[5m])) * 100
```

**Transit encryption availability (success rate):**

```promql
sum(rate(secrets_operations_total{domain="transit",operation=~"transit_encrypt|transit_decrypt",status="success"}[5m]))
/
sum(rate(secrets_operations_total{domain="transit",operation=~"transit_encrypt|transit_decrypt"}[5m])) * 100
```

**API latency SLO compliance (p95 < 300ms):**

```promql
histogram_quantile(0.95, rate(secrets_http_request_duration_seconds_bucket[5m])) < 0.3
```

**Tokenization SLO: p95 tokenize latency < 300ms:**

```promql
histogram_quantile(
  0.95,
  sum by (le) (
    rate(secrets_http_request_duration_seconds_bucket{path="/v1/tokenization/keys/:name/tokenize"}[5m])
  )
) < 0.3
```

**Tokenization SLO: p95 detokenize latency < 400ms:**

```promql
histogram_quantile(
  0.95,
  sum by (le) (
    rate(secrets_http_request_duration_seconds_bucket{path="/v1/tokenization/detokenize"}[5m])
  )
) < 0.4
```

**Tokenization SLO: error rate < 0.2%:**

```promql
sum(rate(secrets_operations_total{domain="tokenization",status="error"}[5m]))
/
sum(rate(secrets_operations_total{domain="tokenization"}[5m])) < 0.002
```

---

## Grafana Dashboards

### Available Dashboards

Pre-built Grafana dashboard JSON files are available in the repository:

| Dashboard | Location | Description |
|-----------|----------|-------------|
| **Secrets Overview** | `docs/operations/dashboards/secrets-overview.json` | Baseline request rate, error rate, and p95 latency view |
| **Rate Limiting** | `docs/operations/dashboards/secrets-rate-limiting.json` | 429 behavior and throttle pressure analysis |

### Import Instructions

1. Open Grafana UI (default: `http://localhost:3000`)
2. Navigate to **Dashboards** → **Import**
3. Click **Upload JSON file**
4. Select one of the dashboard files from `docs/operations/dashboards/`
5. Select your Prometheus datasource
6. Click **Import**

### Recommended Panels

When creating custom dashboards, consider including these panels:

| Panel Type | Metric | Description |
|------------|--------|-------------|
| **Time Series** | `rate(secrets_http_requests_total[5m])` | Request rate by route |
| **Time Series** | `histogram_quantile(0.95, ...)` | p95 latency by route |
| **Stat** | `sum(rate(secrets_http_requests_total{status_code=~"5.."}[5m]))` | Current 5xx error rate |
| **Gauge** | API availability SLO query | Availability percentage with thresholds |
| **Table** | `topk(10, sum(rate(...)) by (operation))` | Top 10 operations by volume |
| **Heatmap** | `secrets_http_request_duration_seconds_bucket` | Latency distribution |
| **Time Series** | `sum(rate(...{status_code="429"}[5m])) by (path)` | Rate limiting pressure |

**Panel configuration tips:**

- Use 5-minute rate windows for responsiveness
- Set appropriate Y-axis units (seconds for latency, ops/sec for rates)
- Add threshold lines for SLOs (e.g., 300ms line on latency panels)
- Use legend format with label templates (e.g., `{{path}}`)

---

## Metric Stability Contract

### Stability Levels

**Stable:** Metrics marked as "Stable" in this document follow these guarantees:

- Metric names will not change
- Label names will not change
- Label value semantics will not change
- New label values may be added (additive changes are non-breaking)
- New metrics may be added

**Breaking changes** (metric/label renaming or removal) will only occur in major version releases (v2.0.0+).

### Current Stable Metrics

All 4 metrics documented in this reference are **Stable**:

- `{namespace}_http_requests_total`
- `{namespace}_http_request_duration_seconds`
- `{namespace}_operations_total`
- `{namespace}_operation_duration_seconds`

### Deprecation Policy

If a metric needs to be deprecated:

1. Deprecation will be announced in `CHANGELOG.md` at least one minor version before removal
2. The old metric will be kept alongside the new metric for at least one minor version
3. Documentation will indicate the deprecation and migration path
4. Removal will only occur in a major version release

### Non-Breaking Changes

The following are considered **non-breaking** and may occur in minor/patch releases:

- Adding new metrics
- Adding new label values to existing labels (e.g., new `operation` values)
- Adding new optional labels (rare, but not breaking for existing queries)
- Changing metric descriptions or documentation
- Changing default histogram buckets (existing queries continue to work)

### Impact on Dashboards and Alerts

**Stable metrics** mean your dashboards and alerts will not break across patch and minor version upgrades. When upgrading to a new major version (e.g., v1.x to v2.x), review the `CHANGELOG.md` for any metric changes and update your queries accordingly.

### Configuration Changes

The following configuration options may change the metric behavior but do not violate stability:

- `METRICS_NAMESPACE` - Changes the namespace prefix (e.g., `secrets_` → `myapp_`)
- `METRICS_ENABLED=false` - Disables all metrics (no metrics exposed)

These are user-controlled and do not represent a breaking change in the metric contract.

---

## See Also

- **[Monitoring Setup Guide](../operations/observability/monitoring.md)** - How to configure Prometheus, Grafana, and alerting
- **[Health Check Endpoints](../operations/observability/health-checks.md)** - Liveness and readiness probes
- **[Incident Response Guide](../operations/observability/incident-response.md)** - Production troubleshooting runbook
- **[Configuration Reference](../configuration.md)** - All environment variables including metrics config
- **[OpenTelemetry Documentation](https://opentelemetry.io/docs/)** - Upstream metrics SDK documentation
- **[Prometheus Documentation](https://prometheus.io/docs/)** - Query language and best practices
