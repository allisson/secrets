# Implementation Plan: Fix Metrics Server Hardcoded Timeouts

## Phase 1: Configuration and Environment
Introduce the new configuration options for Metrics Server timeouts and update the environment files.

- [x] Task: Update `internal/config/config.go` with new constants and fields for Metrics Server timeouts. 10f5e4c
- [ ] Task: Implement validation for Metrics Server timeouts in `internal/config/config.go`.
- [ ] Task: Update `Load()` in `internal/config/config.go` to parse the new environment variables.
- [ ] Task: Update `.env.example` to include the new `METRICS_SERVER_*` variables.
- [ ] Task: Write failing unit tests for new configuration loading and validation in `internal/config/config_test.go`.
- [ ] Task: Implement changes to pass the tests in `internal/config/config.go`.
- [ ] Task: Conductor - User Manual Verification 'Phase 1: Configuration and Environment' (Protocol in workflow.md)

## Phase 2: Metrics Server Implementation
Refactor the Metrics Server to accept configurable timeouts instead of using hardcoded defaults.

- [ ] Task: Write failing tests in `internal/http/metrics_server_test.go` to verify custom timeout initialization.
- [ ] Task: Update `NewDefaultMetricsServer` or adjust its usage in `internal/http/metrics_server.go` to use passed values.
- [ ] Task: Refactor `MetricsServer` initialization to ensure values are propagated correctly.
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Metrics Server Implementation' (Protocol in workflow.md)

## Phase 3: Dependency Injection Integration
Connect the new configuration to the Metrics Server initialization within the DI container.

- [ ] Task: Update `internal/app/di.go` to pass the configured timeouts from `Config` to the Metrics Server.
- [ ] Task: Write tests in `internal/app/di_test.go` (or verify via integration) that the server is correctly initialized.
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Dependency Injection Integration' (Protocol in workflow.md)
