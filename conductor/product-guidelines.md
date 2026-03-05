# Product Guidelines: Secrets

## UX Principles
- **Security-First Defaults:** All default configurations prioritize the highest security settings. Convenience never overrides system integrity.
- **High-Visibility Feedback:** Both CLI and API provide immediate, unambiguous feedback for all operations. Success/failure states are clearly indicated with actionable context.
- **Simplified Setup Experience:** Minimize "day-zero" friction. The system is designed to be operational with minimal required configuration, using sensible defaults where security isn't compromised.

## Error Handling & Messaging
- **Concise & Safe Tone:** Error messages are direct and technical. They must never leak internal system state, sensitive credentials, or detailed stack traces to the end-user.
- **Remediation-Oriented:** Where possible, provide clear next steps for the user to resolve the error without compromising security.

## Engineering Standards
- **Modular Clean Architecture:** Strict adherence to the architecture defined in the `internal/` directory (Domain, Usecase, Service, Repository, HTTP). Cross-layer leaks are prohibited.
- **Test-Driven Development Culture:** Every new feature or bug fix MUST be accompanied by relevant unit and integration tests. No code is merged without verified test coverage.
- **Security-Focused CI/CD:** Mandatory security scanning (`gosec`, `govulncheck`) and linting (`golangci-lint`) on every commit. Vulnerabilities are treated as blockers.

## API & Interface Design
- **URL-Based Versioning:** All API endpoints are versioned in the URL path (e.g., `/v1/secrets`). Breaking changes require a new major version.
- **RESTful Consistency:** Standard HTTP methods (GET, POST, PUT, DELETE) and status codes are used consistently across all resources.
- **Unified Response Format:** All API responses follow a standardized envelope structure to ensure predictability for client integrations.

## Documentation Style (Extended from docs-style-guide.md)
- **Action-Oriented:** Documentation prioritizes "how-to" guides and practical examples over theoretical explanations.
- **Copy-Safe Examples:** All examples use clearly synthetic values (e.g., `<client-id>`, `example.com`) to prevent accidental leaks.
