# 🟨 JavaScript Examples

> Last updated: 2026-02-25

⚠️ Security Warning: base64 is encoding, not encryption. Always use HTTPS/TLS.

## Bootstrap

Prerequisites:

- Node.js 20+
- runtime with global `fetch` support

Recommended environment variables:

```bash
export BASE_URL="http://localhost:8080"
export CLIENT_ID="<client-id>"
export CLIENT_SECRET="<client-secret>"
```

```javascript
const BASE_URL = process.env.BASE_URL || "http://localhost:8080";
const CLIENT_ID = process.env.CLIENT_ID || "<client-id>";
const CLIENT_SECRET = process.env.CLIENT_SECRET || "<client-secret>";

const toBase64 = (value) => Buffer.from(value, "utf8").toString("base64");

async function postWithRetry(path, token, body, maxAttempts = 5) {
  for (let attempt = 1; attempt <= maxAttempts; attempt += 1) {
    const response = await fetch(`${BASE_URL}${path}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(body),
    });

    if (response.status !== 429) {
      return response;
    }

    const retryAfter = Number(response.headers.get("Retry-After") || "1");
    const jitterMs = Math.floor(Math.random() * 500);
    await new Promise((resolve) => setTimeout(resolve, retryAfter * 1000 + jitterMs));
  }

  throw new Error("request failed after retry budget");
}

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

## Tokenization Quick Snippet

```javascript
async function tokenizationFlow(token) {
  await fetch(`${BASE_URL}/v1/tokenization/keys`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({
      name: "js-tokenization",
      format_type: "uuid",
      is_deterministic: false,
      algorithm: "aes-gcm",
    }),
  });

  const tokenizeRes = await fetch(`${BASE_URL}/v1/tokenization/keys/js-tokenization/tokenize`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ plaintext: toBase64("sensitive-value"), ttl: 600 }),
  });
  if (!tokenizeRes.ok) throw new Error(`tokenize failed: ${tokenizeRes.status}`);
  const { token: tokenValue } = await tokenizeRes.json();

  const detokenizeRes = await fetch(`${BASE_URL}/v1/tokenization/detokenize`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ token: tokenValue }),
  });
  if (!detokenizeRes.ok) throw new Error(`detokenize failed: ${detokenizeRes.status}`);
}
```

Deterministic caveat:

- With `is_deterministic: true`, tokenizing the same plaintext with the same active key can produce the same token.
- Prefer non-deterministic mode unless stable equality matching is required.

Rate-limit note:

- For protected endpoints, honor `Retry-After` on `429` with exponential/backoff + jitter

## Common Mistakes

- Sending UTF-8 plaintext directly instead of base64 in transit/secrets payloads
- Reformatting `ciphertext` for decrypt instead of passing encrypt response as-is
- Missing `Authorization: Bearer <token>` header on protected endpoints
- Reusing transit create for existing keys without fallback to rotate on `409`
- Sending tokenization token in URL path instead of JSON body for `detokenize`, `validate`, and `revoke`
- Retrying immediately after `429` without delay/jitter

## See also

- [Authentication API](../api/auth/authentication.md)
- [Secrets API](../api/data/secrets.md)
- [Transit API](../api/data/transit.md)
- [Tokenization API](../api/data/tokenization.md)
- [Response shapes](../api/observability/response-shapes.md)
- [API rate limiting](../api/fundamentals.md#rate-limiting)
