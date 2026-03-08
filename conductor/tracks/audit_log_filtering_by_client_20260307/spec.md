# Specification: Add Audit Log Filtering by Client

## Overview
Currently, audit logs can be retrieved and filtered by date range. This track adds the ability to filter audit logs by a specific Client ID (UUID) via the API.

## Functional Requirements
- **API Filtering:** The `GET /v1/audit-logs` endpoint must support an optional `client_id` query parameter.
- **Repository Support:** The `AuditLogRepository` must implement filtering by `client_id` in its `ListCursor` method for both PostgreSQL and MySQL implementations.
- **UseCase Support:** The `AuditLogUseCase` must pass the `client_id` filter from the handler to the repository.
- **Validation:** The `client_id` provided in the query parameter must be a valid UUID.
- **Empty Results:** If no audit logs match the specified `client_id`, the API should return an empty list with a `200 OK` status.
- **Documentation:**
    - Update `docs/observability/audit-logs.md` to document the new `client_id` filter.
    - Update `docs/openapi.yaml` to include the `client_id` query parameter for the audit logs list endpoint.
- **Integration Tests:**
    - Update `test/integration/auth_flow_test.go` to include a test case for filtering audit logs by Client ID.

## Non-Functional Requirements
- **Performance:** Ensure that the database query for filtering by `client_id` is performant.
- **Consistency:** Maintain existing cursor-based pagination and date filtering logic.

## Acceptance Criteria
- [ ] `GET /v1/audit-logs?client_id=<uuid>` returns only logs belonging to that client.
- [ ] Providing an invalid UUID for `client_id` returns a `400 Bad Request` error.
- [ ] If `client_id` is omitted, the API continues to return logs for all clients (existing behavior).
- [ ] Filtering by `client_id` works correctly in combination with `created_at_from` and `created_at_to` filters.
- [ ] Filtering by `client_id` works correctly with cursor-based pagination (`after_id`).
- [ ] `docs/observability/audit-logs.md` correctly reflects the new filtering capability.
- [ ] `docs/openapi.yaml` includes the new `client_id` query parameter.
- [ ] Integration tests in `test/integration/auth_flow_test.go` pass and verify the new filtering behavior.
- [ ] PostgreSQL implementation is verified with integration tests.
- [ ] MySQL implementation is verified with integration tests.

## Out of Scope
- Filtering by Client Name.
- Adding filtering to the CLI `audit-log list` command.
