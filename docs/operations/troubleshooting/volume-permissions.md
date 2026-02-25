# 🔐 Volume Permission Troubleshooting (v0.10.0+)

> **Document version**: v0.12.0  
> Last updated: 2026-02-21  
> **Audience**: DevOps engineers, SRE teams, container platform operators

## Table of Contents

- [Problem Statement](#problem-statement)

- [Symptoms](#symptoms)

- [Understanding the Issue](#understanding-the-issue)

- [Solutions](#solutions)

- [Verification Checklist](#verification-checklist)

- [Security Comparison](#security-comparison)

- [Rollback to v0.9.0 (Temporary Workaround)](#rollback-to-v090-temporary-workaround)

- [Frequently Asked Questions](#frequently-asked-questions)

- [See Also](#see-also)

- [Need Help?](#need-help)

This guide addresses volume permission errors introduced in v0.10.0 when the Docker container switched to running as a non-root user (UID 65532).

## Problem Statement

Starting in **v0.10.0**, the Docker container runs as a non-root user (`nonroot:nonroot`, UID/GID 65532) for enhanced security. This causes permission errors when mounting host directories as volumes because the non-root user cannot write to directories owned by other users.

## Symptoms

You may encounter these errors after upgrading to v0.10.0:

**Container startup failure**:

```text
Error: failed to start server: open /data/config.yaml: permission denied
```

**Runtime permission errors**:

```text
EACCES: permission denied, open '/data/secrets.db'
Error: operation not permitted
```

**Docker logs showing**:

```text

```text
$ docker logs secrets-api
panic: runtime error: permission denied writing to /data
```

**Container errors**:

```text
$ docker logs secrets-api
Error: failed to write to /data: permission denied

```

## Understanding the Issue

### What Changed in v0.10.0

| Aspect | v0.9.0 and earlier | v0.10.0+ |
|--------|-------------------|----------|
| **Base image** | `scratch` | `gcr.io/distroless/static-debian13` |
| **User** | `root` (UID 0) | `nonroot` (UID 65532) |
| **File permissions** | Can write anywhere | Can only write to files/dirs owned by UID 65532 |

### Why This Matters

When you mount a host directory into a container:

```bash
docker run -v /host/path:/container/path allisson/secrets:v0.12.0

```

The container process (running as UID 65532) tries to access `/container/path`, but the host directory `/host/path` is owned by your user (typically UID 1000) or `root` (UID 0). The non-root container user cannot read or write to these files.

### Security Context

**Why we made this change**:

- ✅ Follows security best practices (principle of least privilege)

- ✅ Reduces attack surface (compromised process can't write to system paths)

- ✅ Meets compliance requirements (PCI-DSS, SOC 2, etc.)

- ✅ Aligns with container security standards

## Solutions

Choose the solution that best fits your deployment environment and security requirements.

### Solution 1: Change Host Directory Ownership (Docker/Podman)

**Best for**: Local development, single-host deployments

**Security level**: ⚠️ Medium (exposes host directory to specific UID)

**Steps**:

```bash
# 1. Find your mounted volume directory
ls -la /path/to/host/data

# Example output:
# drwxr-xr-x  2 root root 4096 Feb 21 10:00 /path/to/host/data

# 2. Change ownership to UID 65532 (nonroot user)
sudo chown -R 65532:65532 /path/to/host/data

# 3. Verify permissions
ls -la /path/to/host/data
# drwxr-xr-x  2 65532 65532 4096 Feb 21 10:00 /path/to/host/data

# 4. Start container
docker run -d --name secrets-api \
  -v /path/to/host/data:/data \
  --env-file .env \
  -p 8080:8080 \
  allisson/secrets:v0.12.0 server

```

**Verification**:

```bash
# Check container logs (should start successfully)
docker logs secrets-api

# Test write permissions inside container
docker exec secrets-api touch /data/test.txt
docker exec secrets-api ls -la /data/test.txt
# -rw-r--r-- 1 nonroot nonroot 0 Feb 21 10:05 /data/test.txt

```

**Pros**:

- ✅ Simple and straightforward

- ✅ Works for local development

- ✅ No changes to docker-compose.yml or container configuration

**Cons**:

- ⚠️ Requires sudo/root access on host

- ⚠️ Host files owned by non-standard UID (may cause confusion)

- ⚠️ Not suitable for shared storage or NFS mounts

---

### Solution 2: Use Named Volumes (Docker/Docker Compose)

**Best for**: Production Docker Compose deployments, persistent data

**Security level**: ✅ High (Docker manages permissions automatically)

**Docker CLI**:

```bash
# 1. Create named volume
docker volume create secrets-data

# 2. Run container with named volume
docker run -d --name secrets-api \
  -v secrets-data:/data \
  --env-file .env \
  -p 8080:8080 \
  allisson/secrets:v0.12.0 server

```

**Docker Compose**:

```yaml
version: '3.8'

services:
  secrets-api:
    image: allisson/secrets:v0.12.0
    env_file: .env
    ports:
      - "8080:8080"

    volumes:
      # Named volume (Docker automatically sets correct permissions)
      - secrets-data:/data

    restart: unless-stopped

  # Optional: Healthcheck sidecar (distroless has no curl/wget)
  healthcheck:
    image: curlimages/curl:latest
    command: >
      sh -c 'while true; do
        curl -f http://secrets-api:8080/health || exit 1;
        sleep 30;
      done'
    depends_on:
      - secrets-api

    restart: unless-stopped

volumes:
  # Define named volume
  secrets-data:
    driver: local

```

**Verification**:

```bash
# Start services
docker-compose up -d

# Check volume
docker volume ls
# DRIVER    VOLUME NAME
# local     myapp_secrets-data

# Inspect volume permissions
docker volume inspect myapp_secrets-data

# Verify container can write
docker-compose exec secrets-api touch /data/test.txt
docker-compose exec secrets-api ls -la /data/test.txt

```

**Pros**:

- ✅ Docker handles permissions automatically

- ✅ No manual chown required

- ✅ Portable across environments

- ✅ Easy backup/restore (docker volume commands)

**Cons**:

- ⚠️ Data not directly accessible from host filesystem

- ⚠️ Requires docker volume commands for backup/inspection

**Accessing volume data from host**:

```bash
# Find volume mountpoint
docker volume inspect myapp_secrets-data | grep Mountpoint

# Copy data out
docker run --rm -v myapp_secrets-data:/data -v $(pwd):/backup busybox tar czf /backup/backup.tar.gz /data

# Copy data in
docker run --rm -v myapp_secrets-data:/data -v $(pwd):/backup busybox tar xzf /backup/backup.tar.gz -C /

```

---

### Solution 3: Run Container as Root (NOT RECOMMENDED)

**Best for**: Emergency debugging, temporary workarounds

**Security level**: ❌ Low (defeats the purpose of v0.10.0 security improvements)

**⚠️ WARNING**: This solution bypasses the security improvements in v0.10.0. Use only for temporary debugging.

**Docker**:

```bash
docker run -d --name secrets-api \
  --user root \
  -v /host/path:/data \
  --env-file .env \
  -p 8080:8080 \
  allisson/secrets:v0.12.0 server

```

**Why this is problematic**:

- ❌ Violates security best practices

- ❌ Increases attack surface (compromised process runs as root)

- ❌ May fail security audits

**When to use**:

- ✅ Emergency debugging to isolate permission issues

- ✅ Temporary workaround while implementing proper solution

- ✅ Local development (never production)

**After debugging, migrate to Solution 1, 2, or 3.**

---

## Verification Checklist

After implementing any solution, verify the fix:

### Docker/Docker Compose

```bash
# 1. Container starts successfully
docker ps | grep secrets-api
# Should show container in "Up" status

# 2. No permission errors in logs
docker logs secrets-api | grep -i "permission denied"
# Should return no results

# 3. Can write to volume
docker exec secrets-api touch /data/test.txt
echo $?
# Should return 0 (success)

# 4. Health check passes
curl http://localhost:8080/health
# Should return 200 OK

# 5. Application functional
curl -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{"client_id": "xxx", "client_secret": "yyy"}'
# Should return token (not permission error)

```

---

## Security Comparison

| Solution | Security Level | Best For |
|----------|---------------|----------|
| **Named volumes** | ✅ High | Docker Compose production |
| **chown host dir** | ⚠️ Medium | Local development |
| **Run as root** | ❌ Low | Emergency debugging only |

---

## Rollback to v0.9.0 (Temporary Workaround)

If you cannot immediately fix permissions, you can temporarily rollback to v0.9.0 (which runs as root):

**Docker**:

```bash
docker pull allisson/secrets:v0.9.0
docker run -d --name secrets-api \
  -v /host/path:/data \
  --env-file .env \
  -p 8080:8080 \
  allisson/secrets:v0.9.0 server

```

**Important**: v0.9.0 is a temporary workaround. Plan to implement proper permissions (Solution 1-3) and return to v0.10.0+ for security improvements.

---

## Frequently Asked Questions

### Q: Why did you change the default user in v0.10.0?

**A**: Running containers as root violates security best practices and increases attack surface. If an attacker exploits a vulnerability in the application, running as non-root limits the damage they can do. This aligns with:

- CIS Docker Benchmarks

- PCI-DSS requirements

- SOC 2 compliance standards

### Q: Can I change the UID from 65532 to something else?

**A**: The distroless base image uses 65532 (nonroot user) by default. Changing this requires building a custom image. We recommend using the default UID and fixing host permissions instead (Solution 1-3).

### Q: Why not use UID 1000 (common user UID)?

**A**: UID 65532 is specifically chosen to:

- Avoid conflicts with real users (typically UIDs 1000-60000)

- Signal "service account" (UIDs 60000+ conventionally for system services)

- Match distroless defaults (consistency across distroless images)

### Q: Will this affect my existing data?

**A**: No, but you need to ensure the container can access it:

- **Named volumes**: Docker handles migration automatically

- **Host directories**: You must `chown` the directory to UID 65532

---

## See Also

- [v0.10.0 Release Notes](../../releases/RELEASES.md#0100---2026-02-21)

- [Container Security Guide](../security/container-security.md)

- [Docker Quick Start](../../getting-started/docker.md)

- [Production Deployment Guide](../deployment/production.md)

- [Migration Guide](../../releases/RELEASES.md#migration-guide)

---

## Need Help?

If you're still experiencing permission errors after trying these solutions:

1. **Check logs**: `docker logs secrets-api` or `docker compose logs secrets`
2. **Verify UID**: `docker exec secrets-api id` (should show `uid=65532(nonroot)`)
3. **Check volume permissions**: `docker exec secrets-api ls -la /data`
4. **Open GitHub issue**: [github.com/allisson/secrets/issues](https://github.com/allisson/secrets/issues) with:
   - v0.10.0 version confirmation

   - Deployment platform (Docker/Docker Compose)

   - Full error message from logs

   - Output of verification commands above
