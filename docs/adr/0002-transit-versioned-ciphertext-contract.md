# ADR 0002: Transit Versioned Ciphertext Contract

> Status: accepted
> Date: 2026-02-14

## Context

Transit decryption must reliably select the correct transit key version and reject malformed ciphertext early.

## Decision

Adopt transit ciphertext contract:

`<version>:<base64-ciphertext>`

- decrypt requires version prefix and base64 payload
- malformed input returns validation errors (`422`)
- callers must pass ciphertext exactly as returned by encrypt

## Consequences

- deterministic key version selection for decrypt
- stronger input validation with predictable errors
- simpler client behavior by treating encrypt output as opaque

## See also

- [Transit API](../api/data/transit.md)
- [Response shapes](../api/observability/response-shapes.md)
- [Troubleshooting](../operations/troubleshooting/index.md)
- [ADR 0007: Path-Based API Versioning](0007-path-based-api-versioning.md) - API versioning context for ciphertext format stability
