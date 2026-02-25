# Security Policy

## Supported Versions

We release patches for security vulnerabilities in the following versions:

| Version | Supported          |
| ------- | ------------------ |
| 0.10.x  | :white_check_mark: |
| 0.9.x   | :white_check_mark: |
| 0.8.x   | :x:                |
| < 0.8.0 | :x:                |

**Recommendation**: Always use the latest released version for the most up-to-date security patches.

## Reporting a Vulnerability

We take security vulnerabilities seriously. If you discover a security issue, please follow these steps:

### 1. Do NOT Create a Public Issue

Please **do not** create a public GitHub issue for security vulnerabilities. Public disclosure before a fix is available puts all users at risk.

### 2. Report Privately

Send your vulnerability report via email to:

**allisson@gmail.com**

Include the following information:

- **Description**: Clear description of the vulnerability
- **Impact**: What an attacker could do with this vulnerability
- **Affected versions**: Which versions are affected (if known)
- **Reproduction steps**: Step-by-step instructions to reproduce the issue
- **Proof of concept**: Code or commands demonstrating the vulnerability (if applicable)
- **Suggested fix**: Your recommended remediation (if you have one)

### 3. Response Timeline

You can expect:

- **Initial response**: Within 48 hours of your report
- **Triage and validation**: Within 5 business days
- **Status update**: Every 7 days until resolution
- **Fix timeline**: Depends on severity (see below)

### 4. Severity Levels and Response Times

| Severity | Response Time | Example |
|----------|---------------|---------|
| **Critical** | 24-48 hours | Remote code execution, authentication bypass |
| **High** | 5-7 days | SQL injection, privilege escalation |
| **Medium** | 14-30 days | Information disclosure, denial of service |
| **Low** | 30-60 days | Minor information leaks, non-security bugs |

### 5. Coordinated Disclosure

We follow **responsible disclosure** practices:

1. **Private fix**: We develop and test a fix privately
2. **Security advisory**: We prepare a security advisory (GitHub Security Advisories)
3. **Release**: We release a patched version
4. **Public disclosure**: We publish the advisory 24 hours after the release
5. **Credit**: We credit the reporter in the advisory (if desired)

If you need more time before public disclosure, please let us know when you submit your report.

## Security Best Practices

When deploying Secrets in production, follow these security recommendations:

### Container Security

- ✅ **Run as non-root**: Use the default UID 65532 (v0.10.0+)
- ✅ **Read-only filesystem**: Use `--read-only` flag when running containers
- ✅ **Drop capabilities**: Use `--cap-drop=ALL` to minimize attack surface
- ✅ **Security scanning**: Scan images with Trivy, Grype, or Docker Scout
- ✅ **Digest pinning**: Use SHA256 digest tags for immutable deployments

See [Container Security Guide](docs/operations/deployment/docker-hardened.md) for complete details.

### Secrets Management

- ✅ **KMS providers**: Use AWS KMS, Google Cloud KMS, or Azure Key Vault (not plaintext)
- ✅ **Key rotation**: Rotate master keys and KEKs regularly (quarterly recommended)
- ✅ **Environment variables**: Never commit `.env` files to version control
- ✅ **Transport security**: Always use TLS/HTTPS in production
- ✅ **Database encryption**: Enable database encryption at rest

See [Security Hardening Guide](docs/operations/deployment/docker-hardened.md) for complete details.

### API Authentication

- ✅ **Client secrets**: Store client secrets securely (never in code)
- ✅ **Token expiration**: Use short-lived tokens (15-60 minutes)
- ✅ **Least privilege**: Grant minimal capabilities required
- ✅ **Rate limiting**: Enable rate limiting to prevent brute force attacks
- ✅ **Audit logs**: Monitor audit logs for suspicious activity

See [Authentication API](docs/api/auth/authentication.md) and [Policy Cookbook](docs/api/auth/policies.md).

### Database Security

- ✅ **Connection encryption**: Use `sslmode=require` (PostgreSQL) or TLS (MySQL)
- ✅ **Least privilege**: Use dedicated database user with minimal permissions
- ✅ **Network isolation**: Use private networks or VPC peering
- ✅ **Backup encryption**: Encrypt database backups at rest
- ✅ **Parameter validation**: All queries use parameterized statements (SQL injection protection)

### Monitoring and Incident Response

- ✅ **Audit log monitoring**: Alert on failed authentication attempts
- ✅ **Security scanning**: Scan container images regularly (daily recommended)
- ✅ **Vulnerability alerts**: Subscribe to GitHub Security Advisories
- ✅ **Incident response plan**: Have a documented incident response process

See [Incident Response Guide](docs/operations/observability/incident-response.md).

## Known Security Considerations

### 1. Audit Log Signing (v0.9.0+)

Audit logs are cryptographically signed with HMAC-SHA256 for tamper detection. However:

- Signature verification requires the original KEK
- If a KEK is deleted, signatures cannot be verified
- Keep KEKs archived for compliance requirements

See [ADR 0011: HMAC-SHA256 Audit Log Signing](docs/adr/0011-hmac-sha256-audit-log-signing.md).

### 2. Envelope Encryption

Secrets use envelope encryption (Master Key → KEK → DEK → Secret Data):

- Compromise of a master key affects all KEKs and secrets
- Store master keys in a hardware security module (HSM) or KMS
- Rotate master keys regularly and re-wrap KEKs

See [Envelope Encryption Model](docs/adr/0001-envelope-encryption-model.md).

### 3. Client Secret Hashing

Client secrets are hashed with Argon2id (PHC winner):

- Hashing parameters: Memory=64MB, Iterations=3, Parallelism=2
- Cannot recover plaintext secrets after creation
- Store generated secrets securely after creation

See [ADR 0010: Argon2id for Client Secret Hashing](docs/adr/0010-argon2id-for-client-secret-hashing.md).

### 4. Rate Limiting

Rate limiting is applied per-client and per-IP:

- Token endpoint: Configurable per-IP rate limit (default: 10 req/sec)
- API endpoints: Per-client rate limit (based on capabilities)
- Configure limits based on your threat model

See [ADR 0006: Dual-Scope Rate Limiting Strategy](docs/adr/0006-dual-scope-rate-limiting-strategy.md).

## Security Scanning

The official Docker image is regularly scanned for vulnerabilities:

```bash
# Scan with Trivy
trivy image allisson/secrets:latest

# Scan with Docker Scout
docker scout cves allisson/secrets:latest

# Scan with Grype
grype allisson/secrets:latest
```

See [Security Scanning Guide](docs/operations/security/scanning.md) for CI/CD integration.

## Security Advisories

Security advisories are published via:

- **GitHub Security Advisories**: https://github.com/allisson/secrets/security/advisories
- **Release notes**: [CHANGELOG.md](CHANGELOG.md) and [RELEASES.md](docs/releases/RELEASES.md)

Subscribe to GitHub notifications to receive alerts for new advisories.

## Acknowledgments

We appreciate the security research community's efforts to help keep Secrets secure. Security researchers who report valid vulnerabilities will be credited in our security advisories (unless they prefer to remain anonymous).

Thank you for helping keep Secrets and its users safe!

## Questions?

If you have questions about this security policy or need clarification on reporting procedures, please email **allisson@gmail.com**.
