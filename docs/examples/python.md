# 🐍 Python Examples

> Last updated: 2026-02-19

⚠️ Security Warning: base64 is encoding, not encryption. Always use HTTPS/TLS.

## Bootstrap

Prerequisites:

- Python 3.10+
- `requests` library (`pip install requests`)

Recommended environment variables:

```bash
export BASE_URL="http://localhost:8080"
export CLIENT_ID="<client-id>"
export CLIENT_SECRET="<client-secret>"
```

```python
import base64
import os
import requests

BASE_URL = os.getenv("BASE_URL", "http://localhost:8080")
CLIENT_ID = os.getenv("CLIENT_ID", "<client-id>")
CLIENT_SECRET = os.getenv("CLIENT_SECRET", "<client-secret>")


def b64(value: str) -> str:
    return base64.b64encode(value.encode("utf-8")).decode("utf-8")


def issue_token() -> str:
    response = requests.post(
        f"{BASE_URL}/v1/token",
        json={"client_id": CLIENT_ID, "client_secret": CLIENT_SECRET},
        timeout=10,
    )
    response.raise_for_status()
    return response.json()["token"]


def create_secret(token: str) -> None:
    headers = {"Authorization": f"Bearer {token}"}
    response = requests.post(
        f"{BASE_URL}/v1/secrets/app/prod/python-example",
        headers=headers,
        json={"value": b64("python-secret-value")},
        timeout=10,
    )
    response.raise_for_status()


def transit_encrypt_decrypt(token: str) -> None:
    headers = {"Authorization": f"Bearer {token}"}

    requests.post(
        f"{BASE_URL}/v1/transit/keys",
        headers=headers,
        json={"name": "python-pii", "algorithm": "aes-gcm"},
        timeout=10,
    )

    encrypted = requests.post(
        f"{BASE_URL}/v1/transit/keys/python-pii/encrypt",
        headers=headers,
        json={"plaintext": b64("john@example.com")},
        timeout=10,
    )
    encrypted.raise_for_status()
    # For transit decrypt, pass ciphertext exactly as returned by encrypt: "<version>:<base64-ciphertext>".
    ciphertext = encrypted.json()["ciphertext"]

    decrypted = requests.post(
        f"{BASE_URL}/v1/transit/keys/python-pii/decrypt",
        headers=headers,
        json={"ciphertext": ciphertext},
        timeout=10,
    )
    decrypted.raise_for_status()
    plaintext_b64 = decrypted.json()["plaintext"]
    if plaintext_b64 != b64("john@example.com"):
        raise RuntimeError("round-trip verification failed")
    plaintext = base64.b64decode(plaintext_b64).decode("utf-8")
    print("decrypted value:", plaintext)
    print("Transit round-trip verified")


if __name__ == "__main__":
    token = issue_token()
    create_secret(token)
    transit_encrypt_decrypt(token)
```

## Tokenization Quick Snippet

```python
def tokenize_detokenize(token: str) -> None:
    headers = {"Authorization": f"Bearer {token}"}

    requests.post(
        f"{BASE_URL}/v1/tokenization/keys",
        headers=headers,
        json={
            "name": "python-tokenization",
            "format_type": "uuid",
            "is_deterministic": False,
            "algorithm": "aes-gcm",
        },
        timeout=10,
    )

    tokenize = requests.post(
        f"{BASE_URL}/v1/tokenization/keys/python-tokenization/tokenize",
        headers=headers,
        json={"plaintext": b64("sensitive-value"), "ttl": 600},
        timeout=10,
    )
    tokenize.raise_for_status()
    token_value = tokenize.json()["token"]

    detokenize = requests.post(
        f"{BASE_URL}/v1/tokenization/detokenize",
        headers=headers,
        json={"token": token_value},
        timeout=10,
    )
    detokenize.raise_for_status()
    assert detokenize.json()["plaintext"] == b64("sensitive-value")
```

Deterministic caveat:

- If you create a key with `is_deterministic=True`, repeated tokenization of identical plaintext can return the same token.
- Use deterministic mode only when equality matching is a functional requirement.

## Common Mistakes

- Passing raw plaintext instead of base64-encoded `value`/`plaintext`
- Constructing decrypt `ciphertext` manually instead of using encrypt output
- Forgetting `Bearer` prefix in `Authorization` header
- Retrying transit create for an existing key name instead of handling `409` with rotate
- Sending tokenization token in URL path instead of JSON body for `detokenize`, `validate`, and `revoke`

## See also

- [Authentication API](../api/authentication.md)
- [Secrets API](../api/secrets.md)
- [Transit API](../api/transit.md)
- [Tokenization API](../api/tokenization.md)
- [Response shapes](../api/response-shapes.md)
