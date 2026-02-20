# ADR 0008: Gin Web Framework with Custom Middleware Strategy

> Status: accepted
> Date: 2026-01-29

## Context

The application requires an HTTP server for REST API endpoints with the following needs:

- High-performance request routing with path parameters and route groups
- Middleware chain support for cross-cutting concerns (logging, authentication, rate limiting)
- Request/response binding and validation
- Custom error handling and response formatting
- Production-ready timeouts and graceful shutdown
- Integration with structured logging (slog) instead of default logging
- Request ID tracking for distributed tracing

Standard library `net/http` provides basic routing but lacks ergonomic path parameters, route grouping, and middleware chaining. Full-featured frameworks like Echo, Fiber, or Chi offer different trade-offs in performance, API design, and ecosystem maturity.

## Decision

Adopt **Gin v1.11.0** as the web framework with a **custom middleware strategy** that bypasses Gin's default middleware:

**Core choices:**

- Use `gin.New()` instead of `gin.Default()` to avoid default middleware (Logger, Recovery)
- Manually configure `http.Server` with explicit timeouts:
  - `ReadTimeout: 15s`
  - `WriteTimeout: 15s`
  - `IdleTimeout: 60s`
- Replace Gin's default logger with custom slog-based middleware
- Keep Gin's Recovery middleware but add custom middleware for:
  - Request ID tracking (UUIDv7 via `gin-contrib/requestid`)
  - Custom structured logging (slog)
  - Authentication (Bearer token validation)
  - Authorization (capability-based access control)
  - Rate limiting (dual-scope: per-client and per-IP)
  - Audit logging (authorization attempts)
- Auto-configure Gin mode from `LOG_LEVEL` environment variable (debug → `gin.DebugMode`, else `gin.ReleaseMode`)

**Middleware execution order:**

1. `gin.Recovery()` - Panic recovery
2. `requestid.New()` - UUIDv7 request ID generation
3. `CustomLoggerMiddleware()` - slog-based HTTP request logging
4. `AuditLogMiddleware()` - Audit log persistence (route group level)
5. `AuthenticationMiddleware()` - Bearer token validation (route-specific)
6. `AuthorizationMiddleware()` - Capability enforcement (route-specific)
7. `RateLimitMiddleware()` - Per-client or per-IP throttling (route-specific)
8. Handler

**Route organization:**

- Health/readiness endpoints outside API versioning: `/health`, `/ready`, `/metrics`
- Versioned API routes under `/v1/*` using route groups
- Per-endpoint middleware chaining for fine-grained control

## Alternatives Considered

### 1. Echo Framework

Popular alternative with similar performance characteristics.

**Rejected because:**

- Less mature ecosystem for middleware (no official `gin-contrib` equivalent)
- Different context abstraction (`echo.Context` vs `gin.Context`) less familiar to team
- Smaller community compared to Gin (fewer third-party middleware packages)
- No significant performance advantage over Gin for our workload

### 2. Fiber Framework

Express.js-inspired framework built on fasthttp.

**Rejected because:**

- Uses fasthttp instead of `net/http` (incompatible with standard middleware ecosystem)
- More opinionated API design conflicts with Clean Architecture boundaries
- Migration complexity if we need to switch back to `net/http` standard library
- Performance gains not justified for cryptographic workload (CPU-bound, not I/O-bound)

### 3. Chi Router

Lightweight router built on `net/http` with middleware support.

**Rejected because:**

- More verbose route parameter extraction compared to Gin (`chi.URLParam(r, "id")` vs `c.Param("id")`)
- No built-in request/response binding (requires manual JSON marshaling)
- No built-in validation framework integration
- Less ergonomic for rapid API development while maintaining Clean Architecture

### 4. Standard Library (`net/http` + `http.ServeMux`)

Minimal dependencies with Go 1.22+ improved routing.

**Rejected because:**

- Lacks route groups for middleware scoping (all middleware must be global or per-handler)
- No built-in path parameter support before Go 1.22, limited after
- No request/response binding helpers (increases boilerplate in handlers)
- No middleware chaining abstraction (manual wrapper functions required)
- Development velocity trade-off not justified for a pre-1.0 project

### 5. Use Gin's Default Middleware (`gin.Default()`)

Simpler setup with built-in Logger and Recovery.

**Rejected because:**

- Default logger outputs plain text, incompatible with structured logging (slog)
- No request ID tracking (essential for distributed tracing and incident correlation)
- Cannot customize log format to include custom fields (client ID, capability, path)
- Default logger not production-ready (no JSON output, no log level control)

## Consequences

**Benefits:**

- **High development velocity**: Ergonomic API with minimal boilerplate for route parameters, JSON binding, and validation
- **Custom observability**: Full control over logging format, request ID propagation, and audit trails
- **Fine-grained middleware control**: Per-route middleware application (auth on protected routes, rate limiting per scope)
- **Production-ready defaults**: Explicit timeout configuration prevents resource exhaustion
- **Ecosystem compatibility**: Large middleware ecosystem (`gin-contrib/*`) and community support
- **Testability**: `gin.CreateTestContext()` simplifies handler unit testing
- **Clean Architecture compatibility**: `gin.Context` isolated to HTTP layer, domain/use case layers remain framework-agnostic

**Trade-offs:**

- **Framework dependency**: Vendor lock-in to Gin's routing and context abstraction (migration requires rewriting HTTP layer)
- **Middleware maintenance**: Custom middleware increases code surface area (vs using default middleware)
- **Learning curve**: Team must understand Gin-specific patterns (middleware chain, context usage, route groups)

**Limitations:**

- **No built-in OpenAPI generation**: Must maintain `docs/openapi.yaml` manually (acceptable for pre-1.0 with stable API)
- **No HTTP/2 server push**: Not supported by Gin's `http.Server` wrapper (not needed for REST API)
- **Context-bound data**: Request-scoped data (authenticated client, audit metadata) stored in `gin.Context` instead of Go `context.Context` (acceptable trade-off for ergonomics)

**Future considerations:**

- Monitor Gin maintenance activity (stable v1.x releases, active issue triage)
- Evaluate OpenAPI code generation tools if manual maintenance becomes burden
- Consider migration path to standard library if Go routing improves significantly
- Could adopt `net/http` middleware adapters if we need standard library compatibility

## See also

- [HTTP server implementation](../../internal/http/server.go)
- [Custom logger middleware](../../internal/http/middleware.go)
- [ADR 0009: UUIDv7 for Identifiers](0009-uuidv7-for-identifiers.md) - Request ID generation strategy
- [ADR 0006: Dual-Scope Rate Limiting Strategy](0006-dual-scope-rate-limiting-strategy.md) - Rate limiting middleware
- [API fundamentals](../api/fundamentals.md)
