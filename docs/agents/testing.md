# Testing

- Name tests `*_test.go`.
- Prefer table-driven test cases.
- Keep unit tests fast and isolated.
- Integration tests use the `integration` build tag.
- Database-backed tests may need the PostgreSQL service from `docker-compose.test.yml` or Makefile database targets.
- CI enforces race-enabled tests and coverage thresholds.
- Add focused tests for behavior changes.
