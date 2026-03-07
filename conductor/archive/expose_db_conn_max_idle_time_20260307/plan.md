# Implementation Plan: Expose DB ConnMaxIdleTime Configuration

## Phase 1: Configuration & Environment [checkpoint: fd1cdc3]
- [x] Task: Update `.env.example` with `DB_CONN_MAX_IDLE_TIME_MINUTES=5` (8385017)
- [x] Task: Update `internal/config/config.go` to include `DefaultDBConnMaxIdleTime` and `DBConnMaxIdleTime` field. (de33982)
- [x] Task: Write failing unit tests in `internal/config/config_test.go` for the new configuration field (Red Phase). (36b3f36)
- [x] Task: Implement `Load()` and `Validate()` in `internal/config/config.go` to support `DB_CONN_MAX_IDLE_TIME_MINUTES` (Green Phase). (c985d84)
- [x] Task: Conductor - User Manual Verification 'Phase 1: Configuration & Environment' (Protocol in workflow.md) (fd1cdc3)

## Phase 2: Dependency Injection & Integration [checkpoint: 67cebf7]
- [x] Task: Update `internal/app/di.go` to pass `DBConnMaxIdleTime` from the configuration to `database.Connect`. (396da05)
- [x] Task: Add integration or manual verification test to ensure `ConnMaxIdleTime` is correctly applied to the database pool. (0089ec3)
- [x] Task: Conductor - User Manual Verification 'Phase 2: Dependency Injection & Integration' (Protocol in workflow.md) (67cebf7)

## Phase 3: Documentation [checkpoint: 95eeb98]
- [x] Task: Update `docs/configuration.md` to document the `DB_CONN_MAX_IDLE_TIME_MINUTES` setting. (95ebccc)
- [x] Task: Conductor - User Manual Verification 'Phase 3: Documentation' (Protocol in workflow.md) (95eeb98)
