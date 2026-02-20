#!/usr/bin/env python3

import json
import re
from pathlib import Path


PINNED_IMAGE_PATTERN = re.compile(r"allisson/secrets:v\d+\.\d+\.\d+")


def main() -> None:
    metadata = json.loads(Path("docs/metadata.json").read_text(encoding="utf-8"))
    current_release = metadata["current_release"]
    current_tag = f"allisson/secrets:{current_release}"

    files_to_check = [
        Path("README.md"),
        Path("docs/getting-started/docker.md"),
        Path("docs/operations/production-rollout.md"),
        Path("docs/cli/commands.md"),
        Path("docs/configuration/environment-variables.md"),
        Path("docs/operations/key-management.md"),
        Path("docs/operations/kms-setup.md"),
    ]

    errors = []

    for file_path in files_to_check:
        if not file_path.exists():
            errors.append(f"missing required docs file: {file_path}")
            continue

        content = file_path.read_text(encoding="utf-8")
        tags = PINNED_IMAGE_PATTERN.findall(content)

        if not tags:
            errors.append(f"{file_path} must include pinned image tag {current_tag}")
            continue

        mismatched = sorted({tag for tag in tags if tag != current_tag})
        if mismatched:
            errors.append(
                f"{file_path} contains non-current pinned tags: "
                + ", ".join(mismatched)
                + f" (expected only {current_tag})"
            )

    if errors:
        raise ValueError(
            "Release image tag consistency check failed: " + " | ".join(errors)
        )

    print("release image tag checks passed")


if __name__ == "__main__":
    main()
