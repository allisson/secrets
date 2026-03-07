# Specification: Transit Key Retrieval API

## Overview
Add a new API endpoint to the transit module to allow clients to retrieve metadata for individual transit keys. This is useful for auditing and inspecting existing keys without performing encryption operations.

## Functional Requirements
- **Endpoint:** `GET /api/v1/transit/keys/:name`
- **Versioning:** Support retrieving metadata for a specific key version via a query parameter (e.g., `?version=2`). If omitted, return metadata for the latest version.
- **Capability:** Require the `read` capability for the requested path.
- **Response:**
    - `name`: String
    - `type`: String (e.g., aes256-gcm96, chacha20-poly1305)
    - `version`: Integer
    - `created_at`: RFC3339 Timestamp
    - `updated_at`: RFC3339 Timestamp

## Non-Functional Requirements
- **Security:** Ensure that the API never returns sensitive key material.
- **Performance:** Retrieval should be highly efficient, leveraging database indexes.

## Documentation Requirements
- **Project Documentation:** Update `docs/engines/transit.md` to document the new key retrieval capability.
- **API Reference:** Update `docs/openapi.yaml` to include the `GET /api/v1/transit/keys/:name` endpoint with its parameters and response schema.

## Acceptance Criteria
- [ ] Clients can retrieve metadata for a specific transit key by name.
- [ ] The API correctly handles the `version` query parameter.
- [ ] Requests without the `read` capability are rejected with `403 Forbidden`.
- [ ] Requests for non-existent keys return `404 Not Found`.
- [ ] API documentation (OpenAPI) is updated to include the new endpoint.
- [ ] Transit engine documentation in `docs/engines/transit.md` is updated.

## Out of Scope
- CLI command implementation.
- Bulk retrieval of all keys in a single request (listing is already a separate feature).
- Modification of key properties via this endpoint.
