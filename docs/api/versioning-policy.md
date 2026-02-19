# 🧩 API Compatibility and Versioning Policy

> Last updated: 2026-02-19
> Applies to: API v1

This page defines compatibility expectations for HTTP API changes.

## Compatibility Contract

- Current public baseline is API v1 (`/v1/*`)
- Existing endpoint paths and JSON field names are treated as stable unless explicitly deprecated
- OpenAPI source of truth: `docs/openapi.yaml`

## OpenAPI Coverage (v0.5.0)

- `docs/openapi.yaml` is a baseline subset focused on high-traffic/common integration flows
- `docs/openapi.yaml` includes tokenization endpoint coverage in `v0.5.0`
- `docs/openapi.yaml` includes `429 Too Many Requests` response modeling for protected routes
- Endpoint pages in `docs/api/*.md` define full public behavior for covered operations
- Endpoints may exist in runtime before they are expanded in OpenAPI detail

## App Version vs API Version

- Application release `v0.5.0` is pre-1.0 software and may evolve quickly
- API v1 path contract (`/v1/*`) remains the compatibility baseline for consumers
- Breaking API behavior changes require explicit documentation and migration notes

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

## Telemetry Change Examples

Breaking telemetry examples:

- renaming a published metric name (for example `secrets_http_requests_total`)
- renaming/removing metric labels used by dashboards or alerts

Non-breaking telemetry examples:

- adding a new metric family
- adding new label values for existing labels
- adding new dashboard examples without changing metric contracts

## Deprecation Guidance

- Mark deprecated behavior clearly in endpoint docs
- Provide replacement behavior and example migration path
- Keep deprecated behavior available long enough for operational rollout

## See also

- [Authentication API](authentication.md)
- [API error decision matrix](error-decision-matrix.md)
- [Response shapes](response-shapes.md)
- [Contributing guide](../contributing.md)
- [Documentation changelog](../CHANGELOG.md)
