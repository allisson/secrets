# Specification: Auth Token Revocation

## Overview
This track implements a token revocation mechanism for the Secrets manager. Since tokens are opaque strings stored in the database, this track will add state management to track their validity beyond their expiration timestamp.

## Functional Requirements
- **Token Revocation Endpoints:**
  - `DELETE /v1/token`: Revokes the current bearer token used in the request.
  - `DELETE /v1/clients/:id/tokens`: Revokes all active tokens for the specified client ID. This endpoint requires the requester to have `delete` capability on the target client.
- **Audit Logs:** Generate an HMAC-signed audit log entry for every successful revocation.
- **Revocation Storage:** Since tokens are already stored in the database, add a `revoked_at` timestamp to the tokens table to mark them as revoked.
- **Validation Logic:** Update the token validation logic in the authentication middleware to reject tokens that have a non-null `revoked_at` timestamp.
- **Purge Command:** Implement a CLI command `purge-auth-tokens` to permanently delete revoked and expired tokens from the database.
- **Documentation:**
  - Update API documentation (OpenAPI and Markdown docs) to include the new revocation endpoints.
  - Update the "Policies Cookbook" to include examples of how to restrict or grant access to the new revocation features.

## Non-Functional Requirements
- **Performance:** Validation checks must be optimized with database indexes on the `revoked_at` field and client IDs.
- **Security:** Ensure that the `DELETE /v1/clients/:id/tokens` endpoint is properly protected by capability-based authorization.
- **Data Integrity:** Maintain consistency between the revocation state and audit logs.

## Acceptance Criteria
- [ ] `DELETE /v1/token` successfully marks the current token as revoked in the database.
- [ ] `DELETE /v1/clients/:id/tokens` successfully marks all tokens for client `:id` as revoked.
- [ ] Subsequent requests using a revoked token are rejected with a `401 Unauthorized` error.
- [ ] Audit logs correctly record the actor, action, and target of each revocation.
- [ ] `secrets purge-auth-tokens` deletes revoked and expired tokens from the database.
- [ ] API documentation is updated.
- [ ] Policies cookbook is updated with revocation-related examples.
- [ ] Implementation is verified against both PostgreSQL and MySQL.
- [ ] Unit and integration tests cover all new logic, middleware, and CLI command.

## Out of Scope
- **Automatic Background Purging:** Periodic removal of expired/revoked records without manual CLI invocation.
- **UI Integration:** Changes to the web UI.
