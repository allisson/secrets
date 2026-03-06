# Specification: Secret Path Validation

## Overview
This track implements validation for secret paths within the Secrets manager. The goal is to ensure that all secret paths follow a consistent format and adhere to specific length constraints to maintain database integrity and API predictability.

## Functional Requirements
- **Maximum Length:** The secret path must not exceed 255 characters.
- **Character Restrictions:**
    - Only alphanumeric characters (`a-z`, `A-Z`, `0-9`), hyphens (`-`), underscores (`_`), and forward slashes (`/`) are allowed.
    - Forward slashes are used to define hierarchical secret paths (e.g., `production/database/password`).
- **Validation Layer:** Validation must be primarily enforced within the **Domain Layer** (entities and use cases) to ensure business logic consistency across all entry points.
- **Error Handling:** When a secret path fails validation, the system must return a `422 Unprocessable Entity` status code with a descriptive error message.

## Non-Functional Requirements
- **Consistency:** The validation logic should be centralized and reused across all secret-related operations (create, update).
- **Comprehensive Testing:** Unit tests must cover all validation logic, including success and various failure cases.
- **Documentation:** Review and update project documentation (e.g., OpenAPI spec, user guides) to reflect the new validation rules.

## Acceptance Criteria
- [ ] Creating a secret with a path longer than 255 characters returns a `422 Unprocessable Entity` error.
- [ ] Creating a secret with a path containing disallowed characters (e.g., spaces, special characters other than `-`, `_`, `/`) returns a `422 Unprocessable Entity` error.
- [ ] Creating a secret with a valid path (e.g., `app-1/prod/db_secret`) succeeds.
- [ ] Updating an existing secret with an invalid path follows the same validation rules.
- [ ] Comprehensive unit tests for validation logic in the domain layer.
- [ ] Project documentation (including `docs/openapi.yaml`) updated to reflect the 255-character limit and allowed characters.

## Out of Scope
- **Retroactive Validation:** Existing secrets with paths that do not meet these criteria will not be automatically updated or flagged. Validation only applies to new creations and updates.
- **Automatic Truncation:** The system will not automatically truncate long paths; it will reject them.
