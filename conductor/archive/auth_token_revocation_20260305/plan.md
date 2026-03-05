# Implementation Plan: Auth Token Revocation

## Phase 1: Data Persistence & Repository
- [x] Task: Update `TokenRepository` interface in `internal/auth/usecase/interface.go` with `RevokeByTokenID`, `RevokeByClientID`, and `PurgeExpiredAndRevoked`. bced163
- [x] Task: Implement new methods in `internal/auth/repository/postgresql/token_repository.go` with integration tests (tagged `//go:build integration`). 2d85877
- [x] Task: Implement new methods in `internal/auth/repository/mysql/token_repository.go` with integration tests (tagged `//go:build integration`). 4d0f84a
- [x] Task: Conductor - User Manual Verification 'Phase 1' (Protocol in workflow.md) 127e4fd

## Phase 2: Application Logic & Audit
- [x] Task: Update `ClientUseCase` and `TokenUseCase` in `internal/auth/usecase/` with audit logging and unit tests. bb3ca60
- [x] Task: Ensure `TokenService` and `SecretService` mocks are generated and utilized in tests. 9d8839b
- [x] Task: Conductor - User Manual Verification 'Phase 2' (Protocol in workflow.md) f4d40c7

## Phase 3: API & Authentication Middleware
- [x] Task: Implement `DELETE /v1/token` handler in `internal/auth/http/token_handler.go` with unit tests in `internal/auth/http/token_handler_test.go`. 6c1c9cc
- [x] Task: Implement `DELETE /v1/clients/:id/tokens` handler in `internal/auth/http/client_handler.go` with unit tests in `internal/auth/http/client_handler_test.go`. 6c1c9cc
- [x] Task: Update `AuthenticationMiddleware` in `internal/http/middleware.go` (or relevant middleware) with unit tests in `internal/http/middleware_test.go`. f984c9c
- [x] Task: Conductor - User Manual Verification 'Phase 3' (Protocol in workflow.md) f984c9c

## Phase 4: Integration Testing & CLI
- [x] Task: Update integration tests in `test/integration/` to verify end-to-end token revocation. 7bd6462
- [x] Task: Add `purge-auth-tokens` command to the main CLI application. af6fdff
- [x] Task: Implement unit tests for the `purge-auth-tokens` CLI command in `cmd/app/commands_test.go`. af6fdff
- [x] Task: Conductor - User Manual Verification 'Phase 4' (Protocol in workflow.md) 0983107

## Phase 5: Documentation
- [x] Task: Update CLI documentation in `docs/cli-commands.md`. af6fdff
- [x] Task: Update `docs/openapi.yaml` with the new DELETE endpoints. af6fdff
- [x] Task: Update `docs/auth/policies.md` with policy examples. af6fdff
- [x] Task: Conductor - User Manual Verification 'Phase 5' (Protocol in workflow.md) 9f36345
