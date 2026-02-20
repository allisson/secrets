#!/usr/bin/env python3

import json
import re
from pathlib import Path


def extract_json_block(content: str, label: str) -> dict:
    pattern = re.compile(re.escape(label) + r"\n\n```json\n(.*?)\n```", re.DOTALL)
    match = pattern.search(content)
    if not match:
        raise ValueError(f"missing JSON block for label: {label}")
    return json.loads(match.group(1))


def require_keys(payload: dict, keys: list[str], label: str) -> None:
    missing = [key for key in keys if key not in payload]
    if missing:
        raise ValueError(f"missing keys for {label}: {', '.join(missing)}")


def main() -> None:
    response_shapes = Path("docs/api/observability/response-shapes.md").read_text(
        encoding="utf-8"
    )
    transit_api = Path("docs/api/data/transit.md").read_text(encoding="utf-8")

    token = extract_json_block(response_shapes, "Token issuance:")
    require_keys(token, ["token", "expires_at"], "Token issuance")

    client_creation = extract_json_block(response_shapes, "Client creation:")
    require_keys(client_creation, ["id", "secret"], "Client creation")

    secret_write = extract_json_block(response_shapes, "Secret write:")
    require_keys(secret_write, ["id", "path", "version", "created_at"], "Secret write")

    secret_read = extract_json_block(response_shapes, "Secret read:")
    require_keys(
        secret_read, ["id", "path", "version", "value", "created_at"], "Secret read"
    )

    transit_encrypt = extract_json_block(response_shapes, "Transit encrypt:")
    require_keys(transit_encrypt, ["ciphertext", "version"], "Transit encrypt")

    transit_decrypt = extract_json_block(response_shapes, "Transit decrypt:")
    require_keys(transit_decrypt, ["plaintext"], "Transit decrypt")

    conflict = extract_json_block(transit_api, "`409 Conflict`")
    require_keys(conflict, ["error", "message"], "Transit conflict")
    if conflict["error"] != "conflict":
        raise ValueError("Transit conflict payload must use error=conflict")

    print("example shape checks passed")


if __name__ == "__main__":
    main()
