# 🧩 API Compatibility and Versioning Policy

> Last updated: 2026-02-14
> Applies to: API v1

This page defines compatibility expectations for HTTP API changes.

## Compatibility Contract

- Current public baseline is API v1 (`/v1/*`)
- Existing endpoint paths and JSON field names are treated as stable unless explicitly deprecated
- OpenAPI source of truth: `docs/openapi.yaml`

## Breaking Changes

Treat these as breaking:

- changing endpoint paths or required path parameters
- removing response fields or changing field meaning/type
- changing required request fields or accepted formats
- changing status code semantics for successful behavior

Required process for breaking changes:

1. Update `docs/openapi.yaml`
2. Update affected API docs and examples
3. Add migration notes in `docs/getting-started/troubleshooting.md` or relevant runbook
4. Add explicit entry in `docs/CHANGELOG.md`

## Non-Breaking Changes

Usually non-breaking:

- adding optional request/response fields
- adding new endpoints under `/v1/*`
- clarifying documentation text and examples
- adding additional error examples without changing behavior

## Deprecation Guidance

- Mark deprecated behavior clearly in endpoint docs
- Provide replacement behavior and example migration path
- Keep deprecated behavior available long enough for operational rollout

## See also

- [Authentication API](authentication.md)
- [Response shapes](response-shapes.md)
- [Contributing guide](../contributing.md)
- [Documentation changelog](../CHANGELOG.md)
