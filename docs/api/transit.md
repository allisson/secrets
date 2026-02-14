# 🚄 Transit API

> Last updated: 2026-02-14
> Applies to: API v1

Transit API encrypts/decrypts data without storing your application payload.

## Compatibility

- API surface: `/v1/transit/keys*`
- Server expectation: Secrets server with initialized KEK and transit key endpoints enabled
- OpenAPI baseline: `docs/openapi.yaml`

All endpoints require Bearer authentication.

## Endpoints

- `POST /v1/transit/keys` (create key)
- `POST /v1/transit/keys/:name/rotate` (rotate key)
- `DELETE /v1/transit/keys/:id` (soft delete key)
- `POST /v1/transit/keys/:name/encrypt`
- `POST /v1/transit/keys/:name/decrypt`

Capability mapping:

- `POST /v1/transit/keys` -> `write`
- `POST /v1/transit/keys/:name/rotate` -> `rotate`
- `DELETE /v1/transit/keys/:id` -> `delete`
- `POST /v1/transit/keys/:name/encrypt` -> `encrypt`
- `POST /v1/transit/keys/:name/decrypt` -> `decrypt`

## Create Transit Key

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
  "algorithm": "aes-gcm",
  "version": 1,
  "created_at": "2026-02-14T18:30:00Z"
}
```

## Rotate Transit Key

```bash
curl -X POST http://localhost:8080/v1/transit/keys/payment-data/rotate \
  -H "Authorization: Bearer <token>"
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
  "plaintext": "c2Vuc2l0aXZlLWRhdGE="
}
```

## Common Errors

- `401 Unauthorized`: missing/invalid bearer token
- `403 Forbidden`: missing capability (`write`, `rotate`, `encrypt`, `decrypt`, `delete`)
- `404 Not Found`: key missing or soft deleted
- `409 Conflict`: key already exists on create
- `422 Unprocessable Entity`: malformed request payload, invalid blob format, or invalid ciphertext base64

## Error Payload Examples

Representative error payloads (exact messages may vary):

`401 Unauthorized`

```json
{
  "error": "unauthorized",
  "message": "missing or invalid bearer token"
}
```

`403 Forbidden`

```json
{
  "error": "forbidden",
  "message": "insufficient capability for path"
}
```

`422 Unprocessable Entity`

```json
{
  "error": "validation_error",
  "message": "plaintext must be base64-encoded"
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
- `docs/api/response-shapes.md`

## See also

- [Authentication API](authentication.md)
- [Policies cookbook](policies.md)
- [Response shapes](response-shapes.md)
- [Curl examples](../examples/curl.md)
