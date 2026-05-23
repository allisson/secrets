# Security Model

Preserve these project security choices:

- AES-GCM or ChaCha20-Poly1305 encryption.
- Argon2id client secret hashing.
- HMAC-SHA256 audit log signing.
- KMS support through `gocloud.dev`.

Record major security or architecture changes as ADRs in `docs/adr`.
