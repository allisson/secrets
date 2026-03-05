# Specification: HTTP Request Body Size Limit Middleware

## Overview
This track introduces an HTTP middleware to limit the size of incoming request bodies. This is a crucial security enhancement to prevent Denial-of-Service (DoS) attacks caused by excessively large payloads.

## Functional Requirements
- **Middleware Implementation:** Create a Gin middleware that intercepts incoming HTTP requests.
- **Size Limitation:** The middleware must restrict the request body size. If the size exceeds the limit, it must immediately return a standard `413 Payload Too Large` HTTP response.
- **Global Application:** The size limit must apply globally to all HTTP routes.
- **Configuration:** The maximum body size must be configurable via an environment variable (e.g., `MAX_REQUEST_BODY_SIZE`).
- **Default Limit:** If the environment variable is not provided, the default maximum request body size should be 1 MB.

## Non-Functional Requirements
- **Performance:** The middleware must evaluate the request size efficiently with minimal overhead.
- **Security:** Prevents resource exhaustion attacks (OOM, excessive disk/CPU usage) from large payloads.

## Acceptance Criteria
- [ ] The middleware is implemented and integrated into the global Gin router.
- [ ] Requests with bodies smaller than or equal to the limit are processed normally.
- [ ] Requests with bodies exceeding the limit are rejected with a standard `413 Payload Too Large` status code.
- [ ] The size limit can be configured via an environment variable.
- [ ] If no environment variable is provided, the limit defaults to 1 MB.
- [ ] Unit tests verify both successful requests and rejected oversized requests.

## Out of Scope
- Custom route-specific limits or exemptions.
- Advanced streaming size limits beyond standard `http.MaxBytesReader` or equivalent Gin mechanics.