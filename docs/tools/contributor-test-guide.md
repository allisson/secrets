# 🧪 Contributor Test Guide

> Last updated: 2026-02-28

This guide defines the testing procedures and useful commands for contributing to Secrets.

## Useful Commands

```bash
make build
make run-server
make run-migrate
make lint
make test
make test-with-db
make mocks
make docs-check-examples
```

## Running Specific Tests

### Run a specific use case test

```bash
go test -v -race -run TestKekUseCase_Create ./internal/crypto/usecase
```

### Run a specific sub-test

```bash
go test -v -race -run "TestKekUseCase_Create/Success" ./internal/crypto/usecase
```

## Test Databases

Secrets supports both PostgreSQL and MySQL. Use these commands to manage test containers:

```bash
make test-db-up     # Start PostgreSQL and MySQL containers
make test-with-db   # Run integration tests against real databases
make test-db-down   # Stop and remove containers
```

## Local Development Loop

1. Update code.
2. Run `make lint`.
3. Run targeted tests for your changes.
4. Run full `make test` before opening a PR.

## See Also

- [Contributing Guide](../contributing.md)
- [Local Run Guide](../getting-started/local-development.md)
