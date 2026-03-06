# Specification: Add Client Secret Rotation Endpoint

## Overview
This track implements a new endpoint to rotate client secrets for the **Secrets** manager. It will support both self-service rotation (a client rotating its own secret) and administrative rotation (an authorized client rotating another client's secret by ID). As an additional security measure, all existing authentication tokens for the client will be revoked upon rotation.

## Functional Requirements
- **Self-Service Rotation:** A client can rotate its own secret by calling `POST /v1/auth/clients/self/rotate-secret`.
- **Administrative Rotation:** A client with sufficient privileges can rotate any client's secret by calling `POST /v1/auth/clients/:id/rotate-secret`.
- **Capability-Based Authorization:** Access to these endpoints must be controlled by the `rotate` capability.
- **Immediate Invalidation:** Once rotated, the old secret must be immediately invalidated.
- **Token Revocation:** All active authentication tokens associated with the client must be revoked immediately after secret rotation.
- **New Secret Generation:** A new, cryptographically secure random secret must be generated and hashed using the existing Argon2id implementation.
- **Response:** The response must include the new plaintext client secret (only once) and updated client metadata.

## Non-Functional Requirements
- **Security:** Plaintext secrets must never be stored; only Argon2id hashes.
- **Audit Logging:** Every rotation event must be logged in the HMAC-signed audit log.
- **Performance:** Rotation should be efficient, with sub-millisecond overhead beyond the Argon2id hashing cost.

## Acceptance Criteria
- [ ] A client can rotate its own secret and use the new secret for subsequent authentication.
- [ ] All existing tokens for the client are invalidated after its secret is rotated.
- [ ] An admin client can rotate another client's secret and receive the new secret.
- [ ] Attempting to rotate a secret without the `rotate` capability returns `403 Forbidden`.
- [ ] The old secret is no longer valid immediately after rotation.
- [ ] The audit log contains a record of the rotation event, including who performed it and for which client.

## Out of Scope
- **Grace Periods:** Not required for this phase.
- **Bulk Rotation:** Rotating multiple client secrets in a single request.
