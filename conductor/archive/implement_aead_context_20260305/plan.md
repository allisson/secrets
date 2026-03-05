# Implementation Plan: AEAD Context in Transit Engine

## Phase 1: Use Case Layer Update
- [x] Task: Update TransitKeyUseCase interface and implementation signatures
    - [x] Update `internal/transit/usecase/interface.go` to add `context []byte` to `Encrypt` and `Decrypt`
    - [x] Update `internal/transit/usecase/transit_key_usecase.go` signatures to match interface
    - [x] Update existing callers in tests to pass `nil` for context
- [x] Task: Implement AEAD context support in `TransitKeyUseCase` (TDD)
    - [x] Write failing unit tests in `internal/transit/usecase/transit_key_usecase_test.go` for encryption/decryption with context
    - [x] Implement logic in `internal/transit/usecase/transit_key_usecase.go` to pass context as `aad` to cipher
    - [x] Verify tests pass and coverage is >80%
- [x] Task: Conductor - User Manual Verification 'Phase 1: Use Case Layer' (Protocol in workflow.md)

## Phase 2: HTTP Layer Update
- [x] Task: Update `EncryptRequest` and `DecryptRequest` DTOs
    - [x] Update `internal/transit/http/dto/request.go` to add optional `context` (base64)
- [x] Task: Implement AEAD context support in `CryptoHandler` (TDD)
    - [x] Write failing unit tests in `internal/transit/http/crypto_handler_test.go` for API calls with context
    - [x] Update `internal/transit/http/crypto_handler.go` to decode base64 context and pass to use case
    - [x] Verify tests pass and coverage is >80%
- [x] Task: Conductor - User Manual Verification 'Phase 2: HTTP Layer' (Protocol in workflow.md)

## Phase 3: Integration and Documentation
- [x] Task: Verify end-to-end flow with integration tests
    - [x] Update `test/integration/transit_flow_test.go` to include AEAD context scenarios
    - [x] Run all transit integration tests and verify they pass
- [x] Task: Update Documentation
    - [x] Update OpenAPI spec in `docs/openapi.yaml`
    - [x] Update transit engine guide in `docs/engines/transit.md` with AEAD context examples
- [x] Task: Conductor - User Manual Verification 'Phase 3: Integration and Documentation' (Protocol in workflow.md)
