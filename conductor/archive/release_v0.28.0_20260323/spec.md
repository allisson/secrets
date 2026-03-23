# Track Specification: v0.28.0 Release Preparation

**Overview:**
Prepare for the release of version v0.28.0. This includes updating the version number, generating the changelog for the significant features and fixes since v0.27.0, and performing final documentation/API audits.

**Functional Requirements:**
- **Update Version:** Update `version` in `cmd/app/main.go` and ensure all relevant locations reflect `v0.28.0`.
- **Update Changelog:** Generate and append release notes to `CHANGELOG.md` based on commits since `v0.27.0`.
- **Key Changes for v0.28.0:**
    - **New Feature:** Configurable batch limit for tokenize and detokenize operations.
    - **Transit Engine:** Key deletion by name instead of UUID (#120).
    - **Transit Engine:** Individual key retrieval API (#115).
    - **Tokenization Engine:** Atomic batch tokenize and detokenize endpoints (#119).
    - **Tokenization Engine:** Delete tokenization keys by name (#117).
    - **Tokenization Engine:** Individual key retrieval API by name (#116).
    - **Audit Logs:** Implement audit log filtering by `client_id` (#118).
    - **Configuration:** Make Metrics Server timeouts configurable (#114).
    - **Database:** Expose DB connection max idle time configuration (#113).
    - **Auth:** Client secret rotation with automatic token revocation.
    - **Auth:** Fix rate limiter goroutine lifecycle and resource leaks (#112).
    - **Auth:** Implement strict capability validation for policies (#111).
- **Documentation/OpenAPI Audit:** Run `make docs-lint` and ensure `docs/openapi.yaml` matches the current API state.

**Acceptance Criteria:**
- `cmd/app/main.go` reports version `v0.28.0`.
- `CHANGELOG.md` contains a comprehensive entry for v0.28.0 with all key changes correctly categorized.
- `make docs-lint` passes without errors.
- `docs/openapi.yaml` reflects the new batch tokenization, filtering, and key retrieval endpoints.

**Out of Scope:**
- Functional code changes (other than version bump).
- Deployment/Infrastructure changes.
