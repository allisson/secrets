#!/usr/bin/env python3

import json
import os
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

    # Basic freshness hygiene: every docs Markdown page must declare a Last updated marker.
    # This helps reduce metadata drift across the documentation set.
    markdown_pages = sorted(Path("docs").glob("**/*.md"))
    missing_last_updated = []
    invalid_last_updated = []

    for page in markdown_pages:
        # ADRs and release notes use their own date headers.
        if page.parts[:2] in [("docs", "adr"), ("docs", "releases")]:
            continue
        content = page.read_text(encoding="utf-8")
        match = re.search(r"^> Last updated: (.+)$", content, re.MULTILINE)
        if match is None:
            missing_last_updated.append(str(page))
            continue
        date_text = match.group(1).strip()
        if not re.match(r"^\d{4}-\d{2}-\d{2}$", date_text):
            invalid_last_updated.append(f"{page} ({date_text})")

    if missing_last_updated:
        raise ValueError(
            "Docs pages missing '> Last updated: YYYY-MM-DD' marker: "
            + ", ".join(missing_last_updated)
        )

    if invalid_last_updated:
        raise ValueError(
            "Docs pages with invalid Last updated date shape: "
            + ", ".join(invalid_last_updated)
        )

    # Optional strict check for changed docs files in CI.
    # Input format accepts comma, whitespace, or newline-separated paths.
    # When set, changed docs pages must refresh Last updated to metadata last_docs_refresh.
    changed_files_raw = os.getenv("DOCS_CHANGED_FILES", "").strip()
    if changed_files_raw:
        changed_files = [
            token for token in re.split(r"[\s,]+", changed_files_raw) if token
        ]
        stale_changed_pages = []
        for token in changed_files:
            page = Path(token)
            if page.suffix != ".md":
                continue
            if page.parts[:2] in [("docs", "adr"), ("docs", "releases")]:
                continue
            if not str(page).startswith("docs/"):
                continue
            if not page.exists():
                continue

            content = page.read_text(encoding="utf-8")
            match = re.search(r"^> Last updated: (.+)$", content, re.MULTILINE)
            if match is None:
                stale_changed_pages.append(f"{page} (missing marker)")
                continue
            if match.group(1).strip() != metadata["last_docs_refresh"]:
                stale_changed_pages.append(
                    f"{page} (expected {metadata['last_docs_refresh']})"
                )

        if stale_changed_pages:
            raise ValueError(
                "Changed docs pages must refresh Last updated to docs/metadata.json last_docs_refresh: "
                + ", ".join(stale_changed_pages)
            )

    print("docs metadata checks passed")


if __name__ == "__main__":
    main()
