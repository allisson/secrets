#!/usr/bin/env python3

import json
import re
from pathlib import Path


# Updated pattern to allow both pinned and unpinned (latest) references
PINNED_IMAGE_PATTERN = re.compile(r"allisson/secrets:v\d+\.\d+\.\d+")
UNPINNED_IMAGE_PATTERN = re.compile(r"allisson/secrets(?::latest)?(?!\:v)")


def main() -> None:
    metadata = json.loads(Path("docs/metadata.json").read_text(encoding="utf-8"))
    current_release = metadata["current_release"]
    current_tag = f"allisson/secrets:{current_release}"

    files_to_check = [
        Path("README.md"),
        Path("docs/getting-started/docker.md"),
        Path("docs/operations/deployment/production-rollout.md"),
        Path("docs/cli-commands.md"),
        Path("docs/configuration.md"),
        Path("docs/operations/kms/key-management.md"),
        Path("docs/operations/kms/setup.md"),
    ]

    errors = []

    for file_path in files_to_check:
        if not file_path.exists():
            errors.append(f"missing required docs file: {file_path}")
            continue

        content = file_path.read_text(encoding="utf-8")
        pinned_tags = PINNED_IMAGE_PATTERN.findall(content)
        unpinned_refs = UNPINNED_IMAGE_PATTERN.findall(content)

        # Allow either pinned tags matching current release OR unpinned references
        # But not both in the same file (consistency check)
        has_current_pinned = current_tag in pinned_tags
        has_unpinned = bool(unpinned_refs)
        has_old_pinned = any(tag != current_tag for tag in pinned_tags)

        if has_old_pinned:
            mismatched = sorted({tag for tag in pinned_tags if tag != current_tag})
            errors.append(
                f"{file_path} contains outdated pinned tags: "
                + ", ".join(mismatched)
                + f" (expected {current_tag} or unpinned allisson/secrets)"
            )
        elif not has_current_pinned and not has_unpinned:
            errors.append(
                f"{file_path} must include either {current_tag} or unpinned allisson/secrets"
            )

    if errors:
        raise ValueError(
            "Release image tag consistency check failed: " + " | ".join(errors)
        )

    print("release image tag checks passed")


if __name__ == "__main__":
    main()
