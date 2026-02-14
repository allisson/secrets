# 📘 Glossary

> Last updated: 2026-02-14

Quick definitions for terms used across API and operations docs.

## Terms

- `Master Key`: Root key material used to protect KEKs; loaded from environment/KMS
- `KEK` (Key Encryption Key): Encrypts/decrypts DEKs; rotated over time
- `DEK` (Data Encryption Key): Encrypts payload data (secret values or transit key material)
- `Transit Key`: Named, versioned key used by transit encrypt/decrypt endpoints
- `Versioned ciphertext`: Transit ciphertext format `<version>:<base64-ciphertext>`
- `Capability`: Authorization permission (`read`, `write`, `delete`, `encrypt`, `decrypt`, `rotate`)
- `Soft delete`: Record marked deleted without immediate physical removal
- `Request ID`: Per-request UUID used for traceability and audit correlation

## See also

- [Architecture](architecture.md)
- [Security model](security-model.md)
- [Transit API](../api/transit.md)
- [Secrets API](../api/secrets.md)
