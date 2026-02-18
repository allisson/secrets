# 🧪 Policy Smoke Tests

> Last updated: 2026-02-18

Use this page to quickly validate authorization behavior after policy changes.

## Why this exists

- Catch capability drift before production rollout
- Prove least-privilege policies actually enforce intended boundaries
- Provide repeatable checks for CI/CD or release validation

## Prerequisites

- Running Secrets API
- `curl` and `jq`
- One client expected to be allowed and one expected to be denied for the target path

```bash
export BASE_URL="http://localhost:8080"
export ALLOW_CLIENT_ID="<allowed-client-id>"
export ALLOW_CLIENT_SECRET="<allowed-client-secret>"
export DENY_CLIENT_ID="<denied-client-id>"
export DENY_CLIENT_SECRET="<denied-client-secret>"
```

## 1) Issue test tokens

```bash
ALLOW_TOKEN=$(curl -s -X POST "$BASE_URL/v1/token" \
  -H "Content-Type: application/json" \
  -d "{\"client_id\":\"$ALLOW_CLIENT_ID\",\"client_secret\":\"$ALLOW_CLIENT_SECRET\"}" | jq -r .token)

DENY_TOKEN=$(curl -s -X POST "$BASE_URL/v1/token" \
  -H "Content-Type: application/json" \
  -d "{\"client_id\":\"$DENY_CLIENT_ID\",\"client_secret\":\"$DENY_CLIENT_SECRET\"}" | jq -r .token)
```

## 2) Capability checks

Secrets read check (`decrypt` required):

```bash
ALLOW_STATUS=$(curl -s -o /tmp/allow-read.json -w "%{http_code}" \
  "$BASE_URL/v1/secrets/app/prod/smoke-policy" \
  -H "Authorization: Bearer $ALLOW_TOKEN")

DENY_STATUS=$(curl -s -o /tmp/deny-read.json -w "%{http_code}" \
  "$BASE_URL/v1/secrets/app/prod/smoke-policy" \
  -H "Authorization: Bearer $DENY_TOKEN")

echo "allowed status=$ALLOW_STATUS denied status=$DENY_STATUS"
test "$ALLOW_STATUS" = "200"
test "$DENY_STATUS" = "403"
```

Transit encrypt check (`encrypt` required):

```bash
ALLOW_STATUS=$(curl -s -o /tmp/allow-transit.json -w "%{http_code}" -X POST \
  "$BASE_URL/v1/transit/keys/payment/encrypt" \
  -H "Authorization: Bearer $ALLOW_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plaintext":"c21va2UtcG9saWN5"}')

DENY_STATUS=$(curl -s -o /tmp/deny-transit.json -w "%{http_code}" -X POST \
  "$BASE_URL/v1/transit/keys/payment/encrypt" \
  -H "Authorization: Bearer $DENY_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plaintext":"c21va2UtcG9saWN5"}')

echo "allowed status=$ALLOW_STATUS denied status=$DENY_STATUS"
test "$ALLOW_STATUS" = "200"
test "$DENY_STATUS" = "403"
```

Tokenization detokenize check (`decrypt` required):

```bash
ALLOW_STATUS=$(curl -s -o /tmp/allow-detokenize.json -w "%{http_code}" -X POST \
  "$BASE_URL/v1/tokenization/detokenize" \
  -H "Authorization: Bearer $ALLOW_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"token":"tok_sample"}')

DENY_STATUS=$(curl -s -o /tmp/deny-detokenize.json -w "%{http_code}" -X POST \
  "$BASE_URL/v1/tokenization/detokenize" \
  -H "Authorization: Bearer $DENY_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"token":"tok_sample"}')

echo "allowed status=$ALLOW_STATUS denied status=$DENY_STATUS"
# allowed may be 200 or 404 depending on sample token existence
test "$ALLOW_STATUS" = "200" -o "$ALLOW_STATUS" = "404"
test "$DENY_STATUS" = "403"
```

## 3) Audit verification

```bash
curl -s "$BASE_URL/v1/audit-logs?limit=100" \
  -H "Authorization: Bearer $ALLOW_TOKEN" \
  | jq '.audit_logs[] | select(.metadata.allowed == false) | {path, capability, client_id, created_at}'
```

Expected:

- denied requests appear with `metadata.allowed=false`
- denied client has expected path/capability mismatches only

## CI-friendly pattern

- Keep smoke checks idempotent
- Assert expected status pairs (allow vs deny)
- Run after policy deployment but before traffic cutover

## See also

- [Capability matrix](../api/capability-matrix.md)
- [Policies cookbook](../api/policies.md)
- [Failure playbooks](failure-playbooks.md)
- [Troubleshooting](../getting-started/troubleshooting.md)
