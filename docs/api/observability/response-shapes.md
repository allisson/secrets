# 🧱 API Response Shapes

> Last updated: 2026-02-20
> Applies to: API v1

Use these representative response schemas as a stable reference across endpoint docs.

## Success shapes

Token issuance:

```json
{
  "token": "tok_...",
  "expires_at": "2026-02-14T20:13:45Z"
}
```

Client creation:

```json
{
  "id": "0194f4a6-7ec7-78e6-9fe7-5ca35fef48db",
  "secret": "sec_..."
}
```

Secret write:

```json
{
  "id": "0194f4a5-73fe-7a7d-a3a0-6fbe9b5ef8f3",
  "path": "/app/prod/database-password",
  "version": 3,
  "created_at": "2026-02-14T18:22:00Z"
}
```

Secret read:

```json
{
  "id": "0194f4a5-73fe-7a7d-a3a0-6fbe9b5ef8f3",
  "path": "/app/prod/database-password",
  "version": 3,
  "value": "YjY0LXBsYWludGV4dA==",
  "created_at": "2026-02-14T18:22:00Z"
}
```

Transit encrypt:

```json
{
  "ciphertext": "1:...",
  "version": 1
}
```

Transit decrypt:

```json
{
  "plaintext": "YjY0LXBsYWludGV4dA=="
}
```

Tokenization key create:

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

Tokenize:

```json
{
  "token": "4532015112830366",
  "metadata": {
    "last_four": "0366"
  },
  "created_at": "2026-02-18T10:35:00Z",
  "expires_at": "2026-02-18T11:35:00Z"
}
```

Detokenize:

```json
{
  "plaintext": "NDUzMjAxNTExMjgzMDM2Ng==",
  "metadata": {
    "last_four": "0366"
  }
}
```

Token validate:

```json
{
  "valid": true
}
```

Input contract note: transit decrypt expects `ciphertext` in format
`<version>:<base64-ciphertext>`. See [Transit API](../data/transit.md#decrypt-input-contract).

Audit log list:

```json
{
  "audit_logs": [
    {
      "id": "0194f4a7-8fbe-7e3b-b7b2-72f3ac8f6ed0",
      "request_id": "0194f4a7-8fbc-73c1-a114-88c1d8682cb7",
      "client_id": "0194f4a6-7ec7-78e6-9fe7-5ca35fef48db",
      "capability": "decrypt",
      "path": "/v1/secrets/app/prod/database-password",
      "metadata": {
        "allowed": true,
        "ip": "192.168.1.10",
        "user_agent": "curl/8.7.1"
      },
      "created_at": "2026-02-14T18:35:12Z"
    }
  ]
}
```

## Error shape

Representative error structure used in docs:

```json
{
  "error": "validation_error",
  "message": "invalid request body"
}
```

Common error categories:

- `unauthorized`
- `forbidden`
- `rate_limit_exceeded`
- `validation_error`
- `not_found`
- `conflict`

Representative rate-limit payload (`429 Too Many Requests`):

```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many requests. Please retry after the specified delay."
}
```

Rate-limit responses include a `Retry-After` header in seconds.

For `POST /v1/token`, the `message` text may be token-endpoint specific while keeping the same
error key and `429` contract.

Representative token endpoint payload (`POST /v1/token`, `429 Too Many Requests`):

```json
{
  "error": "rate_limit_exceeded",
  "message": "Too many token requests from this IP. Please retry after the specified delay."
}
```

Representative conflict payload (for example duplicate transit key create):

```json
{
  "error": "conflict",
  "message": "transit key already exists"
}
```

## See also

- [Authentication API](../auth/authentication.md)
- [API rate limiting](../fundamentals.md#rate-limiting)
- [API error decision matrix](../fundamentals.md#error-decision-matrix)
- [Clients API](../auth/clients.md)
- [Secrets API](../data/secrets.md)
- [Transit API](../data/transit.md)
- [Tokenization API](../data/tokenization.md)
- [API compatibility policy](../fundamentals.md#compatibility-and-versioning-policy)
- [Glossary](../../concepts/architecture.md#glossary)
