# 🔒 Security Model

> Last updated: 2026-02-28

Secrets is designed for practical defense-in-depth around secret storage and cryptographic operations.

## 🛡️ Core security properties

- 🔐 AEAD encryption (`aes-gcm` or `chacha20-poly1305`)
- 🔑 Envelope encryption hierarchy for key separation
- 🧹 Sensitive key material zeroing in critical paths
- 🎫 Time-limited bearer tokens with expiration controls
- 📜 Audit logging for authorization outcomes and request tracing

## 🎯 Threat-oriented view

- 💾 **Database compromise**: ciphertext remains protected without active master key material
- 🔑 **KEK compromise**: rotate KEK and constrain impact to wrapped DEKs
- 🎯 **DEK compromise**: impact scoped to specific data/version boundaries
- 🧪 **Credential abuse**: identify with audit log patterns (`allowed=false`, unusual source IPs)

## 🎫 Tokenization security considerations

- Metadata is not encrypted: do not place full PAN, credentials, or regulated payloads in token metadata.
- Deterministic tokenization leaks equality patterns for identical plaintext under the same active key.
- TTL expiration and revocation both invalidate token usage, but neither should replace endpoint authorization.
- Detokenization is plaintext exposure: isolate clients with `decrypt` capability and avoid shared broad policies.
- Expired tokens should be cleaned on cadence (`clean-expired-tokens`) to reduce stale sensitive mappings.

## 📜 Audit log integrity model

- Audit entries are append-only at API level
- There are no API endpoints to mutate or delete audit records
- Entries carry `request_id` for end-to-end request correlation

## ✅ Production recommendations

- Use HTTPS/TLS everywhere (run behind reverse proxy with TLS termination)
- Store master keys in KMS/HSM/secure secret manager (never in source control)
- Apply least-privilege policies per client and path
- Rotate KEKs and client credentials regularly
- Alert on repeated denied authorization attempts
- Separate `encrypt` and `decrypt` clients for tokenization and transit when possible
- Prefer non-deterministic tokenization unless deterministic matching is an explicit requirement
- Enable rate limiting to protect against abuse and denial-of-service attacks
- Use short token expiration times appropriate for your threat model (default: 4 hours)
- Enable database SSL/TLS in production (`sslmode=require` or `sslmode=verify-full`)
- Restrict network access to `/metrics` endpoint (port 8081)
- Forward audit logs to SIEM/log aggregation for long-term retention
- Disable CORS unless browser-based access is explicitly required

For comprehensive production security guidance, see [Security Hardening Guide](../operations/deployment/docker-hardened.md).

## ⚠️ Known limitations

- Runtime key hot-reload is not supported
- Master key and KEK context are loaded at process startup
- After rotating master keys or KEKs, API servers must be restarted

## 🚨 Incident response playbook (quick)

1. Revoke or deactivate compromised clients
2. Rotate KEK (and master key if needed)
3. Re-issue clients/tokens and validate policy scope
4. Review audit logs for lateral movement indicators

## See also

- [Security hardening guide](../operations/deployment/docker-hardened.md)
- [Production deployment](../operations/deployment/docker-hardened.md)
- [Architecture](architecture.md)
- [Authentication API](../api/auth/authentication.md)
- [Policies cookbook](../api/auth/policies.md)
- [Capability matrix](../api/fundamentals.md#capability-matrix)
- [Tokenization API](../api/data/tokenization.md)
- [Key management operations](../operations/kms/key-management.md)
