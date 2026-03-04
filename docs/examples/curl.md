# 🧪 Curl Examples

⚠️ Security Warning: base64 is encoding, not encryption. Always use HTTPS/TLS.

End-to-end shell workflow.

Need first credentials? Create an API client with `app create-client` first.
See [CLI commands reference](../cli-commands.md).

## Bootstrap

Prerequisites:

- `curl`
- `jq`

```bash
export BASE_URL="http://localhost:8080"
export CLIENT_ID="<client-id>"
export CLIENT_SECRET="<client-secret>"
```

## 1) Get token

```bash
TOKEN=$(curl -s -X POST "$BASE_URL/v1/token" \
  -H "Content-Type: application/json" \
  -d "{\"client_id\":\"$CLIENT_ID\",\"client_secret\":\"$CLIENT_SECRET\"}" | jq -r .token)
```

## 1.1) Optional retry wrapper for `429`

```bash
request_with_retry() {
  local method="$1"
  local url="$2"
  local body="${3:-}"
  local attempt=0

  while [ "$attempt" -lt 5 ]; do
    attempt=$((attempt + 1))
    local headers_file
    headers_file=$(mktemp)

    local status
    if [ -n "$body" ]; then
      status=$(curl -s -o /tmp/resp.json -D "$headers_file" -w "%{http_code}" -X "$method" "$url" \
        -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d "$body")
    else
      status=$(curl -s -o /tmp/resp.json -D "$headers_file" -w "%{http_code}" -X "$method" "$url" \
        -H "Authorization: Bearer $TOKEN")
    fi

    if [ "$status" != "429" ]; then
      rm -f "$headers_file"
      cat /tmp/resp.json
      return 0
    fi

    local retry_after
    retry_after=$(awk 'tolower($1)=="retry-after:" {print $2}' "$headers_file" | tr -d '\r')
    rm -f "$headers_file"
    sleep "${retry_after:-1}"
  done

  echo "request failed after retries" >&2
  return 1
}
```

## 2) Write secret

```bash
curl -X POST "$BASE_URL/v1/secrets/app/prod/api-key" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value":"YXBpLWtleS12YWx1ZQ=="}'
```

## 3) Read secret

```bash
curl "$BASE_URL/v1/secrets/app/prod/api-key" \
  -H "Authorization: Bearer $TOKEN"
```

## 4) Transit encrypt + decrypt

For transit decrypt, pass `ciphertext` exactly as returned by encrypt (`<version>:<base64-ciphertext>`).

```bash
curl -X POST "$BASE_URL/v1/transit/keys" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"pii","algorithm":"aes-gcm"}'

CIPHERTEXT=$(curl -s -X POST "$BASE_URL/v1/transit/keys/pii/encrypt" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plaintext":"am9obkBleGFtcGxlLmNvbQ=="}' | jq -r .ciphertext)

PLAINTEXT_B64=$(curl -s -X POST "$BASE_URL/v1/transit/keys/pii/decrypt" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"ciphertext\":\"$CIPHERTEXT\"}" | jq -r .plaintext)

test "$PLAINTEXT_B64" = "am9obkBleGFtcGxlLmNvbQ==" && echo "Transit round-trip verified"

# Note: `plaintext` is base64. Decode it in your app/runtime before use.
```

### Transit Create Fallback (201 vs 409)

Use this pattern in automation: create once, rotate when key already exists.

```bash
CREATE_STATUS=$(curl -s -o /tmp/transit-create.json -w "%{http_code}" -X POST "$BASE_URL/v1/transit/keys" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"pii","algorithm":"aes-gcm"}')

if [ "$CREATE_STATUS" = "201" ]; then
  echo "Transit key created"
elif [ "$CREATE_STATUS" = "409" ]; then
  echo "Transit key already exists, rotating"
  curl -s -X POST "$BASE_URL/v1/transit/keys/pii/rotate" \
    -H "Authorization: Bearer $TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"algorithm":"aes-gcm"}' >/tmp/transit-rotate.json
else
  echo "Unexpected create status: $CREATE_STATUS"
  cat /tmp/transit-create.json
  exit 1
fi
```

## 5) List audit logs (cursor pagination)

```bash
# First page (no cursor)
curl "$BASE_URL/v1/audit-logs?limit=50" \
  -H "Authorization: Bearer $TOKEN"

# Response (abbreviated)
# {
#   "data": [
#     {
#       "id": "01936c8e-7f2a-7b3c-9d4e-5f6a7b8c9d0e",
#       "client_id": "01936c8e-1234-5678-9abc-def012345678",
#       "action": "secret.read",
#       "resource_type": "secret",
#       "resource_path": "/app/prod/api-key",
#       "created_at": "2026-03-03T10:30:00Z"
#     }
#   ],
#   "next_cursor": "01936c8e-7f2a-7b3c-9d4e-5f6a7b8c9d0e"
# }

# Subsequent page (with after_id cursor)
curl "$BASE_URL/v1/audit-logs?limit=50&after_id=01936c8e-7f2a-7b3c-9d4e-5f6a7b8c9d0e" \
  -H "Authorization: Bearer $TOKEN"

# When no more pages exist, next_cursor is omitted from response
```

## 5.1) List clients (cursor pagination)

```bash
# First page (no cursor)
curl "$BASE_URL/v1/clients?limit=50" \
  -H "Authorization: Bearer $TOKEN"

# Response (abbreviated)
# {
#   "data": [
#     {
#       "id": "01936c8e-1234-5678-9abc-def012345678",
#       "name": "production-app",
#       "description": "Main production application",
#       "is_active": true,
#       "created_at": "2026-03-01T08:00:00Z"
#     }
#   ],
#   "next_cursor": "01936c8e-1234-5678-9abc-def012345678"
# }

# Subsequent page (with after_id cursor)
curl "$BASE_URL/v1/clients?limit=50&after_id=01936c8e-1234-5678-9abc-def012345678" \
  -H "Authorization: Bearer $TOKEN"
```

## 5.2) List secrets (cursor pagination)

```bash
# First page (no cursor)
curl "$BASE_URL/v1/secrets?limit=50" \
  -H "Authorization: Bearer $TOKEN"

# Response (abbreviated)
# {
#   "data": [
#     {
#       "path": "/app/dev/database-url",
#       "version": 3,
#       "created_at": "2026-03-02T14:20:00Z",
#       "updated_at": "2026-03-03T09:15:00Z"
#     },
#     {
#       "path": "/app/prod/api-key",
#       "version": 1,
#       "created_at": "2026-03-03T10:00:00Z",
#       "updated_at": "2026-03-03T10:00:00Z"
#     }
#   ],
#   "next_cursor": "/app/prod/api-key"
# }

# Subsequent page (with after_path cursor)
curl "$BASE_URL/v1/secrets?limit=50&after_path=/app/prod/api-key" \
  -H "Authorization: Bearer $TOKEN"

# Note: Secrets use path-based cursor (after_path) instead of UUID cursor
```

## 6) Tokenization quick flow

```bash
# Create a tokenization key
curl -X POST "$BASE_URL/v1/tokenization/keys" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"payment-cards","format_type":"luhn-preserving","is_deterministic":true,"algorithm":"aes-gcm"}'

# Tokenize a value
TOKENIZED=$(curl -s -X POST "$BASE_URL/v1/tokenization/keys/payment-cards/tokenize" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plaintext":"NDUzMjAxNTExMjgzMDM2Ng==","metadata":{"last_four":"0366"},"ttl":3600}' | jq -r .token)

# Validate token
curl -X POST "$BASE_URL/v1/tokenization/validate" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"token\":\"$TOKENIZED\"}"

# Detokenize token
curl -X POST "$BASE_URL/v1/tokenization/detokenize" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"token\":\"$TOKENIZED\"}"
```

Deterministic caveat:

- When `is_deterministic=true`, the engine uses per-key version salts and HMAC-SHA256 to prevent rainbow table attacks.
- Identical plaintext tokenized with the same active key version will return the same token.
- Prefer non-deterministic mode unless you explicitly need equality matching.

## 6.1) List tokenization keys (cursor pagination)

```bash
# First page (no cursor)
curl "$BASE_URL/v1/tokenization/keys?limit=50" \
  -H "Authorization: Bearer $TOKEN"

# Response (abbreviated)
# {
#   "data": [
#     {
#       "name": "payment-cards",
#       "format_type": "luhn-preserving",
#       "is_deterministic": true,
#       "algorithm": "aes-gcm",
#       "current_version": 1,
#       "created_at": "2026-03-03T10:00:00Z"
#     },
#     {
#       "name": "ssn-tokens",
#       "format_type": "numeric",
#       "is_deterministic": false,
#       "algorithm": "aes-gcm",
#       "current_version": 2,
#       "created_at": "2026-03-02T15:30:00Z"
#     }
#   ],
#   "next_cursor": "ssn-tokens"
# }

# Subsequent page (with after_name cursor)
curl "$BASE_URL/v1/tokenization/keys?limit=50&after_name=ssn-tokens" \
  -H "Authorization: Bearer $TOKEN"

# Note: Tokenization keys use name-based cursor (after_name)
```

## 6.2) List transit keys (cursor pagination)

```bash
# First page (no cursor)
curl "$BASE_URL/v1/transit/keys?limit=50" \
  -H "Authorization: Bearer $TOKEN"

# Response (abbreviated)
# {
#   "data": [
#     {
#       "name": "encryption-key",
#       "algorithm": "aes-gcm",
#       "current_version": 1,
#       "min_decryption_version": 1,
#       "created_at": "2026-03-01T12:00:00Z"
#     },
#     {
#       "name": "pii",
#       "algorithm": "aes-gcm",
#       "current_version": 3,
#       "min_decryption_version": 1,
#       "created_at": "2026-03-02T08:30:00Z"
#     }
#   ],
#   "next_cursor": "pii"
# }

# Subsequent page (with after_name cursor)
curl "$BASE_URL/v1/transit/keys?limit=50&after_name=pii" \
  -H "Authorization: Bearer $TOKEN"

# Note: Transit keys use name-based cursor (after_name)
```

## Common Mistakes

- Sending raw plaintext instead of base64 in `value`/`plaintext`
- Building your own decrypt `ciphertext` instead of reusing encrypt response exactly
- Missing `Bearer` prefix in `Authorization` header
- Using create repeatedly for same transit key name instead of rotate after `409`
- Sending token in URL path for tokenization lifecycle endpoints (the API expects token in JSON body)
- Ignoring `429` and retrying immediately instead of honoring `Retry-After`
- Using offset-based pagination parameters (removed in v1.0.0 - use cursor-based pagination instead)

## See also

- [Authentication API](../auth/authentication.md)
- [Secrets API](../engines/secrets.md)
- [Transit API](../engines/transit.md)
- [Clients API](../auth/clients.md)
- [API rate limiting](../concepts/api-fundamentals.md#rate-limiting)
