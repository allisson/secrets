# 🛠️ Development and Testing

> Last updated: 2026-02-14

## Useful commands

```bash
make build
make run-server
make run-migrate
make lint
make test
make test-with-db
make mocks
```

## Run specific tests

```bash
go test -v -race -run TestKekUseCase_Create ./internal/crypto/usecase
go test -v -race -run "TestKekUseCase_Create/Success" ./internal/crypto/usecase
```

## Test databases

```bash
make test-db-up
make test
make test-db-down
```

## Local development loop

1. Update code
2. Run `make lint`
3. Run targeted tests
4. Run full `make test`
