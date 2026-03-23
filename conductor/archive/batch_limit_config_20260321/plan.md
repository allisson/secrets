# Implementation Plan: Configurable Batch Limit for TokenizeBatchRequest

## Phase 1: Configuration [checkpoint: e43e7cc]
- [x] Task: Add TokenizationBatchLimit to Config struct and DefaultTokenizationBatchLimit constant in internal/config/config.go.
- [x] Task: Update `config.Load()` to load `TOKENIZATION_BATCH_LIMIT` from environment variables in `internal/config/config.go`.
- [x] Task: Update `config.Validate()` to include validation for `TokenizationBatchLimit` in `internal/config/config.go`.
- [x] Task: Update `.env.example` to include `TOKENIZATION_BATCH_LIMIT=100`.
- [x] Task: Conductor - User Manual Verification 'Configuration' (Protocol in workflow.md) e43e7cc

## Phase 2: DTO Updates [checkpoint: 86d62c6]
- [x] Task: Update `TokenizeBatchRequest.Validate` and `DetokenizeBatchRequest.Validate` in `internal/tokenization/http/dto/request.go` to accept `limit int`.
- [x] Task: Update validation rules to use `validation.Length(1, limit).Error(fmt.Sprintf("batch size exceeds limit of %d", limit))`.
- [x] Task: Update all tests calling these `Validate()` methods in `internal/tokenization/http/dto/request_test.go`.
- [x] Task: Conductor - User Manual Verification 'DTO Updates' (Protocol in workflow.md) 86d62c6

## Phase 3: Handler and DI Updates [checkpoint: 86d62c6]
- [x] Task: Update `TokenizationHandler` struct in `internal/tokenization/http/tokenization_handler.go` to include `batchLimit int`.
- [x] Task: Update `NewTokenizationHandler` in `internal/tokenization/http/tokenization_handler.go` to accept `batchLimit int`.
- [x] Task: Update `TokenizeBatchHandler` and `DetokenizeBatchHandler` in `internal/tokenization/http/tokenization_handler.go` to call `Validate(h.batchLimit)`.
- [x] Task: Update `initTokenizationHandler` in `internal/app/di_tokenization.go` to pass `c.config.TokenizationBatchLimit` to `NewTokenizationHandler`.
- [x] Task: Update `TokenizationHandler` tests in `internal/tokenization/http/tokenization_handler_test.go` to pass the batch limit to `NewTokenizationHandler`.
- [x] Task: Conductor - User Manual Verification 'Handler and DI Updates' (Protocol in workflow.md) 86d62c6

## Phase 4: Documentation
- [x] Task: Update `docs/configuration.md` with `TOKENIZATION_BATCH_LIMIT`.
- [x] Task: Update `docs/engines/tokenization.md` with information about the batch limit.
- [x] Task: Conductor - User Manual Verification 'Documentation' (Protocol in workflow.md)
