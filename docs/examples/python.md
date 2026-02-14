# 🐍 Python Examples

> Last updated: 2026-02-14

⚠️ Security Warning: base64 is encoding, not encryption. Always use HTTPS/TLS.

```python
import base64
import requests

BASE_URL = "http://localhost:8080"
CLIENT_ID = "<client-id>"
CLIENT_SECRET = "<client-secret>"


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
    ciphertext = encrypted.json()["ciphertext"]

    decrypted = requests.post(
        f"{BASE_URL}/v1/transit/keys/python-pii/decrypt",
        headers=headers,
        json={"ciphertext": ciphertext},
        timeout=10,
    )
    decrypted.raise_for_status()
    print("decrypted payload:", decrypted.json())


if __name__ == "__main__":
    token = issue_token()
    create_secret(token)
    transit_encrypt_decrypt(token)
```
