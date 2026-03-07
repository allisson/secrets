# Specification: Fix Metrics Server Hardcoded Timeouts (REVISED)

## Overview
The Metrics Server in the `secrets` project currently has hardcoded timeout values for Read, Write, and Idle connections (15s, 15s, 60s). This track aims to make these timeouts configurable via environment variables, following the existing configuration pattern.

## Functional Requirements

1.  **Configurable Metrics Timeouts:**
    *   Introduce three new environment variables:
        *   `METRICS_SERVER_READ_TIMEOUT_SECONDS` (Default: 15)
        *   `METRICS_SERVER_WRITE_TIMEOUT_SECONDS` (Default: 15)
        *   `METRICS_SERVER_IDLE_TIMEOUT_SECONDS` (Default: 60)
    *   **Config Update:** Update `internal/config/Config` struct in `internal/config/config.go` to include these new timeout fields.
    *   **Validation:** Implement validation for these new timeouts (1s to 300s range).
    *   **Default Values:** Set the default values to 15s/15s/60s in `internal/config/config.go`.
    *   **.env.example Update:** Add these new environment variables to the `.env.example` file with their default values.

2.  **Dependency Injection (DI) Integration:**
    *   Update `internal/app/Container.initMetricsServer` in `internal/app/di.go` to pass the new timeout values from the configuration to the `MetricsServer` initialization.

3.  **Metrics Server Update:**
    *   Refactor `internal/http/metrics_server.go` to ensure `MetricsServer` uses values provided via DI instead of hardcoded defaults.
    *   Update `NewDefaultMetricsServer` or adjust its usage in `di.go` to honor the configured values.

## Non-Functional Requirements
*   **Consistency:** The configuration naming and validation logic must mirror the existing patterns for the main server.

## Acceptance Criteria
*   [ ] New environment variables are successfully loaded into the `Config` struct.
*   [ ] Configuration validation fails if any of the new timeouts are outside the 1s-300s range.
*   [ ] `.env.example` is updated with the new environment variables.
*   [ ] The Metrics Server uses the configured timeout values.
*   [ ] Existing unit tests for `MetricsServer` and `Config` pass.
*   [ ] New unit tests verify that the Metrics Server can be initialized with custom timeout values.

## Out of Scope
*   Adding other Metrics Server configuration options.
*   Changing the default values for the main server.
