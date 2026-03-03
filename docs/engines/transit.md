# 🚄 Transit Engine

The Transit API encrypts and decrypts data in transit without storing the application payload. It offers Encryption as a Service (EaaS).

## How it works

The transit engine generates a Data Encryption Key (DEK) for each transit key created. The API then uses this DEK to encrypt or decrypt data provided by the client, returning the ciphertext or plaintext directly. The system handles DEK management, versioning, and rotation automatically.

```mermaid
graph TD
    A[Client] -->|POST /v1/transit/keys/name/encrypt| B(API Server)
    B --> C{Authentication & Authorization}
    C -->|Valid| D[Retrieve DEK for Key Name]
    D --> E[Encrypt Plaintext with DEK]
    E --> F[Return Ciphertext to Client]
    C -->|Invalid| H[401/403 Error]
```

## Endpoints

All endpoints require `Authorization: Bearer <token>`.

### Create Transit Key

Creates the initial transit key version (`version = 1`).

- **Endpoint**: `POST /v1/transit/keys`
- **Capability**: `write`
- **Body**: `{"name": "string", "algorithm": "aes-gcm" | "chacha20-poly1305"}`

```bash
curl -X POST http://localhost:8080/v1/transit/keys 
  -H "Authorization: Bearer <token>" 
  -H "Content-Type: application/json" 
  -d '{"name":"payment-data","algorithm":"aes-gcm"}'
```

### Rotate Transit Key

Creates a new active version for encryption while old versions remain valid for decryption.

- **Endpoint**: `POST /v1/transit/keys/:name/rotate`
- **Capability**: `rotate`
- **Body**: `{"algorithm": "aes-gcm" | "chacha20-poly1305"}`

### Encrypt Data

- **Endpoint**: `POST /v1/transit/keys/:name/encrypt`
- **Capability**: `encrypt`
- **Body**: `{"plaintext": "base64-encoded-string"}`

```bash
curl -X POST http://localhost:8080/v1/transit/keys/payment-data/encrypt 
  -H "Authorization: Bearer <token>" 
  -H "Content-Type: application/json" 
  -d '{"plaintext":"c2Vuc2l0aXZlLWRhdGE="}'
```

Returns:

```json
{
  "ciphertext": "1:ZW5jcnlwdGVkLWJ5dGVzLi4u",
  "version": 1
}
```

Example encrypt response (`200 OK`):

```json
{
  "ciphertext": "1:ZW5jcnlwdGVkLWJ5dGVzLi4u",
  "version": 1
}
```

### Decrypt Data

- **Endpoint**: `POST /v1/transit/keys/:name/decrypt`
- **Capability**: `decrypt`
- **Body**: `{"ciphertext": "1:ZW5jcnlwdGVkLWJ5dGVzLi4u"}`

```bash
curl -X POST http://localhost:8080/v1/transit/keys/payment-data/decrypt 
  -H "Authorization: Bearer <token>" 
  -H "Content-Type: application/json" 
  -d '{"ciphertext":"1:ZW5jcnlwdGVkLi4u"}'
```

Example decrypt response (`200 OK`):

```json
{
  "plaintext": "YjY0LXBsYWludGV4dA==",
  "version": 1
}
```

### List and Delete Keys

- `GET /v1/transit/keys` (Capability: `read`)
- `DELETE /v1/transit/keys/:id` (Capability: `delete`)

## Relevant CLI Commands

- `rewrap-deks`: Rewraps transit key DEKs when rotating the KEK.
