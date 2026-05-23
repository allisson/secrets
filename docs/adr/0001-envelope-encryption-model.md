# ADR 0001: Envelope Encryption Model

> Status: accepted
> Date: 2026-02-14

## Context

The system must protect stored secret payloads while supporting key rotation without re-encrypting all historical data at once.

## Decision

Use envelope encryption hierarchy:

`Master Key -> KEK -> DEK -> Secret Data`

- master keys protect KEKs
- KEKs protect DEKs
- DEKs encrypt secret payloads

## Consequences

- key rotation can happen incrementally
- historical versions remain decryptable with prior key material
- clear separation between root trust, key-wrapping, and data encryption roles

## Module structure

The envelope-encryption model is implemented by a single module,
`internal/keyring`, introduced in [ADR 0013](0013-keyring-as-envelope-encryption-module.md).
Features (secrets, transit, tokenization) call `Keyring.Encrypt` /
`Keyring.Decrypt` (or the persistent-DEK pair `AllocateDek` /
`EncryptWith` / `DecryptWith`) and do not handle the KEK chain, DEK
lifecycle, or AEAD selection directly.

## See also

- [Architecture](../concepts/architecture.md)
- [Security model](../concepts/security-model.md)
- [Key management operations](../operations/kms/key-management.md)
- [ADR 0012: PostgreSQL-Only Database](0012-postgresql-only-database.md) - Database storage for encrypted key material
- [ADR 0013: Keyring as Envelope Encryption Module](0013-keyring-as-envelope-encryption-module.md) - Single-module implementation of this model
