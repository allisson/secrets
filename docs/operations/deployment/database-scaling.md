# 🗄️ Database Scaling Guide

> **Document version**: v0.x
> Last updated: 2026-02-25
> **Audience**: DBAs, SRE teams, platform engineers
>
> **⚠️ UNTESTED PROCEDURES**: The procedures in this guide are reference examples and have not been tested in production. Always test in a non-production environment first and adapt to your infrastructure.

This guide covers database scaling strategies for Secrets, including connection pooling, read replicas, audit log management, and performance optimization.

## Table of Contents

- [Overview](#overview)
- [Connection Pooling](#connection-pooling)
- [Read Replicas](#read-replicas)
- [Audit Log Management](#audit-log-management)
- [Query Optimization](#query-optimization)
- [Monitoring](#monitoring)
- [Troubleshooting](#troubleshooting)
- [See Also](#see-also)

## Overview

### Scaling Challenges

As your Secrets deployment grows, you may encounter:

- **Connection exhaustion**: Database connection pool depleted under load
- **Slow queries**: Large audit log tables cause slow SELECT queries
- **Write contention**: High audit log write volume impacts transaction throughput
- **Storage growth**: Audit logs consume increasing disk space

### Scaling Metrics

| Metric | Threshold | Action |
|--------|-----------|--------|
| **Active connections** | > 80% of max | Increase connection pool or database max connections |
| **Query latency P95** | > 100ms | Add indexes, optimize queries, or add read replicas |
| **Audit log table size** | > 100GB | Archive old logs, partition table, or separate database |
| **Database CPU** | > 70% | Vertical scaling (larger instance) or read replicas |
| **Disk IOPS** | > 80% of provisioned | Increase IOPS or use faster storage |

## Connection Pooling

### Built-in Connection Pool

Secrets uses `database/sql` connection pooling (Go standard library):

```bash
# Environment variables for connection pooling
DB_MAX_OPEN_CONNS=25        # Max connections to database (default: 25)
DB_MAX_IDLE_CONNS=10        # Max idle connections in pool (default: 10)
DB_CONN_MAX_LIFETIME=3600   # Max connection lifetime in seconds (default: 1 hour)
DB_CONN_MAX_IDLE_TIME=1800  # Max idle connection time in seconds (default: 30 min)
```

### Tuning Guidelines

**Small deployment** (< 1000 req/min):

```bash
DB_MAX_OPEN_CONNS=10
DB_MAX_IDLE_CONNS=5
```

**Medium deployment** (1000-10000 req/min):

```bash
DB_MAX_OPEN_CONNS=50
DB_MAX_IDLE_CONNS=25
```

**Large deployment** (> 10000 req/min):

```bash
DB_MAX_OPEN_CONNS=100
DB_MAX_IDLE_CONNS=50
```

**Formula**:

```text
DB_MAX_OPEN_CONNS = (number of application instances) × (connections per instance)

Example:
- 5 application instances
- 20 connections per instance
- DB_MAX_OPEN_CONNS = 5 × 20 = 100
```

### Database Max Connections

Ensure database `max_connections` > total application pool size:

**PostgreSQL**:

```sql
-- Check current max_connections
SHOW max_connections;

-- Increase max_connections (requires restart)
ALTER SYSTEM SET max_connections = 200;
SELECT pg_reload_conf();
```

**MySQL**:

```sql
-- Check current max_connections
SHOW VARIABLES LIKE 'max_connections';

-- Increase max_connections
SET GLOBAL max_connections = 200;
```

### External Connection Pooler (PgBouncer)

For PostgreSQL, use PgBouncer for connection pooling:

```bash
# PgBouncer configuration
[databases]
secrets = host=postgres port=5432 dbname=secrets

[pgbouncer]
pool_mode = transaction
max_client_conn = 1000
default_pool_size = 25
reserve_pool_size = 5
```

**Application configuration**:

```bash
# Point to PgBouncer instead of PostgreSQL directly
DB_CONNECTION_STRING=postgres://user:pass@pgbouncer:6432/secrets
```

## Read Replicas

### When to Use Read Replicas

Use read replicas when:

- Read queries (audit log searches) cause primary database load
- Reporting/analytics queries impact production performance
- Geographic distribution requires low-latency reads

**NOTE**: Secrets does NOT currently support automatic read replica routing. You must implement custom logic or use database proxy.

### Read Replica Setup

**PostgreSQL (AWS RDS)**:

```bash
# Create read replica
aws rds create-db-instance-read-replica \
  --db-instance-identifier secrets-db-replica-1 \
  --source-db-instance-identifier secrets-db-primary \
  --db-instance-class db.t3.medium \
  --availability-zone us-east-1b
```

**PostgreSQL (GCP Cloud SQL)**:

```bash
gcloud sql instances create secrets-db-replica-1 \
  --master-instance-name=secrets-db-primary \
  --replica-type=READ \
  --tier=db-n1-standard-2 \
  --zone=us-central1-b
```

**MySQL (AWS RDS)**:

```bash
aws rds create-db-instance-read-replica \
  --db-instance-identifier secrets-db-replica-1 \
  --source-db-instance-identifier secrets-db-primary
```

### Read Replica Usage Patterns

**Pattern 1: Separate read-only endpoints** (manual routing):

```bash
# Primary (writes)
export DB_WRITE_CONNECTION_STRING=postgres://primary-db:5432/secrets

# Replica (reads)
export DB_READ_CONNECTION_STRING=postgres://replica-db:5432/secrets
```

**Pattern 2: Database proxy** (automatic routing):

Use ProxySQL (MySQL) or PgPool-II (PostgreSQL) to route reads to replicas.

**Pattern 3: Dedicated analytics database**:

Export audit logs to separate analytics database (Redshift, BigQuery) for reporting.

## Audit Log Management

### Audit Log Growth

Audit logs are append-only and can grow rapidly:

| Operations/day | Audit log rows/day | Disk growth/month (estimate) |
|----------------|-------------------|------------------------------|
| 10,000 | 10,000 | ~100 MB |
| 100,000 | 100,000 | ~1 GB |
| 1,000,000 | 1,000,000 | ~10 GB |

### Archiving Strategy

**Option 1: Partition by date** (PostgreSQL 10+):

```sql
-- Create partitioned audit_logs table
CREATE TABLE audit_logs_partitioned (
  id UUID PRIMARY KEY,
  client_id UUID NOT NULL,
  created_at TIMESTAMP NOT NULL,
  -- other columns
) PARTITION BY RANGE (created_at);

-- Create monthly partitions
CREATE TABLE audit_logs_2026_01 PARTITION OF audit_logs_partitioned
  FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');

CREATE TABLE audit_logs_2026_02 PARTITION OF audit_logs_partitioned
  FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
```

**Option 2: Export old logs to S3/GCS**:

```bash
# Export logs older than 90 days
psql -h localhost -U secrets -d secrets -c \
  "COPY (SELECT * FROM audit_logs WHERE created_at < NOW() - INTERVAL '90 days') 
   TO STDOUT WITH CSV HEADER" \
  | gzip > audit-logs-archive-$(date +%Y%m%d).csv.gz

# Upload to S3
aws s3 cp audit-logs-archive-$(date +%Y%m%d).csv.gz \
  s3://my-audit-logs-archive/

# Delete exported logs
psql -h localhost -U secrets -d secrets -c \
  "DELETE FROM audit_logs WHERE created_at < NOW() - INTERVAL '90 days';"
```

**Option 3: Separate audit log database**:

Create dedicated database for audit logs, reducing load on primary database:

```sql
-- Create separate database
CREATE DATABASE secrets_audit_logs;

-- Move audit_logs table to separate database
-- (Requires application schema changes - not currently supported)
```

### Audit Log Indexes

Add indexes to speed up common audit log queries:

```sql
-- Index by client_id (most common filter)
CREATE INDEX idx_audit_logs_client_id ON audit_logs(client_id);

-- Index by created_at (time-range queries)
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- Composite index (client + time range)
CREATE INDEX idx_audit_logs_client_created ON audit_logs(client_id, created_at DESC);
```

## Query Optimization

### Slow Query Identification

**PostgreSQL**:

```sql
-- Enable slow query logging
ALTER SYSTEM SET log_min_duration_statement = 1000; -- 1 second
SELECT pg_reload_conf();

-- View slow queries
SELECT query, calls, total_time, mean_time
FROM pg_stat_statements
WHERE mean_time > 100
ORDER BY total_time DESC
LIMIT 10;
```

**MySQL**:

```sql
-- Enable slow query log
SET GLOBAL slow_query_log = 'ON';
SET GLOBAL long_query_time = 1; -- 1 second

-- View slow queries
SELECT * FROM mysql.slow_log
ORDER BY query_time DESC
LIMIT 10;
```

### Common Optimization Patterns

**Pattern 1: Add indexes on foreign keys**:

```sql
-- Secrets uses foreign keys but may need additional indexes
CREATE INDEX idx_secrets_kek_id ON secrets(kek_id);
CREATE INDEX idx_transit_key_versions_key_id ON transit_key_versions(transit_key_id);
```

**Pattern 2: Analyze query plans**:

```sql
-- PostgreSQL
EXPLAIN ANALYZE SELECT * FROM secrets WHERE kek_id = 'uuid';

-- MySQL
EXPLAIN SELECT * FROM secrets WHERE kek_id = 'uuid';
```

**Pattern 3: Use LIMIT on large result sets**:

```sql
-- Bad: Returns all audit logs (millions of rows)
SELECT * FROM audit_logs;

-- Good: Returns recent 100 audit logs
SELECT * FROM audit_logs ORDER BY created_at DESC LIMIT 100;
```

## Monitoring

### Key Database Metrics

| Metric | Source | Alert Threshold |
|--------|--------|-----------------|
| **Connection count** | `SELECT COUNT(*) FROM pg_stat_activity` | > 80% of max |
| **Active transactions** | `SELECT COUNT(*) FROM pg_stat_activity WHERE state = 'active'` | > 50 |
| **Database size** | `SELECT pg_database_size('secrets')` | > 80% of disk |
| **Table size** | `SELECT pg_total_relation_size('audit_logs')` | > 50GB |
| **Slow queries** | `pg_stat_statements` | > 10 queries/min > 1s |
| **Replication lag** | `SELECT extract(epoch from (now() - pg_last_xact_replay_timestamp()))` | > 60 seconds |

### Monitoring Queries

**PostgreSQL connection usage**:

```sql
SELECT 
  COUNT(*) as connections,
  (SELECT setting::int FROM pg_settings WHERE name = 'max_connections') as max_connections,
  ROUND(COUNT(*)::numeric / (SELECT setting::int FROM pg_settings WHERE name = 'max_connections') * 100, 2) as pct_used
FROM pg_stat_activity;
```

**Table sizes**:

```sql
SELECT 
  schemaname,
  tablename,
  pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

## Troubleshooting

### Connection pool exhausted

**Symptoms**:

```text
ERROR: could not create connection: dial tcp: connection refused
ERROR: pq: sorry, too many clients already
```

**Solution**:

```bash
# Increase application connection pool
DB_MAX_OPEN_CONNS=50

# Or increase database max_connections
ALTER SYSTEM SET max_connections = 200;
```

### Slow audit log queries

**Symptoms**: Audit log queries take > 5 seconds

**Solution**:

```sql
-- Add indexes
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- Or partition table
-- (See Audit Log Management section)
```

### Database CPU at 100%

**Symptoms**: Database CPU constantly at 100%, queries slow

**Solution**:

- Vertical scaling: Increase database instance size (more CPU/RAM)
- Add read replicas for read queries
- Optimize slow queries (see Query Optimization)

### Replication lag increasing

**Symptoms**: Read replica falls behind primary by minutes/hours

**Cause**: High write volume on primary

**Solution**:

```bash
# Increase replica instance size
# AWS
aws rds modify-db-instance \
  --db-instance-identifier secrets-db-replica-1 \
  --db-instance-class db.r5.xlarge \
  --apply-immediately

# Or reduce write load (archive audit logs)
```

## See Also

- [Production Deployment Guide](docker-hardened.md) - Production database setup
- [Backup and Restore Guide](backup-restore.md) - Database backup strategies
- [Monitoring Guide](../observability/monitoring.md) - Database monitoring patterns
- [Scaling Guide](scaling-guide.md) - Application scaling (complements database scaling)
