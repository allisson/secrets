# Implementation Plan: Add Audit Log Filtering by Client

## Phase 1: Repository Layer Update [checkpoint: a640ed9]
- [x] Task: Update `AuditLogRepository` interface in `internal/auth/usecase/interface.go` to include `clientID *uuid.UUID` in `ListCursor`. 97bee6d
- [x] Task: Update `PostgreSQLAuditLogRepository` in `internal/auth/repository/postgresql/postgresql_audit_log_repository.go`. 8501e96
    - [x] Update `ListCursor` to support `client_id` filtering.
    - [x] Update/Add tests in `internal/auth/repository/postgresql/postgresql_audit_log_repository_test.go`.
- [x] Task: Update `MySQLAuditLogRepository` in `internal/auth/repository/mysql/mysql_audit_log_repository.go`. 8501e96
    - [x] Update `ListCursor` to support `client_id` filtering.
    - [x] Update/Add tests in `internal/auth/repository/mysql/mysql_audit_log_repository_test.go`.
- [x] Task: Add database index for `client_id` in `audit_logs` table. c606ac9
    - [x] Create migration `000007_add_audit_log_client_id_index`.
- [x] Task: Conductor - User Manual Verification 'Phase 1' (Protocol in workflow.md) a640ed9

## Phase 2: Use Case Layer Update
- [ ] Task: Update `AuditLogUseCase` interface in `internal/auth/usecase/interface.go` to include `clientID *uuid.UUID` in `ListCursor`.
- [ ] Task: Update `auditLogUseCase` in `internal/auth/usecase/audit_log_usecase.go`.
    - [ ] Update `ListCursor` to pass `clientID` to the repository.
    - [ ] Update/Add tests in `internal/auth/usecase/audit_log_usecase_test.go`.
- [ ] Task: Update `auditLogUseCaseWithMetrics` decorator in `internal/auth/usecase/metrics_decorator.go`.
    - [ ] Update `ListCursor` signature and implementation.
    - [ ] Update tests in `internal/auth/usecase/metrics_decorator_test.go`.
- [ ] Task: Conductor - User Manual Verification 'Phase 2' (Protocol in workflow.md)

## Phase 3: HTTP Handler Layer Update
- [ ] Task: Update `AuditLogHandler.ListHandler` in `internal/auth/http/audit_log_handler.go`.
    - [ ] Parse `client_id` query parameter.
    - [ ] Validate `client_id` is a valid UUID.
    - [ ] Pass `clientID` to the use case.
    - [ ] Update/Add tests in `internal/auth/http/audit_log_handler_test.go`.
- [ ] Task: Conductor - User Manual Verification 'Phase 3' (Protocol in workflow.md)

## Phase 4: Documentation and Integration Testing
- [ ] Task: Update Documentation.
    - [ ] Document `client_id` filter in `docs/observability/audit-logs.md`.
    - [ ] Update `docs/openapi.yaml` with the new query parameter.
- [ ] Task: Update Integration Tests.
    - [ ] Add audit log filtering test case in `test/integration/auth_flow_test.go`.
- [ ] Task: Conductor - User Manual Verification 'Phase 4' (Protocol in workflow.md)
