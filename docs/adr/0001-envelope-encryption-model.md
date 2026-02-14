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

## See also

- [Architecture](../concepts/architecture.md)
- [Security model](../concepts/security-model.md)
- [Key management operations](../operations/key-management.md)
