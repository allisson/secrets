# Observability

This directory contains observability and monitoring documentation for the Secrets application.

## Available Documentation

### [Metrics Reference](metrics-reference.md)

Complete reference catalog for all Prometheus metrics exposed by the application.

**Use this for:**

- Looking up metric definitions and labels
- Finding ready-to-use Prometheus queries
- Understanding business operations and typical latencies
- Checking metric stability guarantees

**Contents:**

- Detailed metric catalog (HTTP and business operation metrics)
- Complete list of 31 instrumented operations across 4 domains
- Prometheus query library organized by use case
- Grafana dashboard guidance
- Metric stability contract

---

### [Monitoring Setup Guide](../operations/observability/monitoring.md)

How-to guide for setting up Prometheus, Grafana, and alerting for the Secrets application.

**Use this for:**

- Initial monitoring setup (Prometheus + Grafana quickstart)
- Configuring metrics collection
- Creating alerts and dashboards
- Troubleshooting metrics issues

**Contents:**

- Configuration instructions
- Prometheus scrape configuration
- Grafana dashboard setup
- Alert rule examples
- Troubleshooting guide

---

### [Health Check Endpoints](../operations/observability/health-checks.md)

Documentation for liveness and readiness probe endpoints.

**Use this for:**

- Kubernetes/container orchestration health probes
- Load balancer health checks
- Understanding health check behavior

**Contents:**

- `/health` endpoint (liveness probe)
- `/health/ready` endpoint (readiness probe)
- Integration with orchestration platforms

---

### [Incident Response Guide](../operations/observability/incident-response.md)

Production troubleshooting runbook for responding to incidents.

**Use this for:**

- Responding to production alerts
- Debugging performance issues
- Root cause analysis
- Emergency procedures

**Contents:**

- Incident response procedures
- Common failure scenarios
- Diagnostic commands
- Recovery procedures

---

## Quick Links

| Task | Documentation |
|------|---------------|
| Find a specific metric | [Metrics Reference - Metric Catalog](metrics-reference.md#metric-catalog) |
| Set up Prometheus | [Monitoring Setup - Prometheus Configuration](../operations/observability/monitoring.md#prometheus-configuration) |
| Create alerts | [Monitoring Setup - Alerting](../operations/observability/monitoring.md#alerting) |
| Query examples | [Metrics Reference - Query Library](metrics-reference.md#prometheus-query-library) |
| Configure health probes | [Health Check Endpoints](../operations/observability/health-checks.md) |
| Troubleshoot outage | [Incident Response Guide](../operations/observability/incident-response.md) |

---

## Related Documentation

- **[Configuration Reference](../configuration.md)** - Environment variables for metrics configuration
- **[API Fundamentals](../concepts/api-fundamentals.md)** - Rate limiting and API behavior
- **[Production Deployment](../operations/deployment/)** - Deployment guides with observability setup

---

## External Resources

- **[OpenTelemetry Documentation](https://opentelemetry.io/docs/)** - Metrics SDK reference
- **[Prometheus Documentation](https://prometheus.io/docs/)** - Query language and best practices
- **[Grafana Documentation](https://grafana.com/docs/)** - Dashboard creation and visualization
