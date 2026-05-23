# Agent Guide

`secrets` is a Go service and CLI for envelope encryption, transit encryption, API auth, and signed audit logs.

Use Go modules and Makefile targets, not npm.

## Essential Commands

- `make build`: compile `bin/app`.
- `make test`: run race-enabled unit tests and write `coverage.out`.
- `make test-with-db`: run database-backed tests.
- `make test-integration`: run tests with the `integration` build tag.
- `make lint`: run `golangci-lint --fix` and `govulncheck`.
- `make docs-lint` and `make docs-check-examples`: validate documentation.

## Task Guides

- [Project layout](docs/agents/project-layout.md)
- [Go conventions](docs/agents/go-conventions.md)
- [Testing](docs/agents/testing.md)
- [Database and migrations](docs/agents/database.md)
- [Security model](docs/agents/security.md)
- [Docs, changelog, and releases](docs/agents/docs-and-releases.md)
- [Git and pull requests](docs/agents/git-workflow.md)
- [Deletion candidates](docs/agents/deletion-candidates.md)
