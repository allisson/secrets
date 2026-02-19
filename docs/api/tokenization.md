# 🎫 Tokenization API

> Last updated: 2026-02-19
> Applies to: API v1

The Tokenization API provides format-preserving token generation for sensitive values,
with optional deterministic behavior and token lifecycle management.

## Compatibility

- API surface: `/v1/tokenization/*`
- Server expectation: Secrets server with initialized KEK and tokenization migrations applied
- OpenAPI baseline: `docs/openapi.yaml` (subset coverage)

OpenAPI coverage note:

- Tokenization endpoint coverage is included in `docs/openapi.yaml` for `v0.5.0`
- This page remains the most detailed contract reference with examples and operational guidance

All endpoints require `Authorization: Bearer <token>`.

## Endpoints

Key management:

- `POST /v1/tokenization/keys` (create key)
- `POST /v1/tokenization/keys/:name/rotate` (rotate key)
- `DELETE /v1/tokenization/keys/:id` (soft delete key)

Token operations:

- `POST /v1/tokenization/keys/:name/tokenize` (generate token)
- `POST /v1/tokenization/detokenize` (retrieve original value)
- `POST /v1/tokenization/validate` (check token validity)
- `POST /v1/tokenization/revoke` (revoke token)

Capability mapping:

| Endpoint | Required capability |
| --- | --- |
| `POST /v1/tokenization/keys` | `write` |
| `POST /v1/tokenization/keys/:name/rotate` | `rotate` |
| `DELETE /v1/tokenization/keys/:id` | `delete` |
| `POST /v1/tokenization/keys/:name/tokenize` | `encrypt` |
| `POST /v1/tokenization/detokenize` | `decrypt` |
| `POST /v1/tokenization/validate` | `read` |
| `POST /v1/tokenization/revoke` | `delete` |

## Status Code Quick Reference

| Endpoint | Success | Common error statuses |
| --- | --- | --- |
| `POST /v1/tokenization/keys` | `201` | `401`, `403`, `409`, `422`, `429` |
| `POST /v1/tokenization/keys/:name/rotate` | `201` | `401`, `403`, `404`, `422`, `429` |
| `DELETE /v1/tokenization/keys/:id` | `204` | `401`, `403`, `404`, `422`, `429` |
| `POST /v1/tokenization/keys/:name/tokenize` | `201` | `401`, `403`, `404`, `422`, `429` |
| `POST /v1/tokenization/detokenize` | `200` | `401`, `403`, `404`, `422`, `429` |
| `POST /v1/tokenization/validate` | `200` | `401`, `403`, `422`, `429` |
| `POST /v1/tokenization/revoke` | `204` | `401`, `403`, `404`, `422`, `429` |

## Create Tokenization Key

Creates the initial tokenization key version (`version = 1`) for a key name.

```bash
curl -X POST http://localhost:8080/v1/tokenization/keys \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "payment-cards",
    "format_type": "luhn-preserving",
    "is_deterministic": true,
    "algorithm": "aes-gcm"
  }'
```

Request fields:

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `name` | string | Yes | Unique key name (1-255 chars) |
| `format_type` | string | Yes | `uuid`, `numeric`, `luhn-preserving`, `alphanumeric` |
| `is_deterministic` | boolean | No | Default `false` |
| `algorithm` | string | Yes | `aes-gcm` or `chacha20-poly1305` |

Example response (`201 Created`):

```json
{
  "id": "0194f4a6-7ec7-78e6-9fe7-5ca35fef48db",
  "name": "payment-cards",
  "version": 1,
  "format_type": "luhn-preserving",
  "is_deterministic": true,
  "created_at": "2026-02-18T10:30:00Z"
}
```

## Rotate Tokenization Key

Creates a new key version for an existing key name.

```bash
curl -X POST http://localhost:8080/v1/tokenization/keys/payment-cards/rotate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "format_type": "luhn-preserving",
    "is_deterministic": true,
    "algorithm": "chacha20-poly1305"
  }'
```

Example response (`201 Created`):

```json
{
  "id": "0194f4a6-8901-7def-abc0-123456789def",
  "name": "payment-cards",
  "version": 2,
  "format_type": "luhn-preserving",
  "is_deterministic": true,
  "created_at": "2026-02-18T11:00:00Z"
}
```

## Delete Tokenization Key

Soft deletes a tokenization key by ID.

```bash
curl -X DELETE http://localhost:8080/v1/tokenization/keys/0194f4a6-7ec7-78e6-9fe7-5ca35fef48db \
  -H "Authorization: Bearer <token>"
```

Response: `204 No Content`

## Tokenize Data

Generates a token for plaintext using the latest version of a key.

In deterministic mode, the same plaintext can return the same active token while valid.

```bash
curl -X POST http://localhost:8080/v1/tokenization/keys/payment-cards/tokenize \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "plaintext": "NDUzMjAxNTExMjgzMDM2Ng==",
    "metadata": {
      "last_four": "0366",
      "card_type": "visa"
    },
    "ttl": 3600
  }'
```

Request fields:

| Field | Type | Required | Description |
| --- | --- | --- | --- |
| `plaintext` | string | Yes | Base64-encoded plaintext |
| `metadata` | object | No | Display metadata; stored unencrypted |
| `ttl` | integer | No | Time-to-live in seconds (`>= 1`) |

Example response (`201 Created`):

```json
{
  "token": "4532015112830366",
  "metadata": {
    "last_four": "0366",
    "card_type": "visa"
  },
  "created_at": "2026-02-18T10:35:00Z",
  "expires_at": "2026-02-18T11:35:00Z"
}
```

## Detokenize Data

Retrieves original plaintext for a token.

```bash
curl -X POST http://localhost:8080/v1/tokenization/detokenize \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"token":"4532015112830366"}'
```

Example response (`200 OK`):

```json
{
  "plaintext": "NDUzMjAxNTExMjgzMDM2Ng==",
  "metadata": {
    "last_four": "0366",
    "card_type": "visa"
  }
}
```

Error behavior:

- Missing token mapping: `404 Not Found`
- Expired token: `422 Unprocessable Entity`
- Revoked token: `422 Unprocessable Entity`

## Validate Token

Checks whether a token is currently valid. Does not return plaintext.

```bash
curl -X POST http://localhost:8080/v1/tokenization/validate \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"token":"4532015112830366"}'
```

Example response (`200 OK`):

```json
{
  "valid": true
}
```

If token is unknown, expired, or revoked, response remains `200` with `valid: false`.

## Revoke Token

Marks a token as revoked.

```bash
curl -X POST http://localhost:8080/v1/tokenization/revoke \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"token":"4532015112830366"}'
```

Response: `204 No Content`

## Token Formats

| Format | Description | Example output |
| --- | --- | --- |
| `uuid` | RFC 4122 UUID | `01933e4a-7890-7abc-def0-123456789abc` |
| `numeric` | Numeric digits | `4532015112830366` |
| `luhn-preserving` | Numeric with Luhn validity | `4532015112830366` |
| `alphanumeric` | Letters and digits | `A3b9X2k7Q1m5` |

## Security Notes

- Metadata is not encrypted; do not store full PAN, secrets, or regulated payloads in metadata.
- Base64 is encoding, not encryption; always use HTTPS/TLS.
- Clear detokenized plaintext from memory after use in application code.

## Data Classification for Metadata

Safe metadata examples (recommended):

- last-four display fragments (for example `"0366"`)
- token source/system tags (for example `"checkout-service"`)
- non-sensitive workflow identifiers

Never place in metadata:

- full PAN, CVV, account numbers, passwords, API keys, or secrets
- plaintext payload copies already represented by the tokenized value
- personal data requiring encryption at rest if your policy requires protected storage

If data must remain confidential at rest, keep it in encrypted plaintext payload, not metadata.

## See also

- [Authentication](authentication.md)
- [API error decision matrix](error-decision-matrix.md)
- [API rate limiting](rate-limiting.md)
- [Policies](policies.md)
- [Capability matrix](capability-matrix.md)
- [CLI Commands](../cli/commands.md)
- [Production operations](../operations/production.md)
