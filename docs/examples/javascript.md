# 🟨 JavaScript Examples

> Last updated: 2026-02-14

⚠️ Security Warning: base64 is encoding, not encryption. Always use HTTPS/TLS.

```javascript
const BASE_URL = "http://localhost:8080";
const CLIENT_ID = "<client-id>";
const CLIENT_SECRET = "<client-secret>";

const toBase64 = (value) => Buffer.from(value, "utf8").toString("base64");

async function issueToken() {
  const response = await fetch(`${BASE_URL}/v1/token`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ client_id: CLIENT_ID, client_secret: CLIENT_SECRET }),
  });
  if (!response.ok) throw new Error(`token failed: ${response.status}`);
  const data = await response.json();
  return data.token;
}

async function createSecret(token) {
  const response = await fetch(`${BASE_URL}/v1/secrets/app/prod/js-example`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ value: toBase64("javascript-secret-value") }),
  });
  if (!response.ok) throw new Error(`create secret failed: ${response.status}`);
}

async function transitFlow(token) {
  await fetch(`${BASE_URL}/v1/transit/keys`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ name: "js-pii", algorithm: "aes-gcm" }),
  });

  const encryptRes = await fetch(`${BASE_URL}/v1/transit/keys/js-pii/encrypt`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ plaintext: toBase64("john@example.com") }),
  });
  if (!encryptRes.ok) throw new Error(`encrypt failed: ${encryptRes.status}`);
  // For transit decrypt, pass ciphertext exactly as returned by encrypt: "<version>:<base64-ciphertext>".
  const { ciphertext } = await encryptRes.json();

  const decryptRes = await fetch(`${BASE_URL}/v1/transit/keys/js-pii/decrypt`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ ciphertext }),
  });
  if (!decryptRes.ok) throw new Error(`decrypt failed: ${decryptRes.status}`);
  const decrypted = await decryptRes.json();
  if (decrypted.plaintext !== toBase64("john@example.com")) {
    throw new Error("round-trip verification failed");
  }
  const decoded = Buffer.from(decrypted.plaintext, "base64").toString("utf8");
  console.log("decrypted value:", decoded);
  console.log("Transit round-trip verified");
}

async function main() {
  const token = await issueToken();
  await createSecret(token);
  await transitFlow(token);
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
```

## See also

- [Authentication API](../api/authentication.md)
- [Secrets API](../api/secrets.md)
- [Transit API](../api/transit.md)
- [Response shapes](../api/response-shapes.md)
