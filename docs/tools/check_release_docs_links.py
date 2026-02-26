#!/usr/bin/env python3
"""
Validates that new releases are properly added to consolidated RELEASES.md.

This script checks:
1. New release entries are added to RELEASES.md (not individual files)
2. Navigation files link to RELEASES.md
"""

import json
import os
import re
import subprocess
from pathlib import Path


# Detect changes to RELEASES.md (modified or new release sections)
RELEASES_FILE = Path("docs/releases/RELEASES.md")


def run(cmd: list[str]) -> str:
    out = subprocess.check_output(cmd, text=True)
    return out.strip()


def get_modified_files(base_sha: str, head_sha: str) -> set[str]:
    """Return set of files that were modified in this PR."""
    output = run(["git", "diff", "--name-only", base_sha, head_sha])
    if not output:
        return set()
    return set(output.splitlines())


def extract_version_headers(content: str) -> list[str]:
    """Extract all version headers from RELEASES.md content."""
    # Match: ## [0.7.0] - 2026-02-20
    pattern = re.compile(r"^## \[(\d+\.\d+\.\d+)\] - \d{4}-\d{2}-\d{2}$", re.MULTILINE)
    return pattern.findall(content)


def get_releases_diff(base_sha: str, head_sha: str) -> tuple[list[str], bool]:
    """Return list of new version entries and whether this is a consolidation.

    Returns:
        tuple: (list of versions to validate, is_consolidation flag)

    During consolidation migrations (when RELEASES.md is newly created with many
    versions), only the current release from metadata.json is validated to avoid
    requiring historical versions in the compatibility matrix.
    """
    try:
        # Get old version of RELEASES.md
        old_content = run(["git", "show", f"{base_sha}:docs/releases/RELEASES.md"])
        old_versions = set(extract_version_headers(old_content))
        is_consolidation = False
    except subprocess.CalledProcessError:
        # File might not exist in base (first time / consolidation)
        old_versions = set()
        is_consolidation = True

    # Get new version of RELEASES.md
    new_content = RELEASES_FILE.read_text(encoding="utf-8")
    new_versions = set(extract_version_headers(new_content))

    # Find newly added versions
    added = new_versions - old_versions

    # If this looks like a consolidation (many versions added at once),
    # only validate the current release from metadata
    if is_consolidation and len(added) > 3:
        metadata_path = Path("docs/metadata.json")
        if metadata_path.exists():
            metadata = json.loads(metadata_path.read_text(encoding="utf-8"))
            current = metadata.get("current_release", "").lstrip("v")
            if current in added:
                # Only validate current release during consolidation
                return [current], True
        # Fallback: if current_release not found, validate all
        return sorted(added), True

    # Normal case: validate all new releases
    return sorted(added), False


def require_contains(path: Path, needle: str) -> None:
    """Verify that path contains needle string."""
    content = path.read_text(encoding="utf-8")
    if needle not in content:
        raise ValueError(f"{path} missing required link/text: {needle}")


def validate_release_in_consolidated(version: str) -> None:
    """Validate that new release is properly documented in consolidated files."""
    # Check that version appears in RELEASES.md
    require_contains(RELEASES_FILE, f"[{version}]")

    # Ensure main navigation points to RELEASES.md
    require_contains(Path("docs/README.md"), "releases/RELEASES.md")
    require_contains(
        Path("docs/operations/runbooks/README.md"), "../../releases/RELEASES.md"
    )


def main() -> None:
    if os.getenv("GITHUB_EVENT_NAME", "") != "pull_request":
        print("release docs guard skipped (non-PR)")
        return

    base_sha = os.getenv("PR_BASE_SHA", "").strip()
    head_sha = os.getenv("PR_HEAD_SHA", "").strip()
    if not base_sha or not head_sha:
        raise ValueError(
            "PR_BASE_SHA and PR_HEAD_SHA must be set for release docs guard"
        )

    # Check if RELEASES.md was modified
    modified_files = get_modified_files(base_sha, head_sha)
    if "docs/releases/RELEASES.md" not in modified_files:
        print("release docs guard passed (RELEASES.md not modified)")
        return

    # Get new releases added
    new_versions, is_consolidation = get_releases_diff(base_sha, head_sha)
    if not new_versions:
        print("release docs guard passed (no new release entries detected)")
        return

    if is_consolidation:
        print(
            f"Detected consolidation migration, validating current release only: {', '.join(new_versions)}"
        )
    else:
        print(f"Detected new release(s): {', '.join(new_versions)}")

    # Validate each new release
    for version in new_versions:
        validate_release_in_consolidated(version)
        print(f"  ✓ {version} properly documented")

    print("release docs guard passed")


if __name__ == "__main__":
    main()
