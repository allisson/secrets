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

if [[ -z "$CLIENT_ID" || -z "$CLIENT_SECRET" ]]; then
  echo "CLIENT_ID and CLIENT_SECRET are required"
  echo "Example: CLIENT_ID=... CLIENT_SECRET=... bash docs/getting-started/smoke-test.sh"
  exit 1
fi

echo "[1/6] Health check"
curl -fsS "$BASE_URL/health" | jq .

echo "[2/6] Issue token"
TOKEN="$(curl -fsS -X POST "$BASE_URL/v1/token" \
  -H "Content-Type: application/json" \
  -d "{\"client_id\":\"$CLIENT_ID\",\"client_secret\":\"$CLIENT_SECRET\"}" | jq -r '.token')"

if [[ -z "$TOKEN" || "$TOKEN" == "null" ]]; then
  echo "Failed to obtain token"
  exit 1
fi

echo "[3/6] Write secret"
curl -fsS -X POST "$BASE_URL/v1/secrets$SECRET_PATH" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value":"c21va2UtdGVzdC12YWx1ZQ=="}' | jq .

echo "[4/6] Read secret"
curl -fsS "$BASE_URL/v1/secrets$SECRET_PATH" \
  -H "Authorization: Bearer $TOKEN" | jq .

echo "[5/6] Create transit key (ignores conflict)"
CREATE_STATUS="$(curl -sS -o /tmp/secrets_transit_create.json -w "%{http_code}" -X POST "$BASE_URL/v1/transit/keys" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$TRANSIT_KEY_NAME\",\"algorithm\":\"aes-gcm\"}")"

if [[ "$CREATE_STATUS" != "201" && "$CREATE_STATUS" != "409" ]]; then
  echo "Transit key creation failed (status=$CREATE_STATUS)"
  cat /tmp/secrets_transit_create.json
  exit 1
fi

echo "[6/6] Encrypt and decrypt with transit key"
CIPHERTEXT="$(curl -fsS -X POST "$BASE_URL/v1/transit/keys/$TRANSIT_KEY_NAME/encrypt" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plaintext":"c21va2UtdHJhbnNpdA=="}' | jq -r '.ciphertext')"

curl -fsS -X POST "$BASE_URL/v1/transit/keys/$TRANSIT_KEY_NAME/decrypt" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"ciphertext\":\"$CIPHERTEXT\"}" | jq .

echo "Smoke test completed successfully"
