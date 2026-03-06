# Implementation Plan: Validate Capabilities in ParseCapabilities

## Overview
Enhance `ParseCapabilities` in `internal/ui` to strictly validate capabilities against the `internal/auth/domain` constants.

## Phase 1: Domain Enhancement [checkpoint: 39909a2]
Add validation helpers to the domain package to ensure extensibility.

- [x] Task: Add `ValidCapabilities()` and `IsValidCapability()` to `internal/auth/domain/const.go`. 6d24b0e
- [x] Task: Add unit tests for `IsValidCapability()` in `internal/auth/domain/const_test.go` (create file if needed). 6d24b0e
- [x] Task: Conductor - User Manual Verification 'Phase 1: Domain Enhancement' (Protocol in workflow.md)

## Phase 2: UI Implementation [checkpoint: 6ce42dd]
Update `ParseCapabilities` to use the domain validation.

- [x] Task: Update `internal/ui/policies_test.go` with failing test cases for invalid capabilities and case-sensitivity. d7770f6
- [x] Task: Update `internal/ui/policies.go`'s `ParseCapabilities` function to perform validation using the domain helpers. d7770f6
- [x] Task: Verify that `PromptForPolicies` correctly bubbles up these errors (existing tests or add new ones). d7770f6
- [x] Task: Conductor - User Manual Verification 'Phase 2: UI Implementation' (Protocol in workflow.md)

## Phase 3: Documentation and Quality Gates [checkpoint: 911c90f]
Ensure everything is consistent and meets quality standards.

- [x] Task: Verify `docs/auth/policies.md` and `docs/cli-commands.md` align with the strict validation (already appear to, but double check examples). 3bb2424
- [x] Task: Run full test suite (`make test`) and linting (`make lint`). 3bb2424
- [x] Task: Conductor - User Manual Verification 'Phase 3: Documentation and Quality Gates' (Protocol in workflow.md)
