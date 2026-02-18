#!/usr/bin/env bash
set -euo pipefail

# Smoke test for Secrets API v1.
# Prerequisites:
# - jq installed
# - server running and reachable
# - valid client credentials with needed capabilities

BASE_URL="${BASE_URL:-http://localhost:8080}"
CLIENT_ID="${CLIENT_ID:-}"
CLIENT_SECRET="${CLIENT_SECRET:-}"
SECRET_PATH="${SECRET_PATH:-/app/prod/smoke-test}"
TRANSIT_KEY_NAME="${TRANSIT_KEY_NAME:-smoke-test-key}"
TOKENIZATION_KEY_NAME="${TOKENIZATION_KEY_NAME:-smoke-test-tokenization-key}"

if [[ -z "$CLIENT_ID" || -z "$CLIENT_SECRET" ]]; then
  echo "CLIENT_ID and CLIENT_SECRET are required"
  echo "Example: CLIENT_ID=... CLIENT_SECRET=... bash docs/getting-started/smoke-test.sh"
  exit 1
fi

echo "[1/8] Health check"
curl -fsS "$BASE_URL/health" | jq .

echo "[2/8] Issue token"
TOKEN="$(curl -fsS -X POST "$BASE_URL/v1/token" \
  -H "Content-Type: application/json" \
  -d "{\"client_id\":\"$CLIENT_ID\",\"client_secret\":\"$CLIENT_SECRET\"}" | jq -r '.token')"

if [[ -z "$TOKEN" || "$TOKEN" == "null" ]]; then
  echo "Failed to obtain token"
  exit 1
fi

echo "[3/8] Write secret"
curl -fsS -X POST "$BASE_URL/v1/secrets$SECRET_PATH" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value":"c21va2UtdGVzdC12YWx1ZQ=="}' | jq .

echo "[4/8] Read secret"
curl -fsS "$BASE_URL/v1/secrets$SECRET_PATH" \
  -H "Authorization: Bearer $TOKEN" | jq .

echo "[5/8] Create transit key (ignores conflict)"
CREATE_STATUS="$(curl -sS -o /tmp/secrets_transit_create.json -w "%{http_code}" -X POST "$BASE_URL/v1/transit/keys" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$TRANSIT_KEY_NAME\",\"algorithm\":\"aes-gcm\"}")"

if [[ "$CREATE_STATUS" != "201" && "$CREATE_STATUS" != "409" ]]; then
  echo "Transit key creation failed (status=$CREATE_STATUS)"
  cat /tmp/secrets_transit_create.json
  exit 1
fi

echo "[6/8] Encrypt and decrypt with transit key"
CIPHERTEXT="$(curl -fsS -X POST "$BASE_URL/v1/transit/keys/$TRANSIT_KEY_NAME/encrypt" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plaintext":"c21va2UtdHJhbnNpdA=="}' | jq -r '.ciphertext')"

curl -fsS -X POST "$BASE_URL/v1/transit/keys/$TRANSIT_KEY_NAME/decrypt" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"ciphertext\":\"$CIPHERTEXT\"}" | jq .

echo "[7/8] Create tokenization key (ignores conflict)"
TOKENIZATION_CREATE_STATUS="$(curl -sS -o /tmp/secrets_tokenization_create.json -w "%{http_code}" -X POST "$BASE_URL/v1/tokenization/keys" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$TOKENIZATION_KEY_NAME\",\"format_type\":\"uuid\",\"is_deterministic\":false,\"algorithm\":\"aes-gcm\"}")"

if [[ "$TOKENIZATION_CREATE_STATUS" != "201" && "$TOKENIZATION_CREATE_STATUS" != "409" ]]; then
  echo "Tokenization key creation failed (status=$TOKENIZATION_CREATE_STATUS)"
  cat /tmp/secrets_tokenization_create.json
  exit 1
fi

echo "[8/8] Tokenization round-trip and revoke"
TOKEN_VALUE="$(curl -fsS -X POST "$BASE_URL/v1/tokenization/keys/$TOKENIZATION_KEY_NAME/tokenize" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plaintext":"c21va2UtdG9rZW5pemF0aW9u","ttl":300}' | jq -r '.token')"

if [[ -z "$TOKEN_VALUE" || "$TOKEN_VALUE" == "null" ]]; then
  echo "Failed to tokenize sample payload"
  exit 1
fi

curl -fsS -X POST "$BASE_URL/v1/tokenization/detokenize" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"token\":\"$TOKEN_VALUE\"}" | jq .

VALID_BEFORE="$(curl -fsS -X POST "$BASE_URL/v1/tokenization/validate" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"token\":\"$TOKEN_VALUE\"}" | jq -r '.valid')"

if [[ "$VALID_BEFORE" != "true" ]]; then
  echo "Token should be valid before revoke"
  exit 1
fi

REVOKE_STATUS="$(curl -sS -o /tmp/secrets_tokenization_revoke.json -w "%{http_code}" -X POST "$BASE_URL/v1/tokenization/revoke" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"token\":\"$TOKEN_VALUE\"}")"

if [[ "$REVOKE_STATUS" != "204" ]]; then
  echo "Token revoke failed (status=$REVOKE_STATUS)"
  cat /tmp/secrets_tokenization_revoke.json
  exit 1
fi

VALID_AFTER="$(curl -fsS -X POST "$BASE_URL/v1/tokenization/validate" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"token\":\"$TOKEN_VALUE\"}" | jq -r '.valid')"

if [[ "$VALID_AFTER" != "false" ]]; then
  echo "Token should be invalid after revoke"
  exit 1
fi

echo "Smoke test completed successfully"
