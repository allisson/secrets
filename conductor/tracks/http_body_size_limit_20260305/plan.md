# Implementation Plan: HTTP Request Body Size Limit Middleware

## Phase 1: Configuration Updates
- [ ] Task: Update Application Configuration
    - [ ] Add `MaxRequestBodySize` to the main configuration struct.
    - [ ] Update config parsing to read `MAX_REQUEST_BODY_SIZE` from environment variables, defaulting to 1 MB (1048576 bytes).
    - [ ] Write unit tests to verify configuration loading and defaults.
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Configuration Updates' (Protocol in workflow.md)

## Phase 2: Middleware Implementation
- [ ] Task: Create Request Body Size Middleware
    - [ ] Write failing unit tests for a new middleware that enforces a maximum body size (e.g., verifying 413 response for large payloads, 200 for small ones).
    - [ ] Implement the middleware in `internal/http/middleware.go` (or a dedicated `body_limit.go` in the same package) using `http.MaxBytesReader` or standard Gin mechanisms.
    - [ ] Ensure the middleware uses the standard `413 Payload Too Large` error format.
    - [ ] Run tests to ensure they pass.
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Middleware Implementation' (Protocol in workflow.md)

## Phase 3: Global Integration
- [ ] Task: Integrate Middleware into Router
    - [ ] Add the body limit middleware to the global Gin router in `internal/http/server.go` (or where the global router is instantiated).
    - [ ] Update any necessary server integration tests to accommodate the middleware.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Global Integration' (Protocol in workflow.md)