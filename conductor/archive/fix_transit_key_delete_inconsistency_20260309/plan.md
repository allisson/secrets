# Implementation Plan: Fix Transit Key Delete Endpoint Inconsistency

## Phase 1: Core Domain and Repository Updates
- [x] Task: Update `TransitKeyRepository` interface in `internal/transit/domain/repository.go` to change `Delete` signature from `transitKeyID uuid.UUID` to `name string`.
- [x] Task: Update `TransitKeyUseCase` interface in `internal/transit/usecase/interface.go` to change `Delete` signature from `transitKeyID uuid.UUID` to `name string`.
- [x] Task: Update PostgreSQL implementation in `internal/transit/repository/postgresql/postgresql_transit_key_repository.go` to soft-delete all versions by name.
- [x] Task: Update MySQL implementation in `internal/transit/repository/mysql/mysql_transit_key_repository.go` to soft-delete all versions by name.
- [x] Task: Update `TransitKeyUseCase` implementation in `internal/transit/usecase/transit_key_usecase.go` to use the new repository method.
- [x] Task: Update use case unit tests in `internal/transit/usecase/transit_key_usecase_test.go`.
- [x] Task: Update repository unit tests for both PostgreSQL and MySQL.
- [x] Task: Conductor - User Manual Verification 'Phase 1: Core Domain and Repository Updates' (Protocol in workflow.md)

## Phase 2: HTTP Handler and Routing Updates
- [x] Task: Update `TransitKeyHandler.DeleteHandler` in `internal/transit/http/transit_key_handler.go` to extract `name` from URL parameters instead of `id` (UUID).
- [x] Task: Update route registration in `internal/http/server.go` to change `DELETE /v1/transit/keys/:id` to `DELETE /v1/transit/keys/:name`.
- [x] Task: Update handler unit tests in `internal/transit/http/transit_key_handler_test.go`.
- [x] Task: Conductor - User Manual Verification 'Phase 2: HTTP Handler and Routing Updates' (Protocol in workflow.md)

## Phase 3: Integration Tests and Documentation
- [x] Task: Update integration tests in `test/integration/transit_flow_test.go` to use the transit key name for the DELETE request.
- [x] Task: Update `docs/openapi.yaml` to change `{id}` to `{name}` for the `DELETE /v1/transit/keys/{id}` endpoint and update its description.
- [x] Task: Update `docs/engines/transit.md` to ensure the Delete Transit Key section uses `:name`.
- [x] Task: Update `docs/concepts/api-fundamentals.md` and `docs/auth/policies.md` to change `:id` to `:name` for the transit delete endpoint.
- [x] Task: Conductor - User Manual Verification 'Phase 3: Integration Tests and Documentation' (Protocol in workflow.md)
