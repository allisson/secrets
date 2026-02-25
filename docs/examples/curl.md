# 🧪 Curl Examples

> Last updated: 2026-02-25

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

## 5) Audit logs query

```bash
curl "$BASE_URL/v1/audit-logs?limit=50&offset=0" \
  -H "Authorization: Bearer $TOKEN"
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

- When `is_deterministic=true`, tokenizing the same plaintext with the same active key can return the same token
- Prefer non-deterministic mode unless you explicitly need equality matching

## Common Mistakes

- Sending raw plaintext instead of base64 in `value`/`plaintext`
- Building your own decrypt `ciphertext` instead of reusing encrypt response exactly
- Missing `Bearer` prefix in `Authorization` header
- Using create repeatedly for same transit key name instead of rotate after `409`
- Sending token in URL path for tokenization lifecycle endpoints (the API expects token in JSON body)
- Ignoring `429` and retrying immediately instead of honoring `Retry-After`

## See also

- [Authentication API](../api/auth/authentication.md)
- [Secrets API](../api/data/secrets.md)
- [Transit API](../api/data/transit.md)
- [Clients API](../api/auth/clients.md)
- [API rate limiting](../api/fundamentals.md#rate-limiting)
