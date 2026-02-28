# ✅ Smoke Test Script

> Last updated: 2026-02-28

Run a fast end-to-end validation of a running Secrets instance.

Script path: `docs/getting-started/smoke-test.sh`

## What it validates

1. `GET /health`
2. `POST /v1/token`
3. `POST /v1/secrets/*path`
4. `GET /v1/secrets/*path`
5. `POST /v1/transit/keys`
6. `POST /v1/transit/keys/:name/encrypt` and `/decrypt`
7. `POST /v1/tokenization/keys`
8. `POST /v1/tokenization/keys/:name/tokenize` + `POST /v1/tokenization/detokenize` + `POST /v1/tokenization/validate` + `POST /v1/tokenization/revoke`

For transit decrypt, pass `ciphertext` exactly as returned by encrypt (`<version>:<base64-ciphertext>`).

## Prerequisites

- Running Secrets server
- `jq` installed
- Client credentials with required capabilities

## Shell Compatibility

- Script requires `bash` (not `sh`)
- Script uses `jq` for JSON parsing
- Script is tested with GNU/Linux and macOS shells

## Usage

> Command status: verified on 2026-02-27

```bash
CLIENT_ID="<client-id>" \
CLIENT_SECRET="<client-secret>" \
BASE_URL="http://localhost:8080" \
bash docs/getting-started/smoke-test.sh
```

Optional variables:

- `SECRET_PATH` (default: `/app/prod/smoke-test`)
- `TRANSIT_KEY_NAME` (default: `smoke-test-key`)
- `TOKENIZATION_KEY_NAME` (default: `smoke-test-tokenization-key`)

Expected output includes `Smoke test completed successfully`.
If transit decrypt fails with `422`, see [Troubleshooting](../operations/troubleshooting/index.md).

## Optional: Token Throttling Verification

> Command status: verified on 2026-02-27

Use this only in non-production environments to verify token endpoint `429` behavior:

```bash
# 1) Issue one token normally (should return 201)
curl -i -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{"client_id":"<client-id>","client_secret":"<client-secret>"}'

# 2) Burst requests to trigger throttling in strict configs
for i in $(seq 1 20); do
  curl -s -o /dev/null -w "%{http_code}\n" -X POST http://localhost:8080/v1/token \
    -H "Content-Type: application/json" \
    -d '{"client_id":"<client-id>","client_secret":"<client-secret>"}'
done
```

Expected result under throttling:

- Some responses return `429 Too Many Requests`
- Response includes `Retry-After` header

⚠️ Security Warning: base64 is encoding, not encryption. Always use HTTPS/TLS.

## See also

- [Docker getting started](docker.md)
- [Local development](local-development.md)
- [Troubleshooting](../operations/troubleshooting/index.md)
- [Release notes](../releases/RELEASES.md)
- [Curl examples](../examples/curl.md)
