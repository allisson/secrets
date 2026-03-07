# Implementation Plan: Transit Key Retrieval API

## Phase 1: Repository Layer
- [ ] Task: Define `GetTransitKey` in `internal/transit/domain/repository.go` and repository interface.
- [ ] Task: Implement `GetTransitKey` in `internal/transit/repository/postgresql/transit_key_repository.go`.
- [ ] Task: Implement `GetTransitKey` in `internal/transit/repository/mysql/transit_key_repository.go`.
- [ ] Task: Write integration tests for `GetTransitKey` in both PostgreSQL and MySQL repositories.
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Repository Layer' (Protocol in workflow.md)

## Phase 2: Use Case Layer
- [ ] Task: Define `GetTransitKey` method in `internal/transit/domain/usecase.go`.
- [ ] Task: Implement `GetTransitKey` in `internal/transit/usecase/transit_key_usecase.go`.
- [ ] Task: Write unit tests for `GetTransitKey` use case in `internal/transit/usecase/transit_key_usecase_test.go`.
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Use Case Layer' (Protocol in workflow.md)

## Phase 3: HTTP API Implementation
- [ ] Task: Create `GetTransitKeyHandler` in `internal/transit/http/transit_key_handler.go`.
- [ ] Task: Register the new route `GET /api/v1/transit/keys/:name` in `internal/transit/http/router.go`.
- [ ] Task: Write unit tests for `GetTransitKeyHandler` in `internal/transit/http/transit_key_handler_test.go`.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: HTTP API Implementation' (Protocol in workflow.md)

## Phase 4: Documentation
- [ ] Task: Update `docs/engines/transit.md` to document the new key retrieval capability.
- [ ] Task: Update `docs/openapi.yaml` to include the `GET /api/v1/transit/keys/:name` endpoint.
- [ ] Task: Conductor - User Manual Verification 'Phase 4: Documentation' (Protocol in workflow.md)
