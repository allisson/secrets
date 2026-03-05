# Specification: Implement AEAD Context in Transit Engine

## Overview
Authenticated Encryption with Associated Data (AEAD) allows providing additional, non-encrypted data (context) that is cryptographically bound to the ciphertext. This prevents ciphertext substitution attacks where a valid ciphertext for one context is used in another.

## Requirements
- Update `TransitKeyUseCase.Encrypt` and `Decrypt` to accept an optional `context` parameter (as `[]byte`).
- Update `TransitKeyUseCase` implementation to pass this `context` as `aad` to the `AEAD` cipher.
- Update `transit.http` handlers to accept an optional `context` field (base64-encoded) in the request body.
- Maintain backward compatibility: if `context` is not provided, it should behave as before (equivalent to empty/nil `aad`).

## Affected Components
- `internal/transit/usecase/interface.go`: Update `TransitKeyUseCase` interface.
- `internal/transit/usecase/transit_key_usecase.go`: Update `Encrypt` and `Decrypt` methods.
- `internal/transit/http/dto/request.go`: Add `Context` field to `EncryptRequest` and `DecryptRequest`.
- `internal/transit/http/crypto_handler.go`: Pass `context` from request to use case.
- `internal/transit/usecase/transit_key_usecase_test.go`: Add tests for AEAD context.
- `internal/transit/http/crypto_handler_test.go`: Add tests for AEAD context in API.

## Design Decisions
- **Context Encoding:** In the API, the `context` will be base64-encoded, consistent with `plaintext`.
- **Naming:** The API field will be named `context` to be more user-friendly, although it maps to `aad` in cryptographic terms.
