# Implementation Plan: Add Secret Value Size Limit

## Phase 1: Configuration Support [checkpoint: ea1a529]
Implement the infrastructure to load the secret value size limit from environment variables.

- [x] Task: Add `SecretValueSizeLimitBytes` to `internal/config/config.go` 000d34d
    - [x] Add field to `Config` struct
    - [x] Update loading logic (default to 524,288 bytes)
    - [x] Add unit test in `internal/config/config_test.go`
- [x] Task: Conductor - User Manual Verification 'Phase 1: Configuration Support' (Protocol in workflow.md) ea1a529

## Phase 2: Use Case Enforcement [checkpoint: 521776b]
Integrate the size limit validation into the core business logic.

- [x] Task: Implement size validation in `internal/secrets/usecase/` 5f90059
    - [x] Identify the method responsible for creating/updating secrets
    - [x] Add check against `config.SecretValueSizeLimitBytes`
    - [x] Add failing TDD test in usecase test file
    - [x] Implement validation logic to pass tests
- [x] Task: Conductor - User Manual Verification 'Phase 2: Use Case Enforcement' (Protocol in workflow.md) 521776b
## Phase 3: HTTP Error Handling and Integration [checkpoint: ca5998c]
Ensure the API layer correctly handles validation errors and returns the appropriate HTTP status code.

- [x] Task: Verify HTTP response for size limit exceeded in `internal/secrets/http/` 51dc385
    - [x] Ensure the usecase returns an error that maps to `413 Payload Too Large`
    - [x] Update or add handler tests to verify 413 status
- [x] Task: Conductor - User Manual Verification 'Phase 3: HTTP Error Handling and Integration' (Protocol in workflow.md) ca5998c

## Phase 4: Documentation Update [checkpoint: 75d32fd]
Update the project documentation to reflect the new configuration and behavior.

- [x] Task: Update configuration documentation a6d3ffe
    - [x] Add `SECRET_VALUE_SIZE_LIMIT_BYTES` to `docs/configuration.md`
    - [x] Update `.env.example` with the new variable
- [x] Task: Update API documentation a6d3ffe
    - [x] Mention the 413 response in `docs/openapi.yaml` for secret creation/update endpoints
- [x] Task: Conductor - User Manual Verification 'Phase 4: Documentation Update' (Protocol in workflow.md) 75d32fd
