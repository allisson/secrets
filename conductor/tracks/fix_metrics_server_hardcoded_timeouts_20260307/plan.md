# Implementation Plan: Fix Metrics Server Hardcoded Timeouts

## Phase 1: Configuration and Environment [checkpoint: 4ec5660]
Introduce the new configuration options for Metrics Server timeouts and update the environment files.

- [x] Task: Update `internal/config/config.go` with new constants and fields for Metrics Server timeouts. 10f5e4c
- [x] Task: Implement validation for Metrics Server timeouts in `internal/config/config.go`. f27dd3f
- [x] Task: Update `Load()` in `internal/config/config.go` to parse the new environment variables. f27dd3f
- [x] Task: Update `.env.example` to include the new `METRICS_SERVER_*` variables. f27dd3f
- [x] Task: Write failing unit tests for new configuration loading and validation in `internal/config/config_test.go`. f27dd3f
- [x] Task: Implement changes to pass the tests in `internal/config/config.go`. f27dd3f
- [x] Task: Conductor - User Manual Verification 'Phase 1: Configuration and Environment' (Protocol in workflow.md) 4ec5660

## Phase 2: Metrics Server Implementation
Refactor the Metrics Server to accept configurable timeouts instead of using hardcoded defaults.

- [x] Task: Write failing tests in `internal/http/metrics_server_test.go` to verify custom timeout initialization. a091f59
- [x] Task: Update `NewDefaultMetricsServer` or adjust its usage in `internal/http/metrics_server.go` to use passed values. a091f59
- [x] Task: Refactor `MetricsServer` initialization to ensure values are propagated correctly. a091f59
- [ ] Task: Conductor - User Manual Verification 'Phase 2: Metrics Server Implementation' (Protocol in workflow.md)

## Phase 3: Dependency Injection Integration
Connect the new configuration to the Metrics Server initialization within the DI container.

- [x] Task: Update `internal/app/di.go` to pass the configured timeouts from `Config` to the Metrics Server. 0e3de70
- [x] Task: Write tests in `internal/app/di_test.go` (or verify via integration) that the server is correctly initialized. 0e3de70
- [ ] Task: Conductor - User Manual Verification 'Phase 3: Dependency Injection Integration' (Protocol in workflow.md)
