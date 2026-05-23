# Project Layout

- Entry points live in `cmd/app`.
- Core code lives under `internal`.
- Feature modules such as `auth`, `secrets`, `transit`, and `tokenization` use `domain`, `usecase`, `service`, `repository`, and `http` layers.
- Documentation, OpenAPI, examples, and ADRs live in `docs`.

## Suggested Docs Structure

```text
docs/
  agents/          Agent-facing repo instructions
  adr/             Architecture decision records
  auth/            Authentication and authorization docs
  concepts/        Architecture and security concepts
  engines/         Secrets, transit, and tokenization behavior
  examples/        Client and deployment examples
  getting-started/ Setup and first-run guides
  operations/      Deployment, KMS, observability, runbooks, troubleshooting
  tools/           Documentation tooling and contributor guides
```
