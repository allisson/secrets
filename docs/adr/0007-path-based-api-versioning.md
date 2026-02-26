# ADR 0007: Path-Based API Versioning

> Status: accepted
> Date: 2026-02-12

## Context

The system requires a versioning strategy to support API evolution while maintaining stability for consumers:

- **Stability requirement**: Consumers need backward compatibility guarantees within an API version
- **Breaking changes**: Cryptographic APIs may require breaking changes (algorithm upgrades, security improvements, schema changes)
- **Deployment constraint**: Same server binary may need to serve multiple API versions simultaneously during migration periods
- **Developer experience**: API version should be immediately visible and discoverable in documentation and examples
- **Migration support**: Clear migration path for consumers when breaking changes are introduced
- **Long-lived integrations**: Financial and cryptographic integrations often run for years without updates

## Decision

Adopt URL path-based versioning using `/v1/*` prefix for all API endpoints:

**Route structure:**

- All API endpoints under `/v1/*` prefix (e.g., `/v1/secrets/*`, `/v1/transit/keys/:name/encrypt`)
- Health/metrics endpoints outside versioning: `/health`, `/ready`, `/metrics` (not `/v1/health`)
- Future breaking changes require new version path: `/v2/*`

**Contract definition:**

- OpenAPI specification at `docs/openapi.yaml` defines v1 contract baseline
- Endpoint documentation in `docs/api/*.md` defines full public behavior
- Breaking changes documented in `releases/RELEASES.md`

**Version independence:**

- No version in headers (no `Accept: application/vnd.secrets.v1+json`)
- No version in query parameters (no `?version=1` or `?api-version=v1`)
- Version specified solely in URL path

**Coexistence strategy:**

- Multiple versions can coexist: `/v1/*` and `/v2/*` routes registered simultaneously
- Gradual migration: consumers transition at their own pace
- Deprecation timeline: old versions maintained until safe to remove (based on consumer usage metrics)

## Alternatives Considered

### 1. Header-Based Versioning

Version specified in request headers: `Accept: application/vnd.secrets.v1+json` or `X-API-Version: v1`.

**Rejected because:**

- Less visible in browser/curl (requires inspecting request headers)
- Not discoverable from URL alone (must read documentation to know header format)
- Harder to cache at CDN/proxy level (varies by header)
- More complex routing configuration (must inspect headers, not just path)
- Developer experience suffers (copy-paste URL doesn't include version)

### 2. Query Parameter Versioning

Version specified in query string: `?version=1` or `?api-version=v1`.

**Rejected because:**

- Harder to route at API gateway/proxy level (requires query param inspection)
- Query parameters typically used for filtering/pagination, not versioning
- URL caching complexity (query params affect cache keys differently)
- Inconsistent with REST best practices (version is not a filter)
- Easy to forget in code (URLs work without version, fail unexpectedly)

### 3. Subdomain Versioning

Version in subdomain: `v1.api.example.com` vs `v2.api.example.com`.

**Rejected because:**

- DNS configuration complexity (must manage multiple DNS entries)
- TLS certificate management overhead (wildcard cert or multiple certs)
- Deployment complexity (routing traffic to correct version per subdomain)
- Higher operational burden for pre-1.0 system
- Acceptable for large-scale APIs, over-engineered for current needs

### 4. No Versioning

Single evolving API with deprecation warnings for old behavior.

**Rejected because:**

- Unacceptable for cryptographic API (breaking changes too risky)
- No clear contract boundary for consumers
- Cannot safely remove deprecated features (no version to sunset)
- Financial integrations require stability guarantees
- Migration path unclear (when is it safe to remove deprecated behavior?)

## Consequences

**Benefits:**

- **URL clarity**: API version immediately visible in every request
- **Developer experience**: Copy-paste examples work, no hidden header configuration
- **Proxy/gateway friendly**: Easy to route by path prefix (`/v1/*` to v1 backend)
- **Documentation simplicity**: All examples show version in URL path
- **Caching friendly**: URL fully determines version, standard HTTP caching applies
- **Migration clarity**: `/v2/*` coexists with `/v1/*`, clear separation

**Coexistence and migration:**

- **Gradual rollout**: Deploy `/v2/*` routes alongside `/v1/*`
- **Consumer autonomy**: Clients migrate at their own pace (no forced upgrade)
- **Monitoring**: Track usage metrics per version to inform deprecation timeline
- **Deprecation process**:
  1. Announce deprecation timeline in release notes
  2. Monitor `/v1/*` usage metrics to identify remaining consumers
  3. Notify active consumers via support channels
  4. Remove deprecated version after safe sunset period (e.g., 6 months)

**Breaking change process:**

When breaking changes needed:

1. **Implement** `/v2/*` endpoints with new behavior
2. **Document** changes in `docs/releases/vX.Y.Z.md`:
   - What changed (endpoint paths, request/response schemas, status codes)
   - Migration examples (v1 request → v2 request)
   - Deprecation timeline for v1
3. **Update** `docs/openapi.yaml` with v2 contract
4. **Announce** in `releases/RELEASES.md` and release notes
5. **Monitor** `/v1/*` and `/v2/*` usage metrics
6. **Remove** `/v1/*` after sunset period

**Limitations:**

- **No per-field versioning**: All fields in API version evolve together
  - Cannot mix v1 and v2 fields in same request/response
  - Acceptable: Simplifies implementation and consumer understanding
- **URL length**: Version prefix adds characters to URL
  - Negligible impact: `/v1/` adds only 4 characters
- **Routing complexity**: Router must handle multiple version prefixes
  - Acceptable: Gin route groups make this straightforward

**Non-breaking changes:**

These can be added to existing `/v1/*` without new version:

- Adding optional request fields
- Adding new response fields (consumers must ignore unknown fields)
- Adding new endpoints under `/v1/*`
- Clarifying documentation without behavior changes

**Breaking changes** (require `/v2/*`):

- Changing endpoint paths or required path parameters
- Removing response fields or changing field meaning/type
- Changing required request fields or accepted formats
- Changing status code semantics for successful behavior

**Future considerations:**

- Could add version in response header for debugging: `X-API-Version: v1`
- Could implement automatic v1-to-v2 adapter middleware for common cases
- Could add version negotiation for advanced use cases (not needed now)

## See also

- [API versioning policy](../api/fundamentals.md#compatibility-and-versioning-policy)
- [Breaking vs non-breaking changes](../contributing.md#breaking-vs-non-breaking-docs-changes)
- [OpenAPI specification](../openapi.yaml)
- [ADR 0002: Transit Versioned Ciphertext Contract](0002-transit-versioned-ciphertext-contract.md)
- [ADR 0003: Capability-Based Authorization Model](0003-capability-based-authorization-model.md)
