# Specification: Validate Capabilities in ParseCapabilities

## Overview
Currently, `ParseCapabilities` in the `internal/ui` package accepts any non-empty string as a valid capability. This track implements strict validation to ensure only predefined domain capabilities are accepted, aligning the CLI behavior with the documented security model.

## Functional Requirements
- **Strict Validation:** Each capability in the input string must match one of the valid capabilities defined in the `internal/auth/domain` package.
- **Error Handling:** If any single capability is invalid or unknown, the entire parsing operation must fail with a descriptive error.
- **Normalization:** Matching must be case-sensitive. Only exact, lowercase matches (as defined in the domain) are considered valid.
- **Extensible Implementation:**
    - Add a `IsValidCapability(Capability) bool` or `ValidCapabilities() []Capability` helper to `internal/auth/domain`.
    - `ParseCapabilities` should use this helper to perform validation.
- **Documentation Alignment:** Ensure that `docs/auth/policies.md` and any other relevant documentation accurately reflect these requirements (already appears aligned, but requires final verification).

## Domain Capabilities (as of v1.0.0)
- `read`
- `write`
- `delete`
- `encrypt`
- `decrypt`
- `rotate`

## Acceptance Criteria
- `ParseCapabilities("read,write")` succeeds and returns `[]authDomain.Capability{"read", "write"}`.
- `ParseCapabilities("read, invalid")` fails with an error indicating that "invalid" is not a valid capability.
- `ParseCapabilities("READ")` fails (strict case-sensitivity).
- `ParseCapabilities("")` and `ParseCapabilities(" , ")` continue to fail.
- Unit tests in `internal/ui/policies_test.go` are updated/added.
- New unit tests for domain helper are added.
- `PromptForPolicies` flow correctly handles these validation errors.
