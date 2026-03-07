# Specification: Tokenization Key Retrieval by Name

## Overview
Currently, the tokenization module supports listing keys and rotating keys, but it lacks a direct endpoint to retrieve a single key's metadata by its name. This track adds a new `GET` endpoint to the tokenization key API to allow efficient lookup of individual keys.

## Functional Requirements
- **Endpoint:** `GET /v1/tokenization/keys/:name`
- **Capability:** `tokenization:read`
- **Input:** `name` (string) as a path parameter.
- **Output:** Returns the latest version of the tokenization key metadata.
- **Status Codes:**
  - `200 OK`: Key found. Returns key metadata (ID, Name, Version, FormatType, IsDeterministic, CreatedAt).
  - `404 Not Found`: Key with the given name does not exist or has been soft-deleted.
  - `401 Unauthorized`: Missing or invalid authentication token.
  - `403 Forbidden`: Authenticated client lacks `tokenization:read` capability.

## Non-Functional Requirements
- **Consistency:** The response format must match the existing `TokenizationKeyResponse` DTO.
- **Performance:** Direct lookup by name should be efficient (indexed in the database).

## Acceptance Criteria
- [ ] A new method `GetByName` is added to `TokenizationKeyUseCase`.
- [ ] A new handler `GetByNameHandler` is added to `TokenizationKeyHandler`.
- [ ] The route `GET /v1/tokenization/keys/:name` is registered in the application.
- [ ] The endpoint requires the `tokenization:read` capability.
- [ ] Unit tests for the use case and handler are implemented.
- [ ] Integration tests verify the end-to-end functionality (MySQL and PostgreSQL).
- [ ] Updated integration tests in `test/integration/tokenization_flow_test.go`.
- [ ] Updated documentation: `docs/engines/tokenization.md`
- [ ] Updated OpenAPI documentation: `docs/openapi.yaml`

## Out of Scope
- Retrieving specific versions of a key (only the latest version is returned).
- Retrieving soft-deleted keys.
- Modifying or deleting keys via this endpoint.
