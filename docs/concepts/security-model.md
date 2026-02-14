# 🔒 Security Model

> Last updated: 2026-02-14

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

## 📜 Audit log integrity model

- Audit entries are append-only at API level
- There are no API endpoints to mutate or delete audit records
- Entries carry `request_id` for end-to-end request correlation

## ✅ Production recommendations

- Use HTTPS/TLS everywhere
- Store master keys in KMS/HSM/secure secret manager
- Apply least-privilege policies per client and path
- Rotate KEKs and client credentials regularly
- Alert on repeated denied authorization attempts

## ⚠️ Known limitations

- Runtime key hot-reload is not supported
- Master key and KEK context are loaded at process startup
- After rotating master keys or KEKs, API servers must be restarted

## 🚨 Incident response playbook (quick)

1. Revoke or deactivate compromised clients
2. Rotate KEK (and master key if needed)
3. Re-issue clients/tokens and validate policy scope
4. Review audit logs for lateral movement indicators
