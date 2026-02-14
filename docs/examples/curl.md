# 🧪 Curl Examples

> Last updated: 2026-02-14

⚠️ Security Warning: base64 is encoding, not encryption. Always use HTTPS/TLS.

End-to-end shell workflow.

## 1) Get token

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/v1/token \
  -H "Content-Type: application/json" \
  -d '{"client_id":"<client-id>","client_secret":"<client-secret>"}' | jq -r .token)
```

## 2) Write secret

```bash
curl -X POST http://localhost:8080/v1/secrets/app/prod/api-key \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value":"YXBpLWtleS12YWx1ZQ=="}'
```

## 3) Read secret

```bash
curl http://localhost:8080/v1/secrets/app/prod/api-key \
  -H "Authorization: Bearer $TOKEN"
```

## 4) Transit encrypt + decrypt

```bash
curl -X POST http://localhost:8080/v1/transit/keys \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"pii","algorithm":"aes-gcm"}'

CIPHERTEXT=$(curl -s -X POST http://localhost:8080/v1/transit/keys/pii/encrypt \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"plaintext":"am9obkBleGFtcGxlLmNvbQ=="}' | jq -r .ciphertext)

curl -X POST http://localhost:8080/v1/transit/keys/pii/decrypt \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"ciphertext\":\"$CIPHERTEXT\"}"
```

## 5) Audit logs query

```bash
curl "http://localhost:8080/v1/audit-logs?limit=50&offset=0" \
  -H "Authorization: Bearer $TOKEN"
```

## See also

- [Authentication API](../api/authentication.md)
- [Secrets API](../api/secrets.md)
- [Transit API](../api/transit.md)
- [Clients API](../api/clients.md)
