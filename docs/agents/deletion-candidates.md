# Deletion Candidates

These former root instructions are redundant, obvious, or too broad for root `AGENTS.md`:

- `make help`: useful locally, but not task-critical for every agent run.
- `make run-server`: useful for server work, but not relevant to every task.
- `make run-migrate`: useful for migration work, but not relevant to every task.
- `make test-all`: convenience wrapper, redundant with focused test targets.
- `make mocks`: useful only when interface mocks change.
- `Use standard Go formatting`: generally known by Go agents; kept only in the Go-specific guide.
- `Name tests *_test.go`: Go default; kept only in the testing guide.
- `Keep docs concise`: broad style preference; kept only in the docs-specific guide.
