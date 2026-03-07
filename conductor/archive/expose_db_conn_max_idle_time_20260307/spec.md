# Specification: Expose DB ConnMaxIdleTime Configuration

## Overview
Currently, the `Secrets` application allows configuring several database connection pool parameters (`DB_MAX_OPEN_CONNECTIONS`, `DB_MAX_IDLE_CONNECTIONS`, `DB_CONN_MAX_LIFETIME_MINUTES`), but it does not expose the `ConnMaxIdleTime` setting, which controls how long a connection can remain idle before being closed. This track will expose this configuration to provide better control over connection management.

## Functional Requirements
1.  **Configuration Structure:** Add `DBConnMaxIdleTime` (of type `time.Duration`) to the `Config` struct in `internal/config/config.go`.
2.  **Environment Variable:** Support loading this value from the environment variable `DB_CONN_MAX_IDLE_TIME_MINUTES`.
3.  **Default Value:** Set a default value of 5 minutes (`DefaultDBConnMaxIdleTime = 5`).
4.  **Dependency Injection:** Update `internal/app/di.go` to pass the `DBConnMaxIdleTime` from the configuration to the `database.Connect` function.
5.  **Validation:** Ensure the value is validated (e.g., non-negative).
6.  **Documentation:** Update `docs/configuration.md` to include information about the new `DB_CONN_MAX_IDLE_TIME_MINUTES` setting.
7.  **Example Environment File:** Update `.env.example` to include the new environment variable with its default value.

## Non-Functional Requirements
- **Consistency:** Follow the existing naming conventions for database configuration.

## Acceptance Criteria
- The application correctly loads `DB_CONN_MAX_IDLE_TIME_MINUTES` from the environment.
- The default value of 5 minutes is used if the environment variable is not set.
- The `database.Connect` function receives and applies the configured `ConnMaxIdleTime`.
- Unit tests in `internal/config/config_test.go` verify the new configuration field.
- `docs/configuration.md` correctly reflects the new configuration option.
- `.env.example` includes the new variable.

## Out of Scope
- Modifying other database pool settings.
- Changing the default values for existing settings.
