# Implementation Plan: Tokenization Key Retrieval by Name

This plan outlines the steps to add a new endpoint `GET /v1/tokenization/keys/:name` to retrieve a single tokenization key by its name.

## Phase 1: Domain and Use Case Layer
Add the `GetByName` functionality to the use case layer.

- [x] Task: Add `GetByName` to `TokenizationKeyUseCase` interface. 2fe4a7a
- [~] Task: Implement `GetByName` in `tokenizationKeyUseCase` struct.
    - File: `internal/tokenization/usecase/tokenization_key_usecase.go`
    - TDD: Write failing unit tests in `internal/tokenization/usecase/tokenization_key_usecase_test.go` first.
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Domain and Use Case Layer' (Protocol in workflow.md)

## Phase 2: HTTP Layer
Expose the new functionality through a REST endpoint.

- [ ] Task: Add `GetByNameHandler` to `TokenizationKeyHandler`.
    - File: `internal/tokenization/http/tokenization_key_handler.go`
    - TDD: Write failing unit tests in `internal/tokenization/http/tokenization_key_handler_test.go` first.
- [ ] Task: Register the new route `GET /v1/tokenization/keys/:name`.
    - File: `internal/app/di_tokenization.go`
- [ ] Task: Conductor - User Manual Verification 'Phase 2: HTTP Layer' (Protocol in workflow.md)

## Phase 3: Integration and Documentation
Ensure end-to-end functionality and update documentation.

- [ ] Task: Update integration tests in `test/integration/tokenization_flow_test.go`.
- [ ] Task: Update project documentation `docs/engines/tokenization.md`.
- [ ] Task: Update OpenAPI specification `docs/openapi.yaml`.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Integration and Documentation' (Protocol in workflow.md)
