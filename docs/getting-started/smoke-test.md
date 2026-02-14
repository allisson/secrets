# ✅ Smoke Test Script

> Last updated: 2026-02-14

Run a fast end-to-end validation of a running Secrets instance.

Script path: `docs/getting-started/smoke-test.sh`

## What it validates

1. `GET /health`
2. `POST /v1/token`
3. `POST /v1/secrets/*path`
4. `GET /v1/secrets/*path`
5. `POST /v1/transit/keys`
6. `POST /v1/transit/keys/:name/encrypt` and `/decrypt`

## Prerequisites

- Running Secrets server
- `jq` installed
- Client credentials with required capabilities

## Shell Compatibility

- Script requires `bash` (not `sh`)
- Script uses `jq` for JSON parsing
- Script is tested with GNU/Linux and macOS shells

## Usage

```bash
CLIENT_ID="<client-id>" \
CLIENT_SECRET="<client-secret>" \
BASE_URL="http://localhost:8080" \
bash docs/getting-started/smoke-test.sh
```

Optional variables:

- `SECRET_PATH` (default: `/app/prod/smoke-test`)
- `TRANSIT_KEY_NAME` (default: `smoke-test-key`)

⚠️ Security Warning: base64 is encoding, not encryption. Always use HTTPS/TLS.

## See also

- [Docker getting started](docker.md)
- [Local development](local-development.md)
- [Troubleshooting](troubleshooting.md)
- [Curl examples](../examples/curl.md)
