# 🎫 Tokenization Engine

The Tokenization API provides format-preserving token generation for sensitive values, with optional deterministic behavior and token lifecycle management.

## How it works

Tokenization replaces sensitive data (like a credit card number) with a non-sensitive substitute, known as a token. The engine handles the generation, mapping, storage, and detokenization of these values.

```mermaid
graph TD
    A[Client] -->|POST /tokenize| B(API Server)
    B --> C{Authentication & Authorization}
    C -->|Valid| D[Generate Token based on Format]
    D --> E[Store Token <-> Ciphertext mapping]
    E --> F[Return Token to Client]
    A -->|POST /detokenize| B
    B -->|Valid| G[Lookup Ciphertext by Token]
    G --> H[Decrypt & Return Plaintext]
```

## Endpoints

All endpoints require `Authorization: Bearer <token>`.

### Create Tokenization Key

- **Endpoint**: `POST /v1/tokenization/keys`
- **Capability**: `write`
- **Body**: `name`, `format_type` (`uuid`, `numeric`, `luhn-preserving`, `alphanumeric`), `is_deterministic`, `algorithm`.

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

### Rotate Tokenization Key

- **Endpoint**: `POST /v1/tokenization/keys/:name/rotate`
- **Capability**: `rotate`

### Tokenize Data

- **Endpoint**: `POST /v1/tokenization/keys/:name/tokenize`
- **Capability**: `encrypt`
- **Body**: `plaintext` (base64), `metadata` (optional object), `ttl` (optional seconds).

```bash
curl -X POST http://localhost:8080/v1/tokenization/keys/payment-cards/tokenize \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "plaintext": "NDUzMjAxNTExMjgzMDM2Ng==",
    "metadata": { "last_four": "0366" }
  }'
```

Example response (`201 Created`):

```json
{
  "token": "4532015112830366",
  "metadata": {
    "last_four": "0366"
  },
  "created_at": "2026-02-27T10:35:00Z",
  "expires_at": "2026-02-27T11:35:00Z"
}
```

### Detokenize Data

- **Endpoint**: `POST /v1/tokenization/detokenize`
- **Capability**: `decrypt`
- **Body**: `{"token": "string"}`

Example response (`200 OK`):

```json
{
  "plaintext": "NDUzMjAxNTExMjgzMDM2Ng==",
  "metadata": {
    "last_four": "0366"
  }
}
```

### Validate and Revoke

- `POST /v1/tokenization/validate` (Capability: `read`) - Check if token is valid without returning plaintext.
- `POST /v1/tokenization/revoke` (Capability: `delete`) - Marks a token as revoked.

### List and Delete Keys

#### List Tokenization Keys

- **Endpoint**: `GET /v1/tokenization/keys`
- **Capability**: `read`
- **Query Params**:
  - `after_name` (optional) - Cursor for pagination. Omit for first page.
  - `limit` (default 50, max 1000) - Number of items per page.
- **Success**: `200 OK`

```bash
# First page
curl "http://localhost:8080/v1/tokenization/keys?limit=50" \
  -H "Authorization: Bearer <token>"

# Subsequent pages (use next_cursor from previous response)
curl "http://localhost:8080/v1/tokenization/keys?after_name=payment-tokens&limit=50" \
  -H "Authorization: Bearer <token>"
```

Example response (`200 OK`):

```json
{
  "data": [
    {
      "id": "0194f4c1-82de-7f9a-c2b3-9def1a7bc5d8",
      "name": "customer-ids",
      "format_type": "uuid",
      "algorithm": "aes-gcm",
      "is_deterministic": true,
      "version": 2,
      "created_at": "2026-02-27T20:10:00Z",
      "updated_at": "2026-02-28T10:30:00Z"
    },
    {
      "id": "0194f4d3-a5bc-7e2f-d8a1-4bef2c9ad7e1",
      "name": "payment-tokens",
      "format_type": "luhn-preserving",
      "algorithm": "chacha20-poly1305",
      "is_deterministic": false,
      "version": 1,
      "created_at": "2026-02-27T21:45:00Z",
      "updated_at": "2026-02-27T21:45:00Z"
    }
  ],
  "next_cursor": "payment-tokens"
}
```

**Note**: The `next_cursor` field is only present when there are more pages available.

#### Get Tokenization Key by Name

- **Endpoint**: `GET /v1/tokenization/keys/:name`
- **Capability**: `read`
- **Success**: `200 OK`

Example response (`200 OK`):

```json
{
  "id": "0194f4c1-82de-7f9a-c2b3-9def1a7bc5d8",
  "name": "customer-ids",
  "format_type": "uuid",
  "algorithm": "aes-gcm",
  "is_deterministic": true,
  "version": 2,
  "created_at": "2026-02-27T20:10:00Z",
  "updated_at": "2026-02-28T10:30:00Z"
}
```

#### Delete Tokenization Key

- **Endpoint**: `DELETE /v1/tokenization/keys/:name`
- **Capability**: `delete`
- **Success**: `204 No Content`

## Deterministic Tokenization

When `is_deterministic` is set to `true`, the engine ensures that the same plaintext value always produces the same token *under the same key version*.

- **Security**: To prevent rainbow table attacks, each key version generates a unique random 32-byte salt. The engine uses HMAC-SHA256 with this salt to compute a unique hash for each plaintext.
- **Equality Matching**: This mode allows for equality matching and duplicate detection within your application without exposing the sensitive plaintext.
- **Rotation**: When a key is rotated, a new salt is generated. Identical plaintext tokenized under the new version will produce a different token than the previous version.

## Relevant CLI Commands

- `rewrap-deks`: Rewraps tokenization key DEKs when rotating the KEK.
