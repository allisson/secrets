# Implementation Plan: Fix Rate Limiter Goroutine Lifecycle

## Phase 1: Research and Infrastructure [checkpoint: b7b5bbd]
- [x] Task: Identify the rate limiter implementation and initialization points b7b5bbd
    - [x] Locate the rate limiter package in `internal/`
    - [x] Find all call sites where the rate limiter is initialized
- [x] Task: Identify existing tests and reproduction steps b7b5bbd
    - [x] Locate existing unit tests for the rate limiter
    - [x] Confirm how the cleanup goroutine is currently started
- [x] Task: Conductor - User Manual Verification 'Research and Infrastructure' (Protocol in workflow.md) b7b5bbd

## Phase 2: Implementation (TDD) [checkpoint: b9177d8]
- [x] Task: Write failing test for goroutine leak c072dd9
    - [x] Create a test case that initializes the rate limiter and then cancels the context
    - [x] Verify that the cleanup goroutine persists after cancellation (failing state)
- [x] Task: Update rate limiter initialization to accept context 2b5cbc2
    - [x] Modify the signature of the initialization function to include `context.Context`
- [x] Task: Update cleanup goroutine to use context 2b5cbc2
    - [x] Modify the cleanup loop to listen for `ctx.Done()`
- [x] Task: Update call sites for rate limiter initialization 2b5cbc2
    - [x] Update all middleware and/or `main.go` call sites to pass the appropriate context
- [x] Task: Verify fix with automated tests 2b5cbc2
    - [x] Run the failing test and ensure it now passes
    - [x] Run the full test suite to ensure no regressions
- [x] Task: Conductor - User Manual Verification 'Implementation' (Protocol in workflow.md) b9177d8

## Phase 3: Final Verification and Cleanup [checkpoint: 9b2011d]
- [x] Task: Run all tests and verify no regressions 2b5cbc2
    - [x] Execute `make test` and `make test-all`
- [x] Task: Verify code coverage for new changes 2b5cbc2
    - [x] Ensure new code has >80% coverage
- [x] Task: Conductor - User Manual Verification 'Final Verification and Cleanup' (Protocol in workflow.md) 9b2011d
