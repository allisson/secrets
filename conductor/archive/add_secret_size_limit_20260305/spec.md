# Specification: Add Secret Value Size Limit

## Overview
Implement a global limit on the size of secret values stored in the system. This prevents potential denial-of-service (DoS) attacks and ensures predictable storage and memory usage.

## Functional Requirements
- **Global Size Limit:** Enforce a maximum size for the value of a secret (e.g., in bytes or kilobytes).
- **Default Limit:** Set a default global limit of 512 KB (524,288 bytes).
- **Enforcement:** The system MUST check the size of the secret value BEFORE it is encrypted and stored.
- **Error Handling:** If a secret value exceeds the configured limit, the system MUST return a `413 Payload Too Large` error with a clear message.
- **Configuration:** The limit MUST be configurable via environment variables (e.g., `SECRET_VALUE_SIZE_LIMIT_BYTES`).

## Non-Functional Requirements
- **Performance:** The size check should have negligible impact on performance.
- **Security:** Ensure the check happens early enough to avoid unnecessary processing (e.g., encryption) for oversized payloads.

## Acceptance Criteria
1.  **GIVEN** a global limit of 512 KB is configured
    **WHEN** a user attempts to store a secret value <= 512 KB
    **THEN** the request should succeed.

2.  **GIVEN** a global limit of 512 KB is configured
    **WHEN** a user attempts to store a secret value > 512 KB
    **THEN** the system should reject the request with a `413 Payload Too Large` error.

3.  **GIVEN** the `SECRET_VALUE_SIZE_LIMIT_BYTES` environment variable is changed
    **WHEN** the server is restarted
    **THEN** the new limit should be enforced.

## Out of Scope
- Per-path or per-secret limits.
- Dynamic limit updates without server restart.
- Limits on metadata or other fields.
