# Specification: Configurable Batch Limit for TokenizeBatchRequest

## Overview
The `TokenizeBatchRequest.Validate` method currently uses a hardcoded value of 100 for the batch limit. This track aims to make this limit configurable via the global configuration using the `TOKENIZATION_BATCH_LIMIT` key, defaulting to 100 if not specified.

## Functional Requirements
- **Configuration Integration**: Add `TOKENIZATION_BATCH_LIMIT` to the global configuration structure and support its initialization from environment variables.
- **Dynamic Validation**: Update the `TokenizeBatchRequest.Validate` method to use the configured batch limit instead of the hardcoded value.
- **Error Message Update**: Ensure the error message returned when the limit is exceeded is `batch size exceeds limit of %d`, where `%d` is the current limit.
- **Update Documentation**: Update relevant documentation (e.g., `docs/configuration.md`, `docs/engines/tokenization.md`) to reflect the new configuration option.
- **Update `.env.example`**: Add `TOKENIZATION_BATCH_LIMIT` with the default value of 100 to the `.env.example` file.

## Acceptance Criteria
- [ ] `TOKENIZATION_BATCH_LIMIT` is successfully added to the configuration and can be set via an environment variable.
- [ ] The `TokenizeBatchRequest.Validate` method correctly uses the value from the configuration.
- [ ] If `TOKENIZATION_BATCH_LIMIT` is not set, the system defaults to a limit of 100.
- [ ] When a batch exceeds the limit, the error message correctly includes the configured limit value.
- [ ] Documentation (`docs/configuration.md`, `docs/engines/tokenization.md`) is updated.
- [ ] `.env.example` is updated.

## Out of Scope
- Modifying the core tokenization or batch processing logic.
- Adding limits to other batch operations beyond `TokenizeBatchRequest`.
