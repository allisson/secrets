# 📈 Application Scaling Guide

> **Document version**: v0.x
> Last updated: 2026-02-26
> **Audience**: Platform engineers, SRE teams, DevOps engineers
>
> **⚠️ UNTESTED PROCEDURES**: The procedures in this guide are reference examples and have not been tested in production. Always test in a non-production environment first and adapt to your infrastructure.

This guide covers horizontal and vertical scaling strategies for the Secrets application, from single-instance deployments to auto-scaling clusters.

## Table of Contents

- [Overview](#overview)
- [Scaling Patterns](#scaling-patterns)
- [Horizontal Scaling](#horizontal-scaling)
- [Vertical Scaling](#vertical-scaling)
- [Auto-Scaling](#auto-scaling)
- [Load Balancing](#load-balancing)
- [Performance Tuning](#performance-tuning)
- [Troubleshooting](#troubleshooting)
- [See Also](#see-also)

## Overview

### When to Scale

Scale when you observe:

| Metric | Threshold | Scaling Strategy |
|--------|-----------|------------------|
| **CPU usage** | > 70% sustained | Horizontal (add instances) or vertical (larger instance) |
| **Memory usage** | > 80% | Vertical scaling (more RAM) |
| **Request latency P95** | > 500ms | Horizontal scaling or performance tuning |
| **Request rate** | > 1000 req/s per instance | Horizontal scaling |
| **Database connections** | > 80% of pool | Horizontal scaling (more app instances) |

### Scaling Architecture

**Single instance** (development/small deployments):

```text
┌─────────┐      ┌──────────┐      ┌────────────┐
│ Clients │─────▶│ Secrets  │─────▶│ PostgreSQL │
└─────────┘      └──────────┘      └────────────┘
```

**Multi-instance** (production):

```text
┌─────────┐      ┌──────────────┐      ┌──────────┐      ┌────────────┐
│ Clients │─────▶│ Load Balancer│─────▶│ Secrets  │─────▶│ PostgreSQL │
└─────────┘      └──────────────┘      │ (3 inst) │      └────────────┘
                                        └──────────┘
```

**Auto-scaling** (high-traffic production):

```text
┌─────────┐      ┌──────────────┐      ┌──────────────┐      ┌────────────┐
│ Clients │─────▶│ Load Balancer│─────▶│ Secrets      │─────▶│ PostgreSQL │
└─────────┘      └──────────────┘      │ (3-10 inst)  │      └────────────┘
                                        │ (auto-scale) │
                                        └──────────────┘
```

## Scaling Patterns

### Pattern 1: Single Instance → Multi-Instance

**Use case**: Development → Production

**Steps**:

1. Deploy 3 instances for high availability
2. Add load balancer (ALB/NLB/GCP LB)
3. Configure health checks (`/health`, `/ready`)
4. Verify session-less operation (Secrets is stateless)

**Expected improvement**:

- 3x throughput (linear scaling)
- High availability (survive 1 instance failure)
- Zero-downtime deployments (rolling restart)

---

### Pattern 2: Multi-Instance → Auto-Scaling

**Use case**: Production → High-traffic production

**Steps**:

1. Configure auto-scaling group (AWS Auto Scaling, GCP Managed Instance Group)
2. Set min/max instance count (3-10 instances)
3. Configure scaling triggers (CPU > 70%, request rate > 1000/s)
4. Test scale-out and scale-in behavior

**Expected improvement**:

- Automatic capacity adjustment
- Cost optimization (scale down during low traffic)
- Handle traffic spikes without manual intervention

---

### Pattern 3: Regional → Multi-Regional

**Use case**: Geographic distribution, disaster recovery

**Steps**:

1. Deploy Secrets in multiple cloud regions
2. Configure global load balancer (Route 53, Cloud Load Balancing)
3. Replicate database across regions (read replicas or multi-region database)
4. Test regional failover

**Expected improvement**:

- Low-latency access from multiple geographies
- Disaster recovery (survive regional outage)
- Compliance (data residency requirements)

## Horizontal Scaling

### AWS Auto Scaling (EC2)

**Launch Template**:

```bash
aws ec2 create-launch-template \
  --launch-template-name secrets-app \
  --version-description "v0.10.0" \
  --launch-template-data '{
    "ImageId": "ami-0c55b159cbfafe1f0",
    "InstanceType": "t3.medium",
    "SecurityGroupIds": ["sg-12345678"],
    "UserData": "'"$(base64 -w0 startup-script.sh)"'"
  }'
```

**Auto Scaling Group**:

```bash
aws autoscaling create-auto-scaling-group \
  --auto-scaling-group-name secrets-asg \
  --launch-template LaunchTemplateName=secrets-app \
  --min-size 3 \
  --max-size 10 \
  --desired-capacity 3 \
  --vpc-zone-identifier "subnet-1,subnet-2,subnet-3" \
  --target-group-arns arn:aws:elasticloadbalancing:...
```

**Scaling Policy** (target tracking):

```bash
aws autoscaling put-scaling-policy \
  --auto-scaling-group-name secrets-asg \
  --policy-name cpu-target-tracking \
  --policy-type TargetTrackingScaling \
  --target-tracking-configuration '{
    "PredefinedMetricSpecification": {
      "PredefinedMetricType": "ASGAverageCPUUtilization"
    },
    "TargetValue": 70.0
  }'
```

### GCP Managed Instance Group

**Instance Template**:

```bash
gcloud compute instance-templates create secrets-template \
  --machine-type=n1-standard-2 \
  --image-family=debian-11 \
  --boot-disk-size=20GB \
  --metadata-from-file startup-script=startup.sh
```

**Managed Instance Group with Auto-Scaling**:

```bash
gcloud compute instance-groups managed create secrets-mig \
  --base-instance-name=secrets \
  --template=secrets-template \
  --size=3 \
  --zones=us-central1-a,us-central1-b,us-central1-c

gcloud compute instance-groups managed set-autoscaling secrets-mig \
  --min-num-replicas=3 \
  --max-num-replicas=10 \
  --target-cpu-utilization=0.7 \
  --cool-down-period=60
```

## Vertical Scaling

### When to Use Vertical Scaling

Use vertical scaling when:

- Single instance performance is bottleneck
- Memory usage grows with workload (caching, large objects)
- CPU-intensive operations (cryptographic operations)

### Docker Compose (Resource Limits)

**Increase container resources** (docker-compose.yml):

```yaml
services:
  secrets:
    image: allisson/secrets:<VERSION>
    deploy:
      resources:
        limits:
          cpus: '1.0'      # Increase from 0.5
          memory: 1G       # Increase from 512M
        reservations:
          cpus: '0.5'      # Increase from 0.25
          memory: 512M     # Increase from 256M
```

**Apply changes**:

```bash
docker-compose up -d
```

### AWS EC2 / GCP Compute Engine

**Resize instance**:

```bash
# AWS: Change instance type (requires stop/start)
aws ec2 stop-instances --instance-ids i-1234567890abcdef0
aws ec2 modify-instance-attribute \
  --instance-id i-1234567890abcdef0 \
  --instance-type t3.large
aws ec2 start-instances --instance-ids i-1234567890abcdef0

# GCP: Change machine type (requires stop/start)
gcloud compute instances stop secrets-vm
gcloud compute instances set-machine-type secrets-vm \
  --machine-type=n1-standard-4
gcloud compute instances start secrets-vm
```

## Auto-Scaling

### Auto-Scaling Best Practices

1. **Set appropriate min/max replicas**:

   - Min: 3 (high availability)
   - Max: Based on database connection pool (e.g., DB max 200 conns, 10 instances × 20 conns/inst = 200)

2. **Use multiple scaling metrics**:

   - CPU utilization (general load)
   - Memory utilization (detect memory leaks)
   - Request rate (traffic spikes)
   - Request latency (performance degradation)

3. **Configure scale-down stabilization**:

   - Wait 5-10 minutes before scaling down (avoid flapping)
   - Scale down gradually (50% max decrease per interval)

4. **Test scale-out and scale-in**:

   - Load test to trigger scale-out
   - Verify new instances receive traffic
   - Verify scale-in removes healthy instances gracefully

### Auto-Scaling Triggers

**Recommended triggers**:

| Metric | Threshold | Action |
|--------|-----------|--------|
| **CPU utilization** | > 70% for 3 min | Scale out |
| **Memory utilization** | > 80% for 3 min | Scale out |
| **Request rate** | > 1000 req/s per instance | Scale out |
| **Request latency P95** | > 500ms for 5 min | Scale out |
| **CPU utilization** | < 30% for 10 min | Scale in |

## Load Balancing

### Load Balancer Configuration

**AWS ALB**:

```bash
aws elbv2 create-target-group \
  --name secrets-tg \
  --protocol HTTP \
  --port 8080 \
  --vpc-id vpc-12345678 \
  --health-check-path /health \
  --health-check-interval-seconds 15 \
  --health-check-timeout-seconds 5 \
  --healthy-threshold-count 2 \
  --unhealthy-threshold-count 3
```

**GCP Load Balancer**:

```bash
gcloud compute health-checks create http secrets-health-check \
  --port=8080 \
  --request-path=/health \
  --check-interval=15s \
  --timeout=5s \
  --healthy-threshold=2 \
  --unhealthy-threshold=3
```

### Session Affinity (Not Required)

Secrets is **stateless** and does NOT require session affinity (sticky sessions). Each request is independent and can be routed to any instance.

## Performance Tuning

### Application-Level Tuning

**Go runtime settings** (environment variables):

```bash
# GOMAXPROCS: Number of OS threads (default: number of CPUs)
GOMAXPROCS=4

# GOGC: Garbage collection target percentage (default: 100)
# Higher value = less frequent GC, higher memory usage
GOGC=200
```

**Connection pool tuning**:

```bash
# Database connection pool (see Database Scaling Guide)
DB_MAX_OPEN_CONNS=50
DB_MAX_IDLE_CONNS=25
```

### Load Testing

**Use Apache Bench** (simple load test):

```bash
# 10,000 requests, 100 concurrent
ab -n 10000 -c 100 \
  -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/health
```

**Use k6** (realistic load test):

```javascript
import http from 'k6/http';

export let options = {
  stages: [
    { duration: '2m', target: 100 },  // Ramp to 100 users
    { duration: '5m', target: 100 },  // Stay at 100 users
    { duration: '2m', target: 0 },    // Ramp down
  ],
};

export default function () {
  http.get('http://localhost:8080/health');
}
```

Run k6:

```bash
k6 run loadtest.js
```

## Troubleshooting

### New instances not receiving traffic

**Symptoms**: Auto-scaling adds instances, but new instances show 0 requests

**Cause**: Health checks failing

**Solution**:

```bash
# Check container health (Docker)
docker ps
docker logs secrets-app | grep -i health

# Check health endpoint directly
curl http://localhost:8080/health
curl http://localhost:8080/ready

# Check load balancer target health (AWS)
aws elbv2 describe-target-health --target-group-arn <arn>
```

### Scaling flapping (rapid scale out/in)

**Symptoms**: Auto-scaling constantly scales between 3-10 instances

**Cause**: Insufficient stabilization window or aggressive scaling policies

**Solution**:

- Increase cooldown periods in auto-scaling configuration
- Adjust scaling thresholds to be less sensitive
- Add delay before scaling down (5-10 minutes)

### High latency despite horizontal scaling

**Symptoms**: P95 latency > 1s even with 10 instances

**Cause**: Database bottleneck, not application bottleneck

**Solution**:

- Scale database (see [Database Scaling Guide](database-scaling.md))
- Add database read replicas
- Optimize slow queries

### Memory usage grows over time

**Symptoms**: Memory usage climbs steadily, requiring restarts

**Cause**: Possible memory leak or unbounded caching

**Solution**:

- Enable memory profiling (`pprof`)
- Review application logs for leaks
- Set memory limits to force OOM restarts (temporary)

## See Also

- [Database Scaling Guide](database-scaling.md) - Database scaling complements application scaling
- [Docker Compose Deployment Guide](docker-compose.md) - Docker Compose deployment patterns
- [Health Check Endpoints](../observability/health-checks.md) - Health check configuration
- [Production Deployment Guide](docker-hardened.md) - Production scaling best practices
- [Monitoring Guide](../observability/monitoring.md) - Metrics for scaling decisions
