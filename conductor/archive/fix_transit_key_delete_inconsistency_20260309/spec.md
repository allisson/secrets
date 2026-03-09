# Specification: Fix Transit Key Delete Endpoint Inconsistency

## Overview
This track addresses an inconsistency in the Transit engine API. Currently, the `DELETE /v1/transit/keys/:id` endpoint uses a UUID (`:id`), while all other transit endpoints (GET, POST rotate/encrypt/decrypt) use the human-readable `:name`. This fix will align the DELETE endpoint with the rest of the API and the Tokenization engine's behavior.

## Functional Requirements
- Change the `DELETE /v1/transit/keys/:id` endpoint to `DELETE /v1/transit/keys/:name` in the router.
- The endpoint must accept a string `:name` instead of a UUID `:id`.
- Soft-delete all versions of the transit key associated with the given name.
- Update the `TransitKeyUseCase.Delete` method signature to accept `name string` instead of `transitKeyID uuid.UUID`.
- Update the `TransitKeyRepository.Delete` method signature to accept `name string` and update all matching records' `deleted_at` field.
- Update both PostgreSQL and MySQL repository implementations.
- Ensure proper authorization is maintained (requires `delete` capability on the path `/v1/transit/keys/:name`).
- Update the integration tests in `test/integration/transit_flow_test.go` to use `:name` instead of `:id` for deletion testing.

## Non-Functional Requirements
- Maintain API consistency across the platform.
- Ensure no regressions in other transit operations.

## Acceptance Criteria
- `DELETE /v1/transit/keys/:name` successfully soft-deletes all versions of the key.
- Integration tests confirm that all versions are marked as deleted.
- Unit tests for the handler, use case, and repository are updated/added.
- Documentation (`docs/openapi.yaml`, `docs/engines/transit.md`, `docs/concepts/api-fundamentals.md`, `docs/auth/policies.md`) reflects the change.

## Out of Scope
- Changing other engines' endpoints (Tokenization is already consistent).
- Hard-deletion of keys (remains a separate process/track).
