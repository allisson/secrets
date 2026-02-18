#!/usr/bin/env python3

import json
import re
from pathlib import Path


def require_contains(path: Path, needle: str) -> None:
    content = path.read_text(encoding="utf-8")
    if needle not in content:
        raise ValueError(f"{path} missing required text: {needle}")


def main() -> None:
    metadata_path = Path("docs/metadata.json")
    metadata = json.loads(metadata_path.read_text(encoding="utf-8"))

    current_release = metadata["current_release"]
    api_version = metadata["api_version"]

    require_contains(Path("README.md"), current_release)
    require_contains(Path("docs/README.md"), current_release)

    openapi = Path("docs/openapi.yaml").read_text(encoding="utf-8")
    if f"version: {api_version}" not in openapi:
        raise ValueError(
            "docs/openapi.yaml version does not match docs/metadata.json api_version"
        )

    api_pages = sorted(Path("docs/api").glob("*.md"))
    missing = []
    marker = f"> Applies to: API {api_version}"
    for page in api_pages:
        content = page.read_text(encoding="utf-8")
        if marker not in content:
            missing.append(str(page))

    if missing:
        raise ValueError("API pages missing applies-to marker: " + ", ".join(missing))

    # Ensure docs index points to metadata source.
    require_contains(Path("docs/README.md"), "docs/metadata.json")

    # Soft check: release notes for current release should exist.
    release_file = Path(f"docs/releases/{current_release}.md")
    if not release_file.exists():
        raise ValueError(f"Missing release notes file: {release_file}")

    # Keep date shape simple for maintainers.
    if not re.match(r"^\d{4}-\d{2}-\d{2}$", metadata["last_docs_refresh"]):
        raise ValueError("docs/metadata.json last_docs_refresh must be YYYY-MM-DD")

    print("docs metadata checks passed")


if __name__ == "__main__":
    main()
