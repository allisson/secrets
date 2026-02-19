#!/usr/bin/env python3

import os
import re
import subprocess
from pathlib import Path


RELEASE_RE = re.compile(r"^docs/releases/(v\d+\.\d+\.\d+)\.md$")


def run(cmd: list[str]) -> str:
    out = subprocess.check_output(cmd, text=True)
    return out.strip()


def changed_added_release_notes(base_sha: str, head_sha: str) -> list[str]:
    output = run(["git", "diff", "--name-status", base_sha, head_sha])
    versions: list[str] = []
    if not output:
        return versions

    for line in output.splitlines():
        parts = line.split("\t", 1)
        if len(parts) != 2:
            continue
        status, path = parts
        if status != "A":
            continue
        match = RELEASE_RE.match(path)
        if not match:
            continue
        versions.append(match.group(1))
    return versions


def require_contains(path: Path, needle: str) -> None:
    content = path.read_text(encoding="utf-8")
    if needle not in content:
        raise ValueError(f"{path} missing required link/text: {needle}")


def validate_release(version: str) -> None:
    release_path = Path(f"docs/releases/{version}.md")
    upgrade_path = Path(f"docs/releases/{version}-upgrade.md")
    compatibility_path = Path("docs/releases/compatibility-matrix.md")

    if not release_path.exists():
        raise ValueError(f"Missing release notes file: {release_path}")
    if not upgrade_path.exists():
        raise ValueError(f"Missing upgrade guide for new release notes: {upgrade_path}")

    require_contains(release_path, f"{version}-upgrade.md")
    require_contains(release_path, "compatibility-matrix.md")
    require_contains(compatibility_path, version)

    # Ensure entry-point navigation includes both links for this release.
    require_contains(Path("docs/README.md"), f"releases/{version}.md")
    require_contains(Path("docs/README.md"), f"releases/{version}-upgrade.md")
    require_contains(
        Path("docs/operations/runbook-index.md"), f"../releases/{version}.md"
    )
    require_contains(
        Path("docs/operations/runbook-index.md"),
        f"../releases/{version}-upgrade.md",
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

    versions = changed_added_release_notes(base_sha, head_sha)
    if not versions:
        print("release docs guard passed (no new release note files)")
        return

    for version in versions:
        validate_release(version)

    print("release docs guard passed")


if __name__ == "__main__":
    main()
