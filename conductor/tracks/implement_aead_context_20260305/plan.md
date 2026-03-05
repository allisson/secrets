# Implementation Plan: AEAD Context in Transit Engine

## Phase 1: Use Case Layer Update
- [ ] Task: Update `TransitKeyUseCase` interface and implementation signatures
    - [ ] Update `internal/transit/usecase/interface.go` to add `context []byte` to `Encrypt` and `Decrypt`
    - [ ] Update `internal/transit/usecase/transit_key_usecase.go` signatures to match interface
    - [ ] Update existing callers in tests to pass `nil` for context
- [ ] Task: Implement AEAD context support in `TransitKeyUseCase` (TDD)
    - [ ] Write failing unit tests in `internal/transit/usecase/transit_key_usecase_test.go` for encryption/decryption with context
    - [ ] Implement logic in `internal/transit/usecase/transit_key_usecase.go` to pass context as `aad` to cipher
    - [ ] Verify tests pass and coverage is >80%
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Use Case Layer' (Protocol in workflow.md)

## Phase 2: HTTP Layer Update
- [ ] Task: Update `EncryptRequest` and `DecryptRequest` DTOs
    - [ ] Update `internal/transit/http/dto/request.go` to add optional `context` (base64)
- [ ] Task: Implement AEAD context support in `CryptoHandler` (TDD)
    - [ ] Write failing unit tests in `internal/transit/http/crypto_handler_test.go` for API calls with context
    - [ ] Update `internal/transit/http/crypto_handler.go` to decode base64 context and pass to use case
    - [ ] Verify tests pass and coverage is >80%
- [ ] Task: Conductor - User Manual Verification 'Phase 2: HTTP Layer' (Protocol in workflow.md)

## Phase 3: Integration and Documentation
- [ ] Task: Verify end-to-end flow with integration tests
    - [ ] Update `test/integration/transit_flow_test.go` to include AEAD context scenarios
    - [ ] Run all transit integration tests and verify they pass
- [ ] Task: Update Documentation
    - [ ] Update OpenAPI spec in `docs/openapi.yaml`
    - [ ] Update transit engine guide in `docs/engines/transit.md` with AEAD context examples
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Integration and Documentation' (Protocol in workflow.md)
