# 🚨 Disaster Recovery Runbook

> **Document version**: v0.13.0  
> Last updated: 2026-02-25  
> **Audience**: SRE teams, platform engineers, incident commanders
>
> **⚠️ UNTESTED PROCEDURES**: The procedures in this guide are reference examples and have not been tested in production. Always test in a non-production environment first and adapt to your infrastructure.

This runbook covers disaster recovery procedures for Secrets, including complete service restoration, data recovery, and failover scenarios.

## Table of Contents

- [Overview](#overview)
- [Disaster Scenarios](#disaster-scenarios)
- [Recovery Procedures](#recovery-procedures)
- [Validation Checklist](#validation-checklist)
- [Recovery Metrics](#recovery-metrics)
- [Post-Recovery Actions](#post-recovery-actions)
- [Troubleshooting](#troubleshooting)
- [See Also](#see-also)

## Overview

### What Qualifies as a Disaster

A disaster is any event that causes **complete service unavailability** or **unrecoverable data loss**:

- **Infrastructure failure**: Complete cloud region outage, datacenter failure, infrastructure destruction
- **Data loss**: Database corruption, accidental deletion, ransomware encryption
- **Security incident**: Master key compromise, unauthorized data access, credential leak
- **Human error**: Accidental `DROP DATABASE`, wrong production deployment, configuration deletion

### Recovery Objectives

| Metric | Target | Critical Path |
|--------|--------|---------------|
| **RTO** (Recovery Time Objective) | 60 minutes | Restore database + master key + deploy application |
| **RPO** (Recovery Point Objective) | 1 hour | Last successful backup (hourly backups) |
| **MTD** (Maximum Tolerable Downtime) | 4 hours | Business continuity limit |

### Prerequisites

**Before a disaster**:

- [ ] Hourly database backups to offsite storage (S3/GCS/Azure Blob)
- [ ] Master key backed up in KMS or encrypted vault
- [ ] Infrastructure-as-Code (IaC) stored in git
- [ ] DR runbook tested quarterly
- [ ] On-call team trained on recovery procedures
- [ ] Access credentials stored in secure vault

## Disaster Scenarios

### Scenario 1: Complete Database Loss

**Symptoms**:

- Database server unreachable
- `FATAL: database "secrets" does not exist`
- All data lost (corruption, deletion, etc.)

**Recovery procedure**: [Database Recovery](#database-recovery)

---

### Scenario 2: Cloud Region Outage

**Symptoms**:

- Entire cloud region (AWS us-east-1, GCP us-central1, etc.) unavailable
- Infrastructure inaccessible
- Database and application unreachable

**Recovery procedure**: [Regional Failover](#regional-failover)

---

### Scenario 3: Master Key Loss / Compromise

**Symptoms**:

- Master key deleted or inaccessible
- KMS key disabled or permissions revoked
- Master key compromised (security incident)

**Recovery procedure**: [Master Key Recovery](#master-key-recovery) or [Master Key Rotation (Compromise)](#master-key-rotation-compromise)

---

### Scenario 4: Complete Infrastructure Destruction

**Symptoms**:

- Infrastructure deleted
- All infrastructure destroyed
- Only backups remain (database + master key)

**Recovery procedure**: Follow [Database Recovery](#database-recovery), [Master Key Recovery](#master-key-recovery), and [Application Deployment](#application-deployment) procedures in sequence.

---

### Scenario 5: Ransomware / Data Corruption

**Symptoms**:

- Database tables encrypted or corrupted
- Application returns gibberish data
- Audit logs show unauthorized access

**Recovery procedure**: Follow [Database Recovery](#database-recovery) to restore from the last known good backup.

## Recovery Procedures

### Database Recovery

**Goal**: Restore database from most recent clean backup

**Steps**:

1. **Identify latest clean backup**:

   ```bash
   # List backups
   aws s3 ls s3://my-backups/secrets/ --recursive | grep dump | tail -10
   
   # Download latest backup
   aws s3 cp s3://my-backups/secrets/secrets-backup-20260221-120000.dump .
   ```

2. **Create new database** (if destroyed):

   ```bash
   # PostgreSQL
   createdb secrets
   
   # MySQL
   mysql -e "CREATE DATABASE secrets;"
   ```

3. **Restore backup**:

   ```bash
   # PostgreSQL
   pg_restore \
     --host=localhost \
     --username=secrets \
     --dbname=secrets \
     --clean \
     --if-exists \
     --verbose \
     secrets-backup-20260221-120000.dump
   
   # MySQL
   mysql --host=localhost --user=secrets --password secrets < secrets-backup.sql
   ```

4. **Verify restoration**:

   ```sql
   -- Check schema version
   SELECT version FROM schema_migrations ORDER BY version DESC LIMIT 1;
   
   -- Count records
   SELECT 
     (SELECT COUNT(*) FROM clients) as clients,
     (SELECT COUNT(*) FROM secrets) as secrets,
     (SELECT COUNT(*) FROM key_encryption_keys) as keks,
     (SELECT COUNT(*) FROM audit_logs) as audit_logs;
   ```

5. **Proceed to [Application Deployment](#application-deployment)**

**Expected RTO**: 15-30 minutes (depends on database size)

---

### Master Key Recovery

**Goal**: Restore master key from KMS or encrypted backup

**KMS-based master key**:

```bash
# Verify KMS key is accessible
# AWS
aws kms describe-key --key-id alias/secrets-master-key

# GCP
gcloud kms keys describe secrets-master-key \
  --location=us-east1 --keyring=secrets

# Set environment variable
export MASTER_KEY_PROVIDER=aws-kms
export MASTER_KEY_KMS_KEY_ID=arn:aws:kms:us-east-1:123456789012:key/abc-def
```

**Plaintext master key from backup**:

```bash
# Decrypt backup
gpg --decrypt master-key-backup.txt.gpg

# Set environment variable
export MASTER_KEY_PROVIDER=plaintext
export MASTER_KEY_PLAINTEXT=<base64-encoded-key>

# Verify length
echo $MASTER_KEY_PLAINTEXT | base64 -d | wc -c
# Should output: 32
```

**Expected RTO**: < 5 minutes

---

### Application Deployment

**Goal**: Deploy Secrets application with restored database and master key

**Docker Compose**:

```bash
# 1. Create/update .env file with database connection
cat > .env <<EOF
DB_DRIVER=postgres
DB_CONNECTION_STRING=postgres://user:pass@postgres:5432/secrets?sslmode=require
MASTER_KEY_PROVIDER=aws-kms
MASTER_KEY_KMS_KEY_ID=arn:aws:kms:...
EOF

# 2. Deploy application with docker-compose
docker-compose up -d

# 3. Verify health
curl http://localhost:8080/health
curl http://localhost:8080/ready
```

**Expected RTO**: 10-20 minutes

---

### Regional Failover

**Goal**: Failover to backup cloud region

**Prerequisites**:

- Multi-region database replication configured
- Application deployed in secondary region (standby)
- DNS/load balancer supports failover

**Steps**:

1. **Promote secondary database to primary**:

   ```bash
   # AWS RDS
   aws rds promote-read-replica --db-instance-identifier secrets-db-us-west-2
   
   # GCP Cloud SQL
   gcloud sql instances promote-replica secrets-db-us-west-2
   ```

2. **Update DNS to point to secondary region**:

   ```bash
   # Route 53 health check failover (automatic)
   # Or manual DNS update
   aws route53 change-resource-record-sets \
     --hosted-zone-id Z1234567890ABC \
     --change-batch file://failover-dns.json
   ```

3. **Scale up application in secondary region**:

   ```bash
   # Docker Compose: Scale up replicas
   docker-compose up -d --scale secrets=3
   
   # Or adjust docker-compose.yml and redeploy
   docker-compose up -d
   ```

**Expected RTO**: 30-60 minutes

**Expected RPO**: Time between corruption and last clean backup (e.g., 6 hours if corruption happened and last clean backup is 6 hours old)

---

### Master Key Rotation (Compromise)

**Goal**: Rotate master key after security compromise and re-encrypt all KEKs

**Steps**:

1. **Generate new master key**:

   ```bash
   # KMS: Create new key
   aws kms create-key --description "Secrets master key v2"
   aws kms create-alias --alias-name alias/secrets-master-key-v2 \
     --target-key-id <new-key-id>
   
   # Plaintext: Generate new key
   openssl rand -base64 32
   ```

2. **Run master key rotation**:

   ```bash
   ./bin/app rotate-master-key \
     --old-master-key-provider=aws-kms \
     --old-master-key-kms-key-id=arn:aws:kms:...:key/old-key \
     --new-master-key-provider=aws-kms \
     --new-master-key-kms-key-id=arn:aws:kms:...:key/new-key
   ```

3. **Update application configuration**:

   ```bash
   # Update .env file
   sed -i 's|MASTER_KEY_KMS_KEY_ID=.*|MASTER_KEY_KMS_KEY_ID=arn:aws:kms:...:key/new-key|' .env
   
   # Restart application
   docker-compose restart secrets
   ```

4. **Disable old master key**:

   ```bash
   aws kms disable-key --key-id <old-key-id>
   ```

5. **Verify all KEKs re-encrypted**:

   ```sql
   -- All KEKs should have updated_at timestamp > rotation time
   SELECT id, created_at, updated_at FROM key_encryption_keys;
   ```

**Expected RTO**: 30-60 minutes (depends on number of KEKs)

## Validation Checklist

After completing recovery, validate the following:

### Health Checks

- [ ] `GET /health` returns `200 OK`
- [ ] `GET /ready` returns `200 OK`
- [ ] Application logs show no errors

### Database Connectivity

- [ ] Database schema version matches expected version
- [ ] Sample queries return expected data
- [ ] Audit logs contain recent entries

### Secret Operations

- [ ] Create new secret succeeds
- [ ] Retrieve existing secret succeeds (data decrypts correctly)
- [ ] Update secret succeeds
- [ ] Delete secret succeeds

### Transit Encryption

- [ ] Create transit key succeeds
- [ ] Encrypt plaintext with transit key succeeds
- [ ] Decrypt ciphertext with transit key succeeds

### Authentication

- [ ] Get auth token with client credentials succeeds
- [ ] Token validates correctly on protected endpoints
- [ ] Token expiration works as expected

### Audit Logging

- [ ] New operations create audit logs
- [ ] Audit log signatures verify correctly (v0.9.0+)
- [ ] Audit logs export to external storage (if configured)

## Recovery Metrics

Track these metrics during recovery:

| Metric | Definition | How to Measure |
|--------|------------|----------------|
| **Detection Time** | Time from disaster to detection | Monitoring alert timestamp - incident timestamp |
| **Response Time** | Time from detection to recovery start | Recovery start timestamp - detection timestamp |
| **Recovery Time** | Time from recovery start to service restored | Service restored timestamp - recovery start timestamp |
| **RTO Actual** | Total downtime (detection to restoration) | Service restored timestamp - incident timestamp |
| **RPO Actual** | Data loss window | Last backup timestamp - incident timestamp |

**Example**:

```text
Incident timestamp:    2026-02-21 10:00:00 (database corruption detected)
Detection timestamp:   2026-02-21 10:05:00 (monitoring alert)
Recovery start:        2026-02-21 10:10:00 (team started runbook)
Service restored:      2026-02-21 10:45:00 (health checks pass)
Last backup:           2026-02-21 09:00:00 (hourly backup)

Detection Time: 5 minutes
Response Time: 5 minutes
Recovery Time: 35 minutes
RTO Actual: 45 minutes (within 60-minute target ✅)
RPO Actual: 1 hour (within 1-hour target ✅)
```

## Post-Recovery Actions

### Immediate (within 24 hours)

1. **Incident report**: Document what happened, root cause, timeline
2. **Customer communication**: Notify affected users (if applicable)
3. **Security review**: If security-related, review access logs and credentials
4. **Backup validation**: Verify backups are still working correctly

### Short-term (within 1 week)

1. **Post-mortem**: Hold blameless post-mortem with team
2. **Runbook update**: Update DR runbook with lessons learned
3. **Monitoring improvements**: Add alerts to detect similar issues earlier
4. **Testing**: Test recovery procedures in non-production environment

### Long-term (within 1 month)

1. **Infrastructure hardening**: Implement changes to prevent recurrence
2. **DR drill**: Schedule quarterly DR drill based on lessons learned
3. **Documentation**: Update architecture diagrams and runbooks
4. **Training**: Train team on updated procedures

## Troubleshooting

### Database restore fails with "relation already exists"

**Cause**: Target database not empty

**Solution**:

```bash
# Use --clean flag
pg_restore --clean --if-exists secrets-backup.dump
```

### Application fails to start after recovery

**Symptoms**:

```text
FATAL: could not decrypt KEK
panic: master key mismatch
```

**Cause**: Wrong master key configured

**Solution**:

```bash
# Verify master key matches backup
# Check KMS key ID or plaintext key hash
echo $MASTER_KEY_KMS_KEY_ID
```

### Health checks pass but secrets return gibberish

**Cause**: Database restored but master key is different

**Solution**: Restore must use the SAME master key as the backup. If master key is lost, data is unrecoverable.

### Backup restore is too slow (hours)

**Cause**: Large database (millions of audit logs)

**Solution**:

```bash
# Restore without audit logs (faster)
pg_restore --exclude-table=audit_logs secrets-backup.dump

# Restore audit logs separately (parallel)
pg_restore --table=audit_logs secrets-backup.dump
```

### Regional failover takes longer than expected

**Cause**: DNS propagation delay or database promotion delay

**Solution**:

- Use health-based DNS failover (Route 53, Cloud DNS)
- Keep read replicas in warm standby mode
- Test failover quarterly to identify bottlenecks

## See Also

- [Backup and Restore Guide](../deployment/backup-restore.md) - Detailed backup procedures
- [Production Deployment Guide](../deployment/docker-hardened.md) - Pre-production disaster recovery checklist
- [Security Hardening Guide](../deployment/docker-hardened.md) - Master key security best practices
- [Health Check Endpoints](../observability/health-checks.md) - Health validation patterns
- [Runbooks README](README.md) - All operational runbooks
