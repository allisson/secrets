# ⚙️ Environment Variables

> Last updated: 2026-02-14

Secrets is configured through environment variables.

## Core configuration

```dotenv
DB_DRIVER=postgres
DB_CONNECTION_STRING=postgres://user:password@localhost:5432/mydb?sslmode=disable
DB_MAX_OPEN_CONNECTIONS=25
DB_MAX_IDLE_CONNECTIONS=5
DB_CONN_MAX_LIFETIME=5

SERVER_HOST=0.0.0.0
SERVER_PORT=8080
LOG_LEVEL=info

MASTER_KEYS=default:BASE64_32_BYTE_KEY
ACTIVE_MASTER_KEY_ID=default

AUTH_TOKEN_EXPIRATION_SECONDS=86400
```

## Notes

- 🔐 `MASTER_KEYS` format is `id1:base64key1,id2:base64key2`
- 📏 Each master key must represent exactly 32 bytes (256 bits)
- ⭐ `ACTIVE_MASTER_KEY_ID` selects which master key encrypts new KEKs
- ⏱️ `AUTH_TOKEN_EXPIRATION_SECONDS` defaults to 24h behavior when set to `86400`
- 🔄 After changing `MASTER_KEYS` or `ACTIVE_MASTER_KEY_ID`, restart API servers to load new values

## Master key generation

```bash
./bin/app create-master-key --id default
```

Or with Docker image:

```bash
docker run --rm allisson/secrets:latest create-master-key --id default
```
