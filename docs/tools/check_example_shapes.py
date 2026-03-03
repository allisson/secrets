#!/usr/bin/env python3

import json
import re
from pathlib import Path


def extract_json_block(content: str, label: str) -> dict:
    pattern = re.compile(re.escape(label) + r"\s*```json\n(.*?)\n```", re.DOTALL)
    match = pattern.search(content)
    if not match:
        raise ValueError(f"missing JSON block for label: {label}")
    return json.loads(match.group(1))


def require_keys(payload: dict, keys: list[str], label: str) -> None:
    missing = [key for key in keys if key not in payload]
    if missing:
        raise ValueError(f"missing keys for {label}: {', '.join(missing)}")


def main() -> None:
    auth_api = Path("docs/auth/authentication.md").read_text(encoding="utf-8")
    clients_api = Path("docs/auth/clients.md").read_text(encoding="utf-8")
    secrets_api = Path("docs/engines/secrets.md").read_text(encoding="utf-8")
    transit_api = Path("docs/engines/transit.md").read_text(encoding="utf-8")
    tokenization_api = Path("docs/engines/tokenization.md").read_text(encoding="utf-8")
    audit_api = Path("docs/observability/audit-logs.md").read_text(encoding="utf-8")

    token = extract_json_block(auth_api, "Response (`201 Created`):")
    require_keys(token, ["token", "expires_at"], "Token issuance")

    client_creation = extract_json_block(clients_api, "Example response (`201 Created`):")
    require_keys(client_creation, ["id", "secret"], "Client creation")

    secret_write = extract_json_block(secrets_api, "Example response (`201 Created`):")
    require_keys(secret_write, ["id", "path", "version", "created_at"], "Secret write")

    secret_read = extract_json_block(secrets_api, "Example response (`200 OK`):")
    require_keys(
        secret_read, ["id", "path", "version", "value", "created_at"], "Secret read"
    )

    transit_encrypt = extract_json_block(transit_api, "Example encrypt response (`200 OK`):")
    require_keys(transit_encrypt, ["ciphertext", "version"], "Transit encrypt")

    transit_decrypt = extract_json_block(transit_api, "Example decrypt response (`200 OK`):")
    require_keys(transit_decrypt, ["plaintext", "version"], "Transit decrypt")

    tokenize_res = extract_json_block(tokenization_api, "Example response (`201 Created`):")
    require_keys(tokenize_res, ["token", "created_at", "expires_at"], "Tokenize")

    detokenize_res = extract_json_block(tokenization_api, "Example response (`200 OK`):")
    require_keys(detokenize_res, ["plaintext"], "Detokenize")

    print("example shape checks passed")


if __name__ == "__main__":
    main()
