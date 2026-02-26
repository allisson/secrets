# 🚄 Transit API

> Last updated: 2026-02-26
> Applies to: API v1

Transit API encrypts/decrypts data without storing your application payload.

## 📑 Table of Contents

- [Endpoints](#endpoints)
- [Status Code Quick Reference](#status-code-quick-reference)
- [Create Transit Key](#create-transit-key)
- [Rotate Transit Key](#rotate-transit-key)
- [Encrypt Data](#encrypt-data)
- [Decrypt Data](#decrypt-data)
- [Common Errors](#common-errors)
- [Endpoint Error Matrix](#endpoint-error-matrix)
- [Error Payload Examples](#error-payload-examples)
- [Quick Flow (Under 2 Minutes)](#-quick-flow-under-2-minutes)

## Compatibility

- API surface: `/v1/transit/keys*`
- Server expectation: Secrets server with initialized KEK and transit key endpoints enabled
- OpenAPI baseline: `docs/openapi.yaml`

All endpoints require Bearer authentication.

## Endpoints

- `GET /v1/transit/keys` (list keys)
- `POST /v1/transit/keys` (create key)
- `POST /v1/transit/keys/:name/rotate` (rotate key)
- `DELETE /v1/transit/keys/:id` (soft delete key)
- `POST /v1/transit/keys/:name/encrypt`
- `POST /v1/transit/keys/:name/decrypt`

Capability mapping:

- `GET /v1/transit/keys` -> `read`
- `POST /v1/transit/keys` -> `write`
- `POST /v1/transit/keys/:name/rotate` -> `rotate`
- `DELETE /v1/transit/keys/:id` -> `delete`
- `POST /v1/transit/keys/:name/encrypt` -> `encrypt`
- `POST /v1/transit/keys/:name/decrypt` -> `decrypt`

Wildcard matcher semantics reference:

- [Policies cookbook / Path matching behavior](../auth/policies.md#path-matching-behavior)

## Status Code Quick Reference

| Endpoint | Success | Common error statuses |
| --- | --- | --- |
| `GET /v1/transit/keys` | `200` | `401`, `403`, `422`, `429` |
| `POST /v1/transit/keys` | `201` | `401`, `403`, `409`, `422`, `429` |
| `POST /v1/transit/keys/:name/rotate` | `200` | `401`, `403`, `404`, `422`, `429` |
| `POST /v1/transit/keys/:name/encrypt` | `200` | `401`, `403`, `404`, `422`, `429` |
| `POST /v1/transit/keys/:name/decrypt` | `200` | `401`, `403`, `404`, `422`, `429` |
| `DELETE /v1/transit/keys/:id` | `204` | `401`, `403`, `404`, `422`, `429` |

## List Transit Keys

```bash
curl "http://localhost:8080/v1/transit/keys?offset=0&limit=50" \
  -H "Authorization: Bearer <token>"
```

Retrieves a paginated list of transit keys. Only the latest active version of each key name is returned.

Example response (`200 OK`):

```json
{
  "data": [
    {
      "id": "0194f4a6-7ec7-78e6-9fe7-5ca35fef48db",
      "name": "payment-data",
      "version": 2,
      "dek_id": "0194f4a6-7ec7-78e6-9fe7-5ca35fef48dc",
      "created_at": "2026-02-15T10:30:00Z"
    }
  ]
}
```

## Create Transit Key

Creates the initial transit key version (`version = 1`) for a key name.
Use `POST /v1/transit/keys/:name/rotate` to create newer versions.

### Create vs Rotate

- Use `POST /v1/transit/keys` when a key name is created for the first time.
- Use `POST /v1/transit/keys/:name/rotate` for every subsequent version.
- If create is called again for the same key name, API returns `409 Conflict`.

```bash
curl -X POST http://localhost:8080/v1/transit/keys \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"payment-data","algorithm":"aes-gcm"}'
```

Example response (`201 Created`):

```json
{
  "id": "0194f4a6-7ec7-78e6-9fe7-5ca35fef48db",
  "name": "payment-data",
  "version": 1,
  "dek_id": "0194f4a6-7ec7-78e6-9fe7-5ca35fef48dc",
  "created_at": "2026-02-14T18:30:00Z"
}
```

If a key with the same `name` already exists at version `1`, create returns
`409 Conflict` (`transit key already exists`).

### Idempotency

- `POST /v1/transit/keys` is not idempotent for an existing key name and returns `409 Conflict`.
- `POST /v1/transit/keys/:name/rotate` intentionally creates a new key version on each successful call.

## Rotate Transit Key

```bash
curl -X POST http://localhost:8080/v1/transit/keys/payment-data/rotate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"algorithm":"aes-gcm"}'
```

Rotation creates a new active version for encryption while old versions remain valid for decryption.

## Encrypt Data

```bash
curl -X POST http://localhost:8080/v1/transit/keys/payment-data/encrypt \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"plaintext":"c2Vuc2l0aXZlLWRhdGE="}'
```

Response contains versioned ciphertext, for example `1:...`.

⚠️ Security Warning: base64 is encoding, not encryption. Always use HTTPS/TLS.

Example encrypt response (`200 OK`):

```json
{
  "ciphertext": "1:ZW5jcnlwdGVkLWJ5dGVzLi4u",
  "version": 1
}
```

## Decrypt Data

```bash
curl -X POST http://localhost:8080/v1/transit/keys/payment-data/decrypt \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"ciphertext":"1:ZW5jcnlwdGVkLi4u"}'
```

For transit decrypt, pass `ciphertext` exactly as returned by encrypt (`<version>:<base64-ciphertext>`).

### Decrypt Input Contract

Valid `ciphertext` examples:

- `1:ZW5jcnlwdGVkLWJ5dGVzLi4u`
- `42:AAEC`

Invalid `ciphertext` examples:

- `ZW5jcnlwdGVkLWJ5dGVzLi4u` (missing version prefix and `:` separator)
- `1:not-base64!!!` (ciphertext segment is not valid base64)
- `abc:ZW5jcnlwdGVk` (version is not numeric)

Example decrypt response (`200 OK`):

```json
{
  "plaintext": "c2Vuc2l0aXZlLWRhdGE=",
  "version": 1
}
```

## Common Errors

- `401 Unauthorized`: missing/invalid bearer token
- `403 Forbidden`: missing capability (`write`, `rotate`, `encrypt`, `decrypt`, `delete`)
- `404 Not Found`: key missing or soft deleted
- `409 Conflict`: key already exists on create
- `422 Unprocessable Entity`: malformed request payload, invalid blob format, or invalid ciphertext base64
- `429 Too Many Requests`: per-client rate limit exceeded

## Endpoint Error Matrix

| Endpoint | 401 | 403 | 404 | 409 | 422 | 429 |
| --- | --- | --- | --- | --- | --- | --- |
| `POST /v1/transit/keys` | missing/invalid token | missing `write` capability | - | key name already initialized (`version=1`) | invalid create payload | per-client rate limit exceeded |
| `POST /v1/transit/keys/:name/rotate` | missing/invalid token | missing `rotate` capability | key name not found | - | invalid rotate payload | per-client rate limit exceeded |
| `POST /v1/transit/keys/:name/encrypt` | missing/invalid token | missing `encrypt` capability | key name not found | - | `plaintext` missing/invalid base64 | per-client rate limit exceeded |
| `POST /v1/transit/keys/:name/decrypt` | missing/invalid token | missing `decrypt` capability | key/version not found | - | malformed `<version>:<base64-ciphertext>` | per-client rate limit exceeded |
| `DELETE /v1/transit/keys/:id` | missing/invalid token | missing `delete` capability | key ID not found | - | invalid UUID | per-client rate limit exceeded |

## Error Payload Examples

Representative error payloads (exact messages may vary):

`401 Unauthorized`

```json
{
  "error": "unauthorized",
  "message": "Authentication is required"
}
```

`403 Forbidden`

```json
{
  "error": "forbidden",
  "message": "You don't have permission to access this resource"
}
```

`422 Unprocessable Entity`

```json
{
  "error": "validation_error",
  "message": "plaintext must be base64-encoded"
}
```

`409 Conflict`

```json
{
  "error": "conflict",
  "message": "transit key already exists"
}
```

Decrypt validation errors also return `422` when ciphertext is not in
`<version>:<base64-ciphertext>` format or when ciphertext payload is not valid base64.

Transit decrypt `422` examples (representative messages):

```json
{
  "error": "validation_error",
  "message": "ciphertext must be in format version:base64-ciphertext"
}
```

```json
{
  "error": "validation_error",
  "message": "ciphertext version must be numeric"
}
```

```json
{
  "error": "validation_error",
  "message": "ciphertext payload must be valid base64"
}
```

## ⚡ Quick Flow (Under 2 Minutes)

```bash
# 1) Create transit key
curl -i -X POST http://localhost:8080/v1/transit/keys \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"quickflow-transit","algorithm":"aes-gcm"}'

# 2) Encrypt value ("hello-transit" -> base64)
curl -i -X POST http://localhost:8080/v1/transit/keys/quickflow-transit/encrypt \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"plaintext":"aGVsbG8tdHJhbnNpdA=="}'
```

Expected result: key creation returns `201 Created`; encrypt returns `200 OK` with `ciphertext`.

## Use Cases

- 💳 Encrypt payment attributes before database storage
- 👤 Protect PII fields in logs or events
- 🧾 Encrypt outbound payload fragments before crossing trust boundaries

## Related Examples

- `docs/examples/curl.md`
- `docs/examples/python.md`
- `docs/examples/javascript.md`
- `docs/examples/go.md`
- `docs/api/observability/response-shapes.md`

## See also

- [Authentication API](../auth/authentication.md)
- [API error decision matrix](../fundamentals.md#error-decision-matrix)
- [API rate limiting](../fundamentals.md#rate-limiting)
- [Policies cookbook](../auth/policies.md)
- [Capability matrix](../fundamentals.md#capability-matrix)
- [Response shapes](../observability/response-shapes.md)
- [API compatibility policy](../fundamentals.md#compatibility-and-versioning-policy)
- [Curl examples](../../examples/curl.md)
- [Glossary](../../concepts/architecture.md#glossary)
