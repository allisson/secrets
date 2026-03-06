# Specification: Fix Rate Limiter Goroutine Lifecycle

## Overview
Fix the rate limiter goroutine lifecycle to prevent resource leaks during server shutdown. Currently, the cleanup goroutine uses `context.Background()`, which prevents it from being cancelled when the application stops.

## Functional Requirements
- **Context Acceptance:** The rate limiter initialization must accept a `context.Context` parameter.
- **Graceful Shutdown:** The cleanup goroutine responsible for purging expired rate limit entries must use the provided context to stop its execution when the context is cancelled.
- **Initialization Update:** All call sites that initialize the rate limiter must be updated to provide a valid application context (typically derived from the main server context).

## Non-Functional Requirements
- **Performance:** The fix should not introduce any measurable overhead to the rate limiting logic or hot paths.
- **Reliability:** The rate limiter should continue to function correctly even if the context is cancelled (i.e., it should fail-safe or stop gracefully).

## Acceptance Criteria
- [ ] The rate limiter's `New` or initialization function accepts `context.Context`.
- [ ] The cleanup goroutine correctly stops when the context is cancelled.
- [ ] Automated tests confirm that the goroutine is cleaned up (e.g., using `goleak` or similar verification).
- [ ] All existing tests pass.

## Out of Scope
- Refactoring the rate limiting algorithm or storage backend.
- Adding new rate limiting features (e.g., per-user vs per-IP changes).
- Changes to other middleware or handlers unless directly related to initializing the rate limiter.
