# ADR 0006: Dual-Scope Rate Limiting Strategy

> Status: accepted
> Date: 2026-02-19

## Context

The system must protect against abuse, denial-of-service attacks, and credential stuffing while maintaining fair resource allocation:

- **Abuse protection**: Prevent malicious actors from overwhelming the system with requests
- **DoS mitigation**: Protect server resources from exhaustion by limiting request rates
- **Different threat models**: Authenticated endpoints vs unauthenticated endpoints face different attack vectors
- **Credential stuffing risk**: Token issuance endpoint vulnerable to brute-force credential attacks
- **Per-client fairness**: Prevent one authenticated client from monopolizing server resources
- **Operational simplicity**: Avoid external dependencies (Redis, API gateway) for pre-1.0 deployment

## Decision

Implement dual-scope rate limiting with different strategies for authenticated vs unauthenticated endpoints:

### Authenticated Endpoints (Per-Client Rate Limiting)

**Scope**: Each authenticated client gets an independent rate limiter identified by client ID.

**Configuration**:

- `RATE_LIMIT_ENABLED` (default: true)
- `RATE_LIMIT_REQUESTS_PER_SEC` (default: 10.0)
- `RATE_LIMIT_BURST` (default: 20)

**Behavior**:

- Requires `AuthenticationMiddleware` (must run after authentication)
- Token bucket algorithm from `golang.org/x/time/rate` library
- Separate bucket per client ID (extracted from authenticated context)
- Automatic cleanup of stale limiters after 1 hour of inactivity

**Protected routes**:

- `/v1/clients/*`
- `/v1/audit-logs`
- `/v1/secrets/*`
- `/v1/transit/*`
- `/v1/tokenization/*`

### Unauthenticated Token Endpoint (Per-IP Rate Limiting)

**Scope**: Each client IP address gets an independent rate limiter.

**Configuration**:

- `RATE_LIMIT_TOKEN_ENABLED` (default: true)
- `RATE_LIMIT_TOKEN_REQUESTS_PER_SEC` (default: 5.0)
- `RATE_LIMIT_TOKEN_BURST` (default: 10)

**Behavior**:

- Applied to `POST /v1/token` (unauthenticated endpoint)
- Token bucket algorithm from `golang.org/x/time/rate` library
- Separate bucket per client IP (extracted via `c.ClientIP()`)
- IP detection handles `X-Forwarded-For`, `X-Real-IP`, and direct connection
- Stricter default limits than authenticated endpoints (credential stuffing mitigation)
- Automatic cleanup of stale limiters after 1 hour of inactivity

**Response behavior** (both scopes):

- Status: `429 Too Many Requests`
- Header: `Retry-After: <seconds>`
- JSON body: `{"error": "rate_limit_exceeded", "message": "..."}`

## Alternatives Considered

### 1. Global Rate Limit

Single shared rate limit across all clients/IPs.

**Rejected because:**

- Noisy neighbor problem: one misbehaving client affects all legitimate clients
- Cannot differentiate between high-volume legitimate users and attackers
- Unfair resource allocation (first come, first served)
- No per-client or per-IP isolation

### 2. Redis-Based Distributed Rate Limiting

External Redis store for rate limit state shared across server instances.

**Rejected because:**

- Adds operational dependency (Redis must be deployed, monitored, backed up)
- Additional latency for every request (Redis round trip)
- Increased complexity (connection pooling, failover, retry logic)
- Not needed for pre-1.0 (single instance deployment acceptable)
- Trade-off: Cannot share rate limit state across multiple instances (acceptable for current scale)

### 3. API Gateway Rate Limiting

Offload rate limiting to external API gateway (Kong, NGINX, AWS API Gateway).

**Rejected because:**

- Requires additional infrastructure and deployment complexity
- Reduces deployment simplicity (goal: single binary + database)
- Splits configuration between application and gateway
- Still need application-level rate limiting for business logic control
- Acceptable for future scale, not needed now

### 4. IP-Only Rate Limiting (Including Authenticated Endpoints)

Use single IP-based mechanism for all endpoints.

**Rejected because:**

- Shared NATs/proxies would unfairly throttle legitimate users behind same IP
- Corporate networks, cloud NATs, and residential ISPs share IPs across many users
- Cannot provide per-client fairness for authenticated API usage
- Credential stuffing protection still needed (addressed by token endpoint IP limiting)

### 5. Client ID-Only Rate Limiting (Including Token Endpoint)

Use single client-based mechanism for all endpoints.

**Rejected because:**

- Token endpoint is unauthenticated (no client ID available yet)
- Attacker can try many client credentials without rate limit
- Credential stuffing attacks would be unrestricted

## Consequences

**Benefits:**

- **Simple implementation**: In-process rate limiting, no external dependencies
- **Low latency**: No external service calls, direct memory access
- **Per-client fairness**: Authenticated clients cannot affect each other's rate limits
- **Credential stuffing protection**: IP-based limiting protects unauthenticated token endpoint
- **Operational simplicity**: No Redis, no API gateway, no additional infrastructure
- **Configurable limits**: Operators can tune limits per deployment environment

**Trade-offs and Limitations:**

- **In-process state**: Rate limiter state lost on server restart
  - Impact: Fresh rate limit buckets after deployment (acceptable, temporary burst allowed)
  - Mitigation: Graceful shutdown drains existing requests before restart

- **Memory growth**: Limiter map grows with unique clients/IPs
  - Impact: Memory usage increases with client/IP diversity
  - Mitigation: Automatic cleanup after 1 hour of inactivity
  - Acceptable: Typical deployments have bounded client/IP counts

- **IP-based limitations for token endpoint**:
  - **Shared IPs (NAT/proxies)**: Multiple legitimate users behind same corporate NAT or ISP may share IP, hitting limit together
  - **X-Forwarded-For spoofing**: Attacker could rotate IPs in header if reverse proxy not properly configured
  - Mitigations:
    - Reasonable burst capacity (10 requests) handles legitimate retries
    - Can disable via `RATE_LIMIT_TOKEN_ENABLED=false` if IP limiting problematic
    - Configure Gin's trusted proxy settings in production deployments
    - Deploy behind properly configured reverse proxy/load balancer

- **No cross-instance coordination**: Each server instance has independent rate limiters
  - Impact: Rate limits are per-instance, not globally enforced
  - Acceptable: Pre-1.0 deployments typically run single instance
  - Future: Could add Redis-based distributed rate limiting if multi-instance needed

**Security considerations:**

- Token endpoint uses stricter limits (5 req/sec vs 10 req/sec) to protect against credential attacks
- Burst capacity allows legitimate retry behavior while limiting sustained abuse
- Rate limit metrics exposed via Prometheus for monitoring and alerting
- `429` responses logged for security audit and attack detection

**Future enhancements:**

- Could add Redis-based distributed rate limiting for multi-instance deployments
- Could add adaptive rate limiting based on system load
- Could add allowlist/blocklist for specific IPs or clients
- Could add custom rate limits per client ID (database-stored configuration)

## See also

- [Rate limiting fundamentals](../api/fundamentals.md#rate-limiting)
- [Monitoring rate limiting metrics](../operations/observability/monitoring.md#rate-limiting-observability-queries)
- [Configuration](../configuration.md)
