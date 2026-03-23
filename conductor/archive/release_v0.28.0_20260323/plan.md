# Implementation Plan: v0.28.0 Release Preparation

## Phase 1: Version Update & Changelog [checkpoint: 1052e1b]
- [x] Task: Update `version` to `v0.28.0` in `cmd/app/main.go` 165ae80
- [x] Task: Add v0.28.0 entry to `CHANGELOG.md` with all key changes: 61fdeda
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
- [x] Task: Conductor - User Manual Verification 'Phase 1: Version Update & Changelog' (Protocol in workflow.md) 1052e1b

## Phase 2: Documentation & OpenAPI Sync [checkpoint: a01f409]
- [x] Task: Run `make docs-lint` and address any issues.
- [x] Task: Audit `docs/openapi.yaml` and update it with new endpoints: cce88e6
    - `/api/v1/tokenization/tokenize/batch` (POST)
    - `/api/v1/tokenization/detokenize/batch` (POST)
    - `/api/v1/tokenization/keys/{name}` (GET)
    - `/api/v1/transit/keys/{name}` (GET)
    - Verify audit log filtering params for `/api/v1/audit/logs`.
- [x] Task: Conductor - User Manual Verification 'Phase 2: Documentation & OpenAPI Sync' (Protocol in workflow.md) a01f409

## Phase 3: Final Verification [checkpoint: ec975f9]
- [x] Task: Run full test suite using `make test-all`. 60b5e6f
- [x] Task: Perform a final sanity check of the CHANGELOG and CLI version output. 60b5e6f
- [x] Task: Conductor - User Manual Verification 'Phase 3: Final Verification' (Protocol in workflow.md) ec975f9
