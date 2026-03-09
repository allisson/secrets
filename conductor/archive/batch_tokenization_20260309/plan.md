# Implementation Plan: Batch Tokenize/Detokenize Endpoints

This plan outlines the steps to implement batch tokenization and detokenization endpoints in the Secrets manager.

## Phase 1: Domain and Repository Layer

- [x] Task: Define Batch Tokenization Interfaces [e8a8cee]
    - [ ] Add `CreateBatch` to `TokenRepository` interface in `internal/tokenization/domain/token.go`.
    - [ ] Add `GetBatchByTokens` to `TokenRepository` interface in `internal/tokenization/domain/token.go`.
- [x] Task: Implement Batch Repository Methods (PostgreSQL) [517777c]
    - [ ] Implement `CreateBatch` in `internal/tokenization/repository/postgresql/token_repository.go`.
    - [ ] Implement `GetBatchByTokens` in `internal/tokenization/repository/postgresql/token_repository.go`.
    - [ ] Write integration tests for these methods (tagged with `//go:build integration`).
- [x] Task: Implement Batch Repository Methods (MySQL) [cc03816]
    - [ ] Implement `CreateBatch` in `internal/tokenization/repository/mysql/token_repository.go`.
    - [ ] Implement `GetBatchByTokens` in `internal/tokenization/repository/mysql/token_repository.go`.
    - [ ] Write integration tests for these methods (tagged with `//go:build integration`).
- [x] Task: Implement Batch Usecase Logic [191cb29]
    - [ ] Add `TokenizeBatch` to `TokenizationUsecase` in `internal/tokenization/usecase/tokenization_usecase.go`.
    - [ ] Add `DetokenizeBatch` to `TokenizationUsecase` in `internal/tokenization/usecase/tokenization_usecase.go`.
    - [ ] Ensure both methods use `TxManager` for atomicity.
    - [ ] Implement the loop over existing single-item logic.
    - [ ] Write unit tests for the new usecase methods.
- [x] Task: Conductor - User Manual Verification 'Phase 1: Domain and Repository Layer' (Protocol in workflow.md)

## Phase 2: HTTP Layer

- [x] Task: Define Request/Response DTOs [cc85bfe]
    - [ ] Create `TokenizeBatchRequest` and `TokenizeBatchResponse` in `internal/tokenization/http/dto.go` (or equivalent).
    - [ ] Create `DetokenizeBatchRequest` and `DetokenizeBatchResponse` in `internal/tokenization/http/dto.go`.
    - [ ] Implement validation rules (e.g., max 100 items).
- [x] Task: Implement HTTP Handlers [ee3290b]
    - [ ] Implement `TokenizeBatch` handler in `internal/tokenization/http/tokenization_handler.go`.
    - [ ] Implement `DetokenizeBatch` handler in `internal/tokenization/http/tokenization_handler.go`.
    - [ ] Write unit tests for the new handlers in `internal/tokenization/http/tokenization_handler_test.go`.
- [x] Task: Register Routes [ee3290b]
    - [ ] Add the new batch routes to the router in `internal/tokenization/http/tokenization_handler.go` (or `internal/app/di_tokenization.go`).
- [x] Task: Conductor - User Manual Verification 'Phase 2: HTTP Layer' (Protocol in workflow.md)

## Phase 3: Documentation and Integration Testing

- [x] Task: Update Integration Flow Tests [efa3c2c]
    - [ ] Add batch operation test cases to `test/integration/tokenization_flow_test.go`.
    - [ ] Verify atomicity by intentionally failing one item in a batch.
- [x] Task: Update OpenAPI Specification [fa50f71]
    - [ ] Add the new batch endpoints to `docs/openapi.yaml`.
- [x] Task: Update Engine Documentation [9faab2e]
    - [ ] Update `docs/engines/tokenization.md` with examples of batch requests and responses.
- [x] Task: Conductor - User Manual Verification 'Phase 3: Documentation and Integration Testing' (Protocol in workflow.md)
