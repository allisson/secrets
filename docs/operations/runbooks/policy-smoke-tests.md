# 🧪 Policy Smoke Tests

> Last updated: 2026-02-25

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

Transit rotate check (mid-path wildcard, `rotate` required):

```bash
ALLOW_STATUS=$(curl -s -o /tmp/allow-rotate.json -w "%{http_code}" -X POST \
  "$BASE_URL/v1/transit/keys/payment/rotate" \
  -H "Authorization: Bearer $ALLOW_TOKEN")

DENY_STATUS=$(curl -s -o /tmp/deny-rotate.json -w "%{http_code}" -X POST \
  "$BASE_URL/v1/transit/keys/payment/rotate" \
  -H "Authorization: Bearer $DENY_TOKEN")

echo "allowed status=$ALLOW_STATUS denied status=$DENY_STATUS"
test "$ALLOW_STATUS" = "200"
test "$DENY_STATUS" = "403"
```

Tip: this check validates policies like `/v1/transit/keys/*/rotate` and catches wildcard path drift.

Malformed path shape check (extra segment should not match rotate route):

```bash
BAD_SHAPE_STATUS=$(curl -s -o /tmp/bad-shape-rotate.json -w "%{http_code}" -X POST \
  "$BASE_URL/v1/transit/keys/payment/extra/rotate" \
  -H "Authorization: Bearer $ALLOW_TOKEN")

echo "bad shape status=$BAD_SHAPE_STATUS"
test "$BAD_SHAPE_STATUS" = "404"
```

Tip: this validates caller path shape expectations; use the allow/deny rotate checks above to validate
capability enforcement.
See [Route shape vs policy shape](../../api/auth/policies.md#route-shape-vs-policy-shape) for triage guidance.

Secrets malformed path-shape check (missing wildcard subpath should not match):

```bash
BAD_SECRET_SHAPE_STATUS=$(curl -s -o /tmp/bad-shape-secret.json -w "%{http_code}" \
  "$BASE_URL/v1/secrets" \
  -H "Authorization: Bearer $ALLOW_TOKEN")

echo "bad secret shape status=$BAD_SECRET_SHAPE_STATUS"
test "$BAD_SECRET_SHAPE_STATUS" = "404"
```

Tip: use this check to ensure policy path logic is not confused with route-template shape.
See [Route shape vs policy shape](../../api/auth/policies.md#route-shape-vs-policy-shape) for details.

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

Pre-deploy automation pattern:

1. Run static policy lint checks (JSON shape, wildcard rules, capability allow-list)
2. Deploy policy to staging
3. Run allow/deny smoke assertions from this page
4. Block production rollout on first mismatch

Optional strict CI mode:

```bash
set -euo pipefail

# Run the checks in this document and fail fast on first mismatch
# (each `test` command exits non-zero on failure)

echo "policy smoke checks: PASS"
```

GitHub Actions example:

```yaml
- name: Policy smoke checks
  env:
    BASE_URL: ${{ vars.SECRETS_BASE_URL }}
    ALLOW_CLIENT_ID: ${{ secrets.POLICY_ALLOW_CLIENT_ID }}
    ALLOW_CLIENT_SECRET: ${{ secrets.POLICY_ALLOW_CLIENT_SECRET }}
    DENY_CLIENT_ID: ${{ secrets.POLICY_DENY_CLIENT_ID }}
    DENY_CLIENT_SECRET: ${{ secrets.POLICY_DENY_CLIENT_SECRET }}
  run: |
    set -euo pipefail
    # Run commands from this page and fail on first mismatch.
```

Scripted wrapper example:

```bash
#!/usr/bin/env bash
set -euo pipefail

echo "[1/3] issuing allow/deny tokens"
# insert token issuance block from this page

echo "[2/3] running allow/deny assertions"
# insert capability checks from this page

echo "[3/3] verifying denied audit events"
# insert audit verification block from this page

echo "policy smoke suite: PASS"
```

## See also

- [Capability matrix](../../api/fundamentals.md#capability-matrix)
- [Policies cookbook](../../api/auth/policies.md)
- [Incident response guide](../observability/incident-response.md)
- [Troubleshooting](../../operations/troubleshooting/index.md)
