# Specification: Batch Tokenize/Detokenize Endpoints

## Overview
This track introduces batch processing capabilities to the Tokenization Engine. Currently, tokenization and detokenization are performed on a single item at a time. This feature will add new endpoints to allow clients to tokenize or detokenize multiple items in a single request, wrapped in a database transaction for atomicity.

## Functional Requirements
- **New Endpoints:**
  - `POST /v1/tokenization/keys/:name/tokenize-batch`: Batch tokenize a list of values using a named key.
  - `POST /v1/tokenization/detokenize-batch`: Batch detokenize a list of tokens.
- **Batch Limit:** A configurable limit of 100 items per batch request will be enforced to ensure performance and prevent resource exhaustion.
- **Atomicity:** Both batch endpoints MUST be atomic. If any single item in the batch fails (e.g., validation error, database failure), the entire request MUST fail, and any database changes MUST be rolled back.
- **Request/Response Formats:**
  - `tokenize-batch`:
    - Request: `{"values": ["val1", "val2", ...]}`
    - Response: `{"tokens": ["token1", "token2", ...]}`
  - `detokenize-batch`:
    - Request: `{"tokens": ["token1", "token2", ...]}`
    - Response: `{"values": ["val1", "val2", ...]}`
- **Documentation:**
  - Update `docs/engines/tokenization.md` to include batch operations.
  - Update `docs/openapi.yaml` with the new endpoint definitions.

## Non-Functional Requirements
- **Performance:** Batch processing should be more efficient than multiple single-item calls by reducing network round-trips and utilizing a single database transaction.
- **Security:** Standard capability validation (`tokenize` or `detokenize`) must be enforced for the batch operations.

## Acceptance Criteria
- [ ] Clients can successfully tokenize up to 100 values in a single call.
- [ ] Clients can successfully detokenize up to 100 tokens in a single call.
- [ ] If any value in a `tokenize-batch` request is invalid, the entire request returns an error (400 Bad Request) and no tokens are created.
- [ ] If any token in a `detokenize-batch` request is invalid, the entire request returns an error (400 Bad Request) and no values are returned.
- [ ] The batch limit is enforced and returns a 400 Bad Request if exceeded.
- [ ] Unit tests cover new domain logic, usecase methods, and HTTP handlers.
- [ ] Integration tests in `test/integration/tokenization_flow_test.go` cover batch operations for both PostgreSQL and MySQL.
- [ ] Documentation (`docs/engines/tokenization.md`) and OpenAPI spec (`docs/openapi.yaml`) are updated.

## Out of Scope
- Partial success/failure handling for batch requests.
- Asynchronous batch processing.
