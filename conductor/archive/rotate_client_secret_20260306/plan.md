# Implementation Plan: Add Client Secret Rotation Endpoint

This plan outlines the implementation of a new endpoint to rotate client secrets, including automatic token revocation for the rotated client.

## Phase 1: Usecase Layer [checkpoint: 54d001b]
Implement the core rotation logic in the `ClientUseCase`.

- [x] Task: Add `RotateSecret(ctx context.Context, clientID uuid.UUID) (*authDomain.CreateClientOutput, error)` to `ClientUseCase` interface in `internal/auth/usecase/interface.go`. (9ba1694)
- [x] Task: Implement `RotateSecret` in `internal/auth/usecase/client_usecase.go`. The implementation should:
    - Get the client.
    - Generate a new secret using `secretService.GenerateSecret()`.
    - Update the client's hashed secret.
    - Save the client using `clientRepo.Update` within a transaction.
    - Revoke all existing tokens for the client using `tokenRepo.RevokeByClientID`.
    - Create an audit log entry for the rotation. (3429ca8)
- [x] Task: Add unit tests for `RotateSecret` in `internal/auth/usecase/client_usecase_test.go`. (3429ca8)
- [x] Task: Conductor - User Manual Verification 'Phase 1' (Protocol in workflow.md) (54d001b)

## Phase 2: HTTP Transport Layer [checkpoint: e8fcdae]
Expose the rotation logic via new API endpoints.

- [x] Task: Add `RotateSecret` method to `ClientHandler` in `internal/auth/http/client_handler.go`. (b3104f1)
- [x] Task: Implement `RotateSecret` handler logic to handle both self-service (`/self/rotate-secret`) and administrative (`/:id/rotate-secret`) requests. (b3104f1)
- [x] Task: Register the new routes in the Gin router (likely in `internal/app/di_auth.go` or where routes are defined). (b3104f1)
- [x] Task: Add unit tests for the new handler in `internal/auth/http/client_handler_test.go`. (b3104f1)
- [x] Task: Conductor - User Manual Verification 'Phase 2' (Protocol in workflow.md) (e8fcdae)

## Phase 3: CLI Commands [checkpoint: 3faf19f]
Expose the rotation functionality via the CLI.

- [x] Task: Add a new `rotate-secret` command to the client-related CLI commands in `cmd/app/auth_commands.go`. (c029ff7)
- [x] Task: Implement the CLI logic to call the new rotation endpoint. (c029ff7)
- [x] Task: Update CLI documentation in `docs/cli-commands.md`. (c029ff7)
- [x] Task: Conductor - User Manual Verification 'Phase 3' (Protocol in workflow.md) (3faf19f)

## Phase 4: Final Validation and Documentation [checkpoint: 855f89f]
Complete integration testing and update project documentation.

- [x] Task: Add integration tests in `test/integration/auth_flow_test.go` to verify the full rotation and revocation flow. (7542bc3)
- [x] Task: Update OpenAPI specification in `docs/openapi.yaml` to include the new endpoint. (7542bc3)
- [x] Task: Verify HMAC-signed audit logs for rotation events. (ea2ffc1)
- [x] Task: Conductor - User Manual Verification 'Phase 4' (Protocol in workflow.md) (855f89f)
